package store

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dipesh/bifrost/backend/internal/domain"
)

var ErrNotFound = errors.New("not found")

type SeedData struct {
	Users            []domain.User
	Servers          []domain.Server
	Services         []domain.Service
	Metrics          map[string][]domain.MetricSeries
	ContainerMetrics map[string]domain.ContainerMetricBundle
	Logs             map[string][]domain.LogLine
	Agents           []domain.Agent
}

type MemoryStore struct {
	mu               sync.RWMutex
	users            map[string]domain.User
	byToken          map[string]domain.User
	byAPIKey         map[string]domain.Agent
	servers          map[string]domain.Server
	services         map[string]domain.Service
	metrics          map[string][]domain.MetricSeries
	containerMetrics map[string]domain.ContainerMetricBundle
	logs             map[string][]domain.LogLine
	agents           map[string]domain.Agent
}

func NewMemoryStore(seed SeedData) *MemoryStore {
	store := &MemoryStore{
		users:            map[string]domain.User{},
		byToken:          map[string]domain.User{},
		byAPIKey:         map[string]domain.Agent{},
		servers:          map[string]domain.Server{},
		services:         map[string]domain.Service{},
		metrics:          map[string][]domain.MetricSeries{},
		containerMetrics: map[string]domain.ContainerMetricBundle{},
		logs:             map[string][]domain.LogLine{},
		agents:           map[string]domain.Agent{},
	}

	for _, user := range seed.Users {
		store.users[user.Email] = user
		store.byToken[user.AuthToken] = user
	}

	for _, server := range seed.Servers {
		store.servers[server.ID] = server
	}

	for _, service := range seed.Services {
		store.services[service.ID] = service
	}

	for key, series := range seed.Metrics {
		store.metrics[key] = series
	}

	for key, series := range seed.ContainerMetrics {
		store.containerMetrics[key] = series
	}

	for key, lines := range seed.Logs {
		store.logs[key] = lines
	}

	for _, agent := range seed.Agents {
		store.agents[agent.ID] = agent
		store.byAPIKey[agent.APIKey] = agent
	}

	return store
}

func (s *MemoryStore) Authenticate(email, password string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[email]
	if !ok || user.Password != password {
		return domain.User{}, ErrNotFound
	}

	return user, nil
}

func (s *MemoryStore) UserByToken(token string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.byToken[token]
	if !ok {
		return domain.User{}, ErrNotFound
	}

	return user, nil
}

func (s *MemoryStore) ListServers(tenantID string) []domain.Server {
	s.mu.RLock()
	defer s.mu.RUnlock()

	servers := make([]domain.Server, 0)
	for _, server := range s.servers {
		if server.TenantID == tenantID {
			servers = append(servers, server)
		}
	}

	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})

	return servers
}

func (s *MemoryStore) ServerByID(tenantID, serverID string) (domain.Server, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	server, ok := s.servers[serverID]
	if !ok || server.TenantID != tenantID {
		return domain.Server{}, ErrNotFound
	}

	return server, nil
}

func (s *MemoryStore) ServicesByServer(tenantID, serverID string) []domain.Service {
	s.mu.RLock()
	defer s.mu.RUnlock()

	services := make([]domain.Service, 0)
	for _, service := range s.services {
		if service.TenantID == tenantID && service.ServerID == serverID {
			services = append(services, service)
		}
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services
}

func (s *MemoryStore) ServiceByID(tenantID, serviceID string) (domain.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	service, ok := s.services[serviceID]
	if !ok || service.TenantID != tenantID {
		return domain.Service{}, ErrNotFound
	}

	return service, nil
}

func (s *MemoryStore) ProjectByID(tenantID, serverID, projectID string) (domain.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	service, ok := s.services[projectID]
	if !ok || service.TenantID != tenantID || service.ServerID != serverID || service.ComposeProject == "" {
		return domain.Service{}, ErrNotFound
	}

	return cloneService(service), nil
}

func (s *MemoryStore) ProjectsByServer(tenantID, serverID string) []domain.Service {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projects := make([]domain.Service, 0)
	for _, service := range s.services {
		if service.TenantID == tenantID && service.ServerID == serverID && service.ComposeProject != "" {
			projects = append(projects, cloneService(service))
		}
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects
}

func (s *MemoryStore) StandaloneContainersByServer(tenantID, serverID string) []domain.Container {
	s.mu.RLock()
	defer s.mu.RUnlock()

	containers := make([]domain.Container, 0)
	for _, service := range s.services {
		if service.TenantID != tenantID || service.ServerID != serverID || service.ComposeProject != "" {
			continue
		}

		for _, container := range service.Containers {
			containerCopy := container
			containerCopy.Ports = append([]string(nil), container.Ports...)
			containers = append(containers, containerCopy)
		}
	}

	sort.Slice(containers, func(i, j int) bool {
		return containers[i].Name < containers[j].Name
	})

	return containers
}

func (s *MemoryStore) MetricsByServer(serverID string) []domain.MetricSeries {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return cloneMetricSeries(s.metrics[serverID])
}

func (s *MemoryStore) ContainerMetricsByServer(serverID string) domain.ContainerMetricBundle {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return cloneContainerMetricBundle(s.containerMetrics[serverID])
}

func (s *MemoryStore) LogsByService(serviceID string) []domain.LogLine {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lines := append([]domain.LogLine(nil), s.logs[serviceID]...)
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].Timestamp.Before(lines[j].Timestamp)
	})
	return lines
}

func (s *MemoryStore) LogsByContainer(serviceID, containerID string) []domain.LogLine {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lines := make([]domain.LogLine, 0)
	for _, line := range s.logs[serviceID] {
		if line.ContainerID == containerID {
			lines = append(lines, line)
		}
	}

	sort.Slice(lines, func(i, j int) bool {
		return lines[i].Timestamp.Before(lines[j].Timestamp)
	})
	return lines
}

func (s *MemoryStore) ServerBundle(tenantID, serverID string) (domain.ServerBundle, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	server, ok := s.servers[serverID]
	if !ok || server.TenantID != tenantID {
		return domain.ServerBundle{}, ErrNotFound
	}

	services := make([]domain.Service, 0)
	for _, service := range s.services {
		if service.TenantID == tenantID && service.ServerID == serverID {
			services = append(services, cloneService(service))
		}
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	containerNames := currentContainerNames(services)

	return domain.ServerBundle{
		Server:           server,
		Services:         services,
		Metrics:          cloneMetricSeries(s.metrics[serverID]),
		ContainerMetrics: filterContainerMetricBundleByNames(cloneContainerMetricBundle(s.containerMetrics[serverID]), containerNames),
	}, nil
}

func (s *MemoryStore) ContainerByID(tenantID, serverID, containerID string) (domain.Container, domain.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, service := range s.services {
		if service.TenantID != tenantID || service.ServerID != serverID {
			continue
		}

		for _, container := range service.Containers {
			if container.ID == containerID {
				containerCopy := container
				containerCopy.Ports = append([]string(nil), container.Ports...)
				return containerCopy, cloneService(service), nil
			}
		}
	}

	return domain.Container{}, domain.Service{}, ErrNotFound
}

func (s *MemoryStore) ProjectMetrics(tenantID, serverID, projectID string) (domain.ContainerMetricBundle, error) {
	project, err := s.ProjectByID(tenantID, serverID, projectID)
	if err != nil {
		return domain.ContainerMetricBundle{}, err
	}

	names := make([]string, 0, len(project.Containers))
	for _, container := range project.Containers {
		names = append(names, container.Name)
	}

	return filterContainerMetricBundleByNames(s.ContainerMetricsByServer(serverID), names), nil
}

func (s *MemoryStore) ContainerMetrics(tenantID, serverID, containerID string) (domain.ContainerMetricHistory, domain.Container, domain.Service, error) {
	container, service, err := s.ContainerByID(tenantID, serverID, containerID)
	if err != nil {
		return domain.ContainerMetricHistory{}, domain.Container{}, domain.Service{}, err
	}

	bundle := s.ContainerMetricsByServer(serverID)
	return containerMetricHistoryByName(bundle, container.Name), container, service, nil
}

func (s *MemoryStore) ProjectEvents(tenantID, serverID, projectID string) ([]domain.EventLog, domain.Service, error) {
	project, err := s.ProjectByID(tenantID, serverID, projectID)
	if err != nil {
		return nil, domain.Service{}, err
	}

	return deriveProjectEvents(project, s.LogsByService(project.ID)), project, nil
}

func (s *MemoryStore) ContainerEvents(tenantID, serverID, containerID string) ([]domain.EventLog, domain.Container, domain.Service, error) {
	container, service, err := s.ContainerByID(tenantID, serverID, containerID)
	if err != nil {
		return nil, domain.Container{}, domain.Service{}, err
	}

	return deriveContainerEvents(container, service, s.LogsByContainer(service.ID, container.ID)), container, service, nil
}

func (s *MemoryStore) ContainerEnv(tenantID, serverID, containerID string) (map[string]string, domain.Container, domain.Service, error) {
	container, service, err := s.ContainerByID(tenantID, serverID, containerID)
	if err != nil {
		return nil, domain.Container{}, domain.Service{}, err
	}

	return deriveContainerEnv(container, service), container, service, nil
}

func (s *MemoryStore) EnrollAgent(agent domain.Agent) domain.Agent {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	agent.EnrolledAt = now
	agent.LastSeenAt = now
	s.agents[agent.ID] = agent
	s.byAPIKey[agent.APIKey] = agent
	return agent
}

func (s *MemoryStore) AgentByAPIKey(apiKey string) (domain.Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, ok := s.byAPIKey[apiKey]
	if !ok {
		return domain.Agent{}, ErrNotFound
	}

	return agent, nil
}

func (s *MemoryStore) UpdateAgentLastSeen(agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	agent, ok := s.agents[agentID]
	if !ok {
		return ErrNotFound
	}

	agent.LastSeenAt = time.Now().UTC()
	s.agents[agentID] = agent
	return nil
}

func (s *MemoryStore) Ingest(payload domain.IngestPayload) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	server, ok := s.servers[payload.Server.ID]
	if !ok {
		server = domain.Server{
			ID:       payload.Server.ID,
			TenantID: "tenant-demo",
		}
	}

	server.Name = payload.Server.Name
	server.Hostname = payload.Server.Hostname
	server.PublicIP = payload.Server.PublicIP
	server.AgentVersion = payload.Server.AgentVersion
	server.Status = normalizeServerStatus(payload.Server.Status)
	server.LastSeenAt = payload.Server.CollectedAt
	server.UptimeSeconds = payload.Server.UptimeSeconds
	server.CPUUsagePct = payload.Server.CPUUsagePct
	server.MemoryUsagePct = payload.Server.MemoryUsagePct
	server.DiskUsagePct = payload.Server.DiskUsagePct
	server.NetworkRXMB = payload.Server.NetworkRXMB
	server.NetworkTXMB = payload.Server.NetworkTXMB
	server.LoadAverage = payload.Server.LoadAverage
	server.OS = payload.Server.OS
	server.Kernel = payload.Server.Kernel
	server.CPUModel = payload.Server.CPUModel
	server.CPUCores = payload.Server.CPUCores
	server.CPUThreads = payload.Server.CPUThreads
	server.TotalMemoryGB = payload.Server.TotalMemoryGB
	server.TotalDiskGB = payload.Server.TotalDiskGB
	s.servers[server.ID] = server

	incomingServiceIDs := make(map[string]struct{}, len(payload.Server.Services))

	for _, snapshot := range payload.Server.Services {
		incomingServiceIDs[snapshot.ID] = struct{}{}
		service := domain.Service{
			ID:             snapshot.ID,
			TenantID:       "tenant-demo",
			ServerID:       server.ID,
			Name:           snapshot.Name,
			ComposeProject: snapshot.ComposeProject,
			Status:         snapshot.Status,
			ContainerCount: len(snapshot.Containers),
			PublishedPorts: snapshot.PublishedPorts,
			Containers:     make([]domain.Container, 0, len(snapshot.Containers)),
		}

		restarts := 0
		serviceStatus := normalizeServiceStatus(snapshot.Status)
		lastSeenAt := payload.Server.CollectedAt
		for _, containerSnapshot := range snapshot.Containers {
			restarts += containerSnapshot.RestartCount
			container := domain.Container{
				ID:           containerSnapshot.ID,
				ServiceID:    snapshot.ID,
				Name:         containerSnapshot.Name,
				Image:        containerSnapshot.Image,
				Status:       normalizeContainerStatus(containerSnapshot.Status),
				Health:       normalizeHealth(containerSnapshot.Health),
				CPUUsagePct:  containerSnapshot.CPUUsagePct,
				MemoryMB:     containerSnapshot.MemoryMB,
				NetworkMB:    containerSnapshot.NetworkMB,
				RestartCount: containerSnapshot.RestartCount,
				Uptime:       containerSnapshot.Uptime,
				Ports:        containerSnapshot.Ports,
				Command:      containerSnapshot.Command,
				LastSeenAt:   containerSnapshot.LastSeenAt,
			}
			if container.LastSeenAt.IsZero() {
				container.LastSeenAt = lastSeenAt
			}
			service.Containers = append(service.Containers, container)
			serviceStatus = rollupServiceStatus(serviceStatus, container.Status, container.Health)
		}

		service.RestartCount = restarts
		service.Status = serviceStatus
		s.services[service.ID] = service
	}

	s.pruneServicesForServer(server.ID, incomingServiceIDs)

	for _, metric := range payload.Metrics {
		s.metrics[metric.ServerID] = mergeMetricSeries(s.metrics[metric.ServerID], domain.MetricSeries{
			Key:    metric.Key,
			Unit:   metric.Unit,
			Points: metric.Points,
		})
	}

	s.containerMetrics[server.ID] = appendContainerMetrics(s.containerMetrics[server.ID], payload.Server.CollectedAt, payload.Server.Services)

	for _, logLine := range payload.Logs {
		containerName, serviceTag := lookupLogContext(payload.Server.Services, logLine.ServiceID, logLine.ContainerID)
		line := domain.LogLine{
			ID:            logLine.ContainerID + "-" + logLine.Timestamp.Format(time.RFC3339Nano),
			ServerID:      logLine.ServerID,
			ServiceID:     logLine.ServiceID,
			ContainerID:   logLine.ContainerID,
			ContainerName: containerName,
			ServiceTag:    serviceTag,
			Level:         logLine.Level,
			Message:       logLine.Message,
			Timestamp:     logLine.Timestamp,
		}
		s.logs[line.ServiceID] = append(s.logs[line.ServiceID], line)
		if service, ok := s.services[line.ServiceID]; ok && (service.LastLogTimestamp.IsZero() || line.Timestamp.After(service.LastLogTimestamp)) {
			service.LastLogTimestamp = line.Timestamp
			s.services[line.ServiceID] = service
		}
	}

	return nil
}

func mergeMetricSeries(existing []domain.MetricSeries, incoming domain.MetricSeries) []domain.MetricSeries {
	for index := range existing {
		if existing[index].Key == incoming.Key {
			existing[index].Points = append(existing[index].Points, incoming.Points...)
			if len(existing[index].Points) > 30 {
				existing[index].Points = existing[index].Points[len(existing[index].Points)-30:]
			}
			return existing
		}
	}

	return append(existing, incoming)
}

func cloneService(service domain.Service) domain.Service {
	copyService := service
	copyService.PublishedPorts = append([]string(nil), service.PublishedPorts...)
	copyService.Containers = make([]domain.Container, 0, len(service.Containers))
	for _, container := range service.Containers {
		containerCopy := container
		containerCopy.Ports = append([]string(nil), container.Ports...)
		copyService.Containers = append(copyService.Containers, containerCopy)
	}
	return copyService
}

func cloneMetricSeries(series []domain.MetricSeries) []domain.MetricSeries {
	cloned := make([]domain.MetricSeries, 0, len(series))
	for _, item := range series {
		cloned = append(cloned, domain.MetricSeries{
			Key:    item.Key,
			Unit:   item.Unit,
			Points: append([]domain.MetricPoint(nil), item.Points...),
		})
	}
	return cloned
}

func cloneContainerMetricPoints(points []domain.ContainerMetricPoint) []domain.ContainerMetricPoint {
	cloned := make([]domain.ContainerMetricPoint, 0, len(points))
	for _, point := range points {
		values := make(map[string]float64, len(point.Values))
		for key, value := range point.Values {
			values[key] = value
		}
		cloned = append(cloned, domain.ContainerMetricPoint{
			Timestamp: point.Timestamp,
			Values:    values,
		})
	}
	return cloned
}

func cloneContainerMetricBundle(bundle domain.ContainerMetricBundle) domain.ContainerMetricBundle {
	return domain.ContainerMetricBundle{
		CPU:     cloneContainerMetricPoints(bundle.CPU),
		Memory:  cloneContainerMetricPoints(bundle.Memory),
		Network: cloneContainerMetricPoints(bundle.Network),
	}
}

func appendContainerMetrics(existing domain.ContainerMetricBundle, timestamp time.Time, services []domain.ServiceSnapshot) domain.ContainerMetricBundle {
	cpuValues := map[string]float64{}
	memoryValues := map[string]float64{}
	networkValues := map[string]float64{}

	for _, service := range services {
		for _, container := range service.Containers {
			cpuValues[container.Name] = container.CPUUsagePct
			memoryValues[container.Name] = container.MemoryMB
			networkValues[container.Name] = container.NetworkMB
		}
	}

	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	existing.CPU = appendMetricPoint(existing.CPU, timestamp, cpuValues)
	existing.Memory = appendMetricPoint(existing.Memory, timestamp, memoryValues)
	existing.Network = appendMetricPoint(existing.Network, timestamp, networkValues)
	return existing
}

func appendMetricPoint(existing []domain.ContainerMetricPoint, timestamp time.Time, values map[string]float64) []domain.ContainerMetricPoint {
	pointValues := make(map[string]float64, len(values))
	for key, value := range values {
		pointValues[key] = value
	}

	existing = append(existing, domain.ContainerMetricPoint{
		Timestamp: timestamp,
		Values:    pointValues,
	})
	if len(existing) > 30 {
		existing = existing[len(existing)-30:]
	}
	return existing
}

func currentContainerNames(services []domain.Service) []string {
	names := make([]string, 0)
	for _, service := range services {
		for _, container := range service.Containers {
			names = append(names, container.Name)
		}
	}
	return names
}

func (s *MemoryStore) pruneServicesForServer(serverID string, keep map[string]struct{}) {
	for serviceID, service := range s.services {
		if service.ServerID != serverID {
			continue
		}
		if _, ok := keep[serviceID]; ok {
			continue
		}
		delete(s.services, serviceID)
		delete(s.logs, serviceID)
	}
}

func filterContainerMetricBundleByNames(bundle domain.ContainerMetricBundle, names []string) domain.ContainerMetricBundle {
	allowed := map[string]struct{}{}
	for _, name := range names {
		allowed[name] = struct{}{}
	}

	return domain.ContainerMetricBundle{
		CPU:     filterContainerMetricPoints(bundle.CPU, allowed),
		Memory:  filterContainerMetricPoints(bundle.Memory, allowed),
		Network: filterContainerMetricPoints(bundle.Network, allowed),
	}
}

func filterContainerMetricPoints(points []domain.ContainerMetricPoint, allowed map[string]struct{}) []domain.ContainerMetricPoint {
	filtered := make([]domain.ContainerMetricPoint, 0, len(points))
	for _, point := range points {
		values := map[string]float64{}
		for key, value := range point.Values {
			if _, ok := allowed[key]; ok {
				values[key] = value
			}
		}
		filtered = append(filtered, domain.ContainerMetricPoint{
			Timestamp: point.Timestamp,
			Values:    values,
		})
	}
	return filtered
}

func containerMetricHistoryByName(bundle domain.ContainerMetricBundle, name string) domain.ContainerMetricHistory {
	return domain.ContainerMetricHistory{
		CPU:     metricSeriesForContainer(bundle.CPU, name),
		Memory:  metricSeriesForContainer(bundle.Memory, name),
		Network: metricSeriesForContainer(bundle.Network, name),
	}
}

func metricSeriesForContainer(points []domain.ContainerMetricPoint, name string) []domain.MetricPoint {
	series := make([]domain.MetricPoint, 0, len(points))
	for _, point := range points {
		series = append(series, domain.MetricPoint{
			Timestamp: point.Timestamp,
			Value:     point.Values[name],
		})
	}
	return series
}

func deriveProjectEvents(project domain.Service, logs []domain.LogLine) []domain.EventLog {
	events := make([]domain.EventLog, 0)
	for _, container := range project.Containers {
		events = append(events, deriveContainerEvents(container, project, filterLogsByContainer(logs, container.ID))...)
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})
	return limitEvents(events, 20)
}

func deriveContainerEvents(container domain.Container, service domain.Service, logs []domain.LogLine) []domain.EventLog {
	events := make([]domain.EventLog, 0)

	if container.Status == "running" {
		events = append(events, domain.EventLog{
			ID:         fmt.Sprintf("event-start-%s", container.ID),
			Timestamp:  container.LastSeenAt,
			Type:       "start",
			Message:    "Container started",
			EntityName: container.Name,
		})
	} else {
		events = append(events, domain.EventLog{
			ID:         fmt.Sprintf("event-stop-%s", container.ID),
			Timestamp:  container.LastSeenAt,
			Type:       "stop",
			Message:    "Container stopped",
			EntityName: container.Name,
		})
	}

	if container.RestartCount > 0 {
		events = append(events, domain.EventLog{
			ID:         fmt.Sprintf("event-restart-%s", container.ID),
			Timestamp:  container.LastSeenAt.Add(-1 * time.Minute),
			Type:       "restart",
			Message:    fmt.Sprintf("Container restarted %d times", container.RestartCount),
			EntityName: container.Name,
		})
	}

	if container.Health == "unhealthy" {
		events = append(events, domain.EventLog{
			ID:         fmt.Sprintf("event-health-%s", container.ID),
			Timestamp:  container.LastSeenAt.Add(-30 * time.Second),
			Type:       "health_change",
			Message:    "Health status changed to unhealthy",
			EntityName: container.Name,
		})
	}

	for _, line := range logs {
		if strings.EqualFold(line.Level, "error") {
			events = append(events, domain.EventLog{
				ID:         fmt.Sprintf("event-crash-%s-%s", container.ID, line.ID),
				Timestamp:  line.Timestamp,
				Type:       "crash",
				Message:    line.Message,
				EntityName: container.Name,
			})
			break
		}
	}

	if len(events) == 0 && service.Name != "" {
		events = append(events, domain.EventLog{
			ID:         fmt.Sprintf("event-service-%s", container.ID),
			Timestamp:  container.LastSeenAt,
			Type:       "start",
			Message:    "Container observed by backend",
			EntityName: container.Name,
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})
	return limitEvents(events, 10)
}

func deriveContainerEnv(container domain.Container, service domain.Service) map[string]string {
	port := "8080"
	if len(container.Ports) > 0 {
		port = strings.Split(container.Ports[0], ":")[0]
	}

	env := map[string]string{
		"CONTAINER_NAME": container.Name,
		"IMAGE":          container.Image,
		"LOG_LEVEL":      "info",
		"NODE_ENV":       "production",
		"PORT":           port,
		"SERVICE_NAME":   service.Name,
		"TZ":             "UTC",
	}
	if service.ComposeProject != "" {
		env["COMPOSE_PROJECT"] = service.ComposeProject
	}

	return env
}

func filterLogsByContainer(lines []domain.LogLine, containerID string) []domain.LogLine {
	filtered := make([]domain.LogLine, 0)
	for _, line := range lines {
		if line.ContainerID == containerID {
			filtered = append(filtered, line)
		}
	}
	return filtered
}

func FilterLogs(lines []domain.LogLine, search string, limit int) []domain.LogLine {
	filtered := make([]domain.LogLine, 0, len(lines))
	search = strings.TrimSpace(strings.ToLower(search))
	for _, line := range lines {
		if search != "" {
			if !strings.Contains(strings.ToLower(line.Message), search) &&
				!strings.Contains(strings.ToLower(line.ContainerName), search) &&
				!strings.Contains(strings.ToLower(line.ServiceTag), search) {
				continue
			}
		}
		filtered = append(filtered, line)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.Before(filtered[j].Timestamp)
	})

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	return filtered
}

func limitEvents(events []domain.EventLog, limit int) []domain.EventLog {
	if limit > 0 && len(events) > limit {
		return events[:limit]
	}
	return events
}

func normalizeServerStatus(status string) string {
	switch status {
	case "healthy", "running", "up":
		return "up"
	case "down", "offline", "stopped":
		return "down"
	default:
		return status
	}
}

func normalizeServiceStatus(status string) string {
	switch status {
	case "healthy", "running", "up":
		return "running"
	case "degraded", "unhealthy":
		return "degraded"
	case "stopped", "down", "exited":
		return "stopped"
	case "":
		return "running"
	default:
		return status
	}
}

func normalizeContainerStatus(status string) string {
	switch status {
	case "healthy", "running", "up":
		return "running"
	case "exited", "stopped", "dead", "down":
		return "exited"
	default:
		return status
	}
}

func normalizeHealth(health string) string {
	switch health {
	case "", "unknown":
		return "unknown"
	case "degraded":
		return "unhealthy"
	default:
		return health
	}
}

func rollupServiceStatus(current, containerStatus, health string) string {
	if containerStatus != "running" {
		return "stopped"
	}
	if health == "unhealthy" && current == "running" {
		return "degraded"
	}
	return current
}

func lookupLogContext(services []domain.ServiceSnapshot, serviceID, containerID string) (string, string) {
	for _, service := range services {
		if service.ID != serviceID {
			continue
		}

		for _, container := range service.Containers {
			if container.ID == containerID {
				return container.Name, serviceTag(service, container.Name)
			}
		}
	}

	return "", ""
}

func serviceTag(service domain.ServiceSnapshot, containerName string) string {
	if service.ComposeProject == "" || service.Name == "" {
		return containerName
	}

	prefix := service.Name + "-"
	if len(containerName) > len(prefix) && containerName[:len(prefix)] == prefix {
		remainder := containerName[len(prefix):]
		for index := 0; index < len(remainder); index++ {
			if remainder[index] == '-' {
				return remainder[:index]
			}
		}
		return remainder
	}

	return service.Name
}
