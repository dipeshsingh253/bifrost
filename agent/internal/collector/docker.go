package collector

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dipesh/bifrost/agent/internal/client"
	"github.com/dipesh/bifrost/agent/internal/config"
)

type commandRunner func(name string, args ...string) ([]byte, error)

type DockerCollector struct {
	cfg       config.Config
	run       commandRunner
	lastLogAt map[string]time.Time
}

type dockerPSRow struct {
	ID      string `json:"ID"`
	Image   string `json:"Image"`
	Names   string `json:"Names"`
	Status  string `json:"Status"`
	Command string `json:"Command"`
	Ports   string `json:"Ports"`
	Labels  string `json:"Labels"`
}

type dockerInspectEntry struct {
	ID           string `json:"Id"`
	Name         string `json:"Name"`
	RestartCount int    `json:"RestartCount"`
	Config       struct {
		Image  string            `json:"Image"`
		Labels map[string]string `json:"Labels"`
		Cmd    []string          `json:"Cmd"`
	} `json:"Config"`
	State struct {
		Status    string `json:"Status"`
		StartedAt string `json:"StartedAt"`
		Health    *struct {
			Status string `json:"Status"`
		} `json:"Health"`
	} `json:"State"`
}

type dockerStatsRow struct {
	ID       string `json:"ID"`
	Name     string `json:"Name"`
	CPUPerc  string `json:"CPUPerc"`
	MemUsage string `json:"MemUsage"`
	NetIO    string `json:"NetIO"`
}

type dockerRuntime struct {
	id           string
	name         string
	image        string
	command      string
	ports        []string
	project      string
	status       string
	health       string
	cpuUsagePct  float64
	memoryMB     float64
	networkMB    float64
	restartCount int
	uptime       string
	lastSeenAt   time.Time
}

func NewDockerCollector(cfg config.Config) *DockerCollector {
	return &DockerCollector{
		cfg:       cfg,
		run:       runDockerCommand,
		lastLogAt: map[string]time.Time{},
	}
}

func (c *DockerCollector) Collect() ([]client.ServiceSnapshot, []client.LogPayload) {
	if !c.cfg.Collectors.Docker && !c.cfg.Collectors.Logs {
		return nil, nil
	}

	rows, err := c.listContainers()
	if err != nil || len(rows) == 0 {
		return nil, nil
	}

	allowedRows := c.filterRows(rows)
	if len(allowedRows) == 0 {
		return nil, nil
	}

	ids := containerIDs(allowedRows)
	inspects := c.inspectContainers(ids)
	stats := c.containerStats()

	runtimes := make([]dockerRuntime, 0, len(allowedRows))
	for _, row := range allowedRows {
		runtime := buildDockerRuntime(row, inspects[row.ID], stats[row.ID])
		runtimes = append(runtimes, runtime)
	}

	services := buildServiceSnapshots(runtimes)
	logs := c.collectLogs(services)
	return services, logs
}

func (c *DockerCollector) listContainers() ([]dockerPSRow, error) {
	output, err := c.run("docker", "ps", "-a", "--no-trunc", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	rows := make([]dockerPSRow, 0)
	for scanner.Scan() {
		var row dockerPSRow
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			continue
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func (c *DockerCollector) filterRows(rows []dockerPSRow) []dockerPSRow {
	filtered := make([]dockerPSRow, 0, len(rows))
	for _, row := range rows {
		name := sanitizeContainerName(row.Names)
		project := composeProject(parseLabelMap(row.Labels), name)
		if !includedDockerRuntime(project, name, c.cfg.Docker) {
			continue
		}
		if excludedDockerRuntime(project, name, c.cfg.Docker) {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}

func containerIDs(rows []dockerPSRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func (c *DockerCollector) inspectContainers(ids []string) map[string]dockerInspectEntry {
	if len(ids) == 0 {
		return map[string]dockerInspectEntry{}
	}

	args := append([]string{"inspect"}, ids...)
	output, err := c.run("docker", args...)
	if err != nil {
		return map[string]dockerInspectEntry{}
	}

	var entries []dockerInspectEntry
	if err := json.Unmarshal(output, &entries); err != nil {
		return map[string]dockerInspectEntry{}
	}

	result := make(map[string]dockerInspectEntry, len(entries))
	for _, entry := range entries {
		result[entry.ID] = entry
	}
	return result
}

func (c *DockerCollector) containerStats() map[string]dockerStatsRow {
	if !c.cfg.Collectors.Docker {
		return map[string]dockerStatsRow{}
	}

	output, err := c.run("docker", "stats", "--no-stream", "--no-trunc", "--format", "{{json .}}")
	if err != nil {
		return map[string]dockerStatsRow{}
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	stats := make(map[string]dockerStatsRow)
	for scanner.Scan() {
		var row dockerStatsRow
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			continue
		}
		stats[row.ID] = row
	}
	return stats
}

func buildDockerRuntime(row dockerPSRow, inspect dockerInspectEntry, stats dockerStatsRow) dockerRuntime {
	labels := parseLabelMap(row.Labels)
	if len(inspect.Config.Labels) > 0 {
		labels = inspect.Config.Labels
	}

	name := sanitizeContainerName(row.Names)
	if inspect.Name != "" {
		name = sanitizeContainerName(inspect.Name)
	}

	image := row.Image
	if inspect.Config.Image != "" {
		image = inspect.Config.Image
	}

	command := row.Command
	if command == "" && len(inspect.Config.Cmd) > 0 {
		command = strings.Join(inspect.Config.Cmd, " ")
	}

	project := composeProject(labels, name)
	status := normalizeDockerStatus(row.Status, inspect.State.Status)
	health := normalizeDockerHealth(inspect)
	startedAt := parseDockerTime(inspect.State.StartedAt)

	return dockerRuntime{
		id:           row.ID,
		name:         name,
		image:        image,
		command:      command,
		ports:        splitPorts(row.Ports),
		project:      project,
		status:       status,
		health:       health,
		cpuUsagePct:  parsePercent(stats.CPUPerc),
		memoryMB:     parseMemoryUsage(stats.MemUsage),
		networkMB:    parseNetIO(stats.NetIO),
		restartCount: inspect.RestartCount,
		uptime:       formatUptime(startedAt, time.Now().UTC(), status),
		lastSeenAt:   time.Now().UTC(),
	}
}

func buildServiceSnapshots(runtimes []dockerRuntime) []client.ServiceSnapshot {
	servicesByID := map[string]*client.ServiceSnapshot{}
	for _, runtime := range runtimes {
		serviceID, serviceName, composeProjectName := serviceIdentity(runtime)
		service, ok := servicesByID[serviceID]
		if !ok {
			service = &client.ServiceSnapshot{
				ID:             serviceID,
				Name:           serviceName,
				ComposeProject: composeProjectName,
				Status:         "running",
			}
			servicesByID[serviceID] = service
		}

		service.Containers = append(service.Containers, client.ContainerSnapshot{
			ID:           runtime.id,
			Name:         runtime.name,
			Image:        runtime.image,
			Status:       runtime.status,
			Health:       runtime.health,
			CPUUsagePct:  runtime.cpuUsagePct,
			MemoryMB:     runtime.memoryMB,
			NetworkMB:    runtime.networkMB,
			RestartCount: runtime.restartCount,
			Uptime:       runtime.uptime,
			Ports:        runtime.ports,
			Command:      runtime.command,
			LastSeenAt:   runtime.lastSeenAt,
		})
		service.PublishedPorts = append(service.PublishedPorts, runtime.ports...)
		service.Status = rollupDockerServiceStatus(service.Status, runtime.status, runtime.health)
	}

	services := make([]client.ServiceSnapshot, 0, len(servicesByID))
	for _, service := range servicesByID {
		service.PublishedPorts = uniqueStrings(service.PublishedPorts)
		sort.Slice(service.Containers, func(i, j int) bool {
			return service.Containers[i].Name < service.Containers[j].Name
		})
		services = append(services, *service)
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})
	return services
}

func (c *DockerCollector) collectLogs(services []client.ServiceSnapshot) []client.LogPayload {
	if !c.cfg.Collectors.Logs {
		return nil
	}

	logs := make([]client.LogPayload, 0)
	for _, service := range services {
		for _, container := range service.Containers {
			lines := c.collectContainerLogs(service, container)
			logs = append(logs, lines...)
		}
	}

	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp.Before(logs[j].Timestamp)
	})
	return logs
}

func (c *DockerCollector) collectContainerLogs(service client.ServiceSnapshot, container client.ContainerSnapshot) []client.LogPayload {
	args := []string{"logs", "--timestamps"}
	if last, ok := c.lastLogAt[container.ID]; ok && !last.IsZero() {
		args = append(args, "--since", last.Add(time.Nanosecond).Format(time.RFC3339Nano))
	} else {
		args = append(args, "--tail", strconv.Itoa(c.cfg.Logs.MaxLinesPerFetch))
	}
	args = append(args, container.ID)

	output, err := c.run("docker", args...)
	if err != nil {
		return nil
	}

	lines, latest := parseDockerLogOutput(c.cfg.ServerID, service.ID, container.ID, output)
	if !latest.IsZero() {
		c.lastLogAt[container.ID] = latest
	}
	return lines
}

func parseDockerLogOutput(serverID, serviceID, containerID string, output []byte) ([]client.LogPayload, time.Time) {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	logs := make([]client.LogPayload, 0)
	var latest time.Time

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}

		timestamp, err := time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			continue
		}
		if timestamp.After(latest) {
			latest = timestamp
		}

		message := parts[1]
		logs = append(logs, client.LogPayload{
			ServerID:    serverID,
			ServiceID:   serviceID,
			ContainerID: containerID,
			Level:       inferLogLevel(message),
			Message:     message,
			Timestamp:   timestamp.UTC(),
		})
	}

	return logs, latest
}

func runDockerCommand(name string, args ...string) ([]byte, error) {
	command := exec.Command(name, args...)
	return command.CombinedOutput()
}

func parseLabelMap(raw string) map[string]string {
	if strings.TrimSpace(raw) == "" {
		return map[string]string{}
	}

	labels := map[string]string{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		segments := strings.SplitN(part, "=", 2)
		if len(segments) != 2 {
			continue
		}
		labels[segments[0]] = segments[1]
	}

	return labels
}

func composeProject(labels map[string]string, name string) string {
	if project, ok := labels["com.docker.compose.project"]; ok && project != "" {
		return project
	}
	return ""
}

func serviceIdentity(runtime dockerRuntime) (string, string, string) {
	if runtime.project != "" {
		return "svc-" + sanitize(runtime.project), runtime.project, runtime.project
	}
	return "svc-" + sanitize(runtime.name), runtime.name, ""
}

func includedDockerRuntime(project, name string, docker config.DockerConfig) bool {
	if docker.IncludeAll {
		return true
	}

	return stringInList(project, docker.IncludeProjects) || stringInList(name, docker.IncludeContainers)
}

func excludedDockerRuntime(project, name string, docker config.DockerConfig) bool {
	return stringInList(project, docker.ExcludeProjects) || stringInList(name, docker.ExcludeContainers)
}

func stringInList(value string, list []string) bool {
	for _, candidate := range list {
		if value == strings.TrimSpace(candidate) {
			return true
		}
	}

	return false
}

func splitPorts(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	ports := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			ports = append(ports, part)
		}
	}
	return ports
}

func sanitize(value string) string {
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, "/", "-")
	return value
}

func sanitizeContainerName(name string) string {
	return strings.TrimPrefix(strings.TrimSpace(name), "/")
}

func normalizeDockerStatus(psStatus, inspectStatus string) string {
	switch {
	case inspectStatus == "running":
		return "running"
	case inspectStatus == "exited" || inspectStatus == "dead" || inspectStatus == "stopped":
		return "exited"
	case inspectStatus != "":
		return inspectStatus
	case strings.HasPrefix(strings.ToLower(psStatus), "up"):
		return "running"
	default:
		return "exited"
	}
}

func normalizeDockerHealth(inspect dockerInspectEntry) string {
	if inspect.State.Health == nil || inspect.State.Health.Status == "" {
		if inspect.State.Status == "running" {
			return "unknown"
		}
		return "unknown"
	}

	return inspect.State.Health.Status
}

func rollupDockerServiceStatus(current, containerStatus, health string) string {
	if containerStatus != "running" {
		return "stopped"
	}
	if health == "unhealthy" && current == "running" {
		return "degraded"
	}
	if current == "" {
		return "running"
	}
	return current
}

func parseDockerTime(value string) time.Time {
	if value == "" || value == "0001-01-01T00:00:00Z" {
		return time.Time{}
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func formatUptime(startedAt, now time.Time, status string) string {
	if startedAt.IsZero() || status != "running" {
		return "-"
	}

	duration := now.Sub(startedAt)
	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes < 1 {
			minutes = 1
		}
		return strconv.Itoa(minutes) + "m"
	}

	days := int(duration / (24 * time.Hour))
	hours := int(duration/time.Hour) % 24
	if days > 0 {
		return strconv.Itoa(days) + "d " + pad2(hours) + "h"
	}
	return strconv.Itoa(int(duration/time.Hour)) + "h"
}

func pad2(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}

func parsePercent(value string) float64 {
	value = strings.TrimSpace(strings.TrimSuffix(value, "%"))
	if value == "" || value == "--" {
		return 0
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func parseMemoryUsage(value string) float64 {
	parts := strings.Split(value, "/")
	if len(parts) == 0 {
		return 0
	}
	return parseByteValue(parts[0]) / (1024 * 1024)
}

func parseNetIO(value string) float64 {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return 0
	}
	return (parseByteValue(parts[0]) + parseByteValue(parts[1])) / (1024 * 1024)
}

func parseByteValue(value string) float64 {
	clean := strings.TrimSpace(strings.ReplaceAll(value, "iB", "B"))
	if clean == "" || clean == "--" {
		return 0
	}

	unitMultipliers := []struct {
		suffix     string
		multiplier float64
	}{
		{suffix: "TB", multiplier: 1024 * 1024 * 1024 * 1024},
		{suffix: "GB", multiplier: 1024 * 1024 * 1024},
		{suffix: "MB", multiplier: 1024 * 1024},
		{suffix: "kB", multiplier: 1024},
		{suffix: "B", multiplier: 1},
	}

	for _, unit := range unitMultipliers {
		if strings.HasSuffix(clean, unit.suffix) {
			number := strings.TrimSpace(strings.TrimSuffix(clean, unit.suffix))
			parsed, err := strconv.ParseFloat(number, 64)
			if err != nil {
				return 0
			}
			return parsed * unit.multiplier
		}
	}

	parsed, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func inferLogLevel(message string) string {
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "error"), strings.Contains(lower, "fatal"), strings.Contains(lower, "panic"):
		return "error"
	case strings.Contains(lower, "warn"):
		return "warn"
	case strings.Contains(lower, "debug"):
		return "debug"
	default:
		return "info"
	}
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}
