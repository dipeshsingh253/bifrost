package collector

import (
	"bufio"
	"bytes"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dipesh/bifrost/agent/internal/client"
	"github.com/dipesh/bifrost/agent/internal/config"
)

const (
	agentVersion        = "0.1.0"
	initialSampleDelay  = 250 * time.Millisecond
	diskSectorSizeBytes = 512
	megabyteDivisor     = 1024 * 1024
	hostRootEnv         = "BIFROST_HOST_ROOT"
)

func hostRootPath() string {
	return strings.TrimRight(strings.TrimSpace(os.Getenv(hostRootEnv)), "/")
}

// When the agent runs in Docker, BIFROST_HOST_ROOT can point at a bind-mounted
// copy of the host filesystem so host collectors keep reading real machine data.
func hostPath(root, path string) string {
	trimmed := strings.TrimSpace(path)
	if root == "" {
		if trimmed == "" {
			return "/"
		}
		return trimmed
	}
	if trimmed == "" || trimmed == "/" {
		return root
	}
	return root + "/" + strings.TrimLeft(trimmed, "/")
}

type HostCollector struct {
	lastCountersAt time.Time
	lastCPU        cpuCounters
	lastNetwork    networkCounters
	lastDisk       diskCounters
}

type cpuCounters struct {
	idle  uint64
	total uint64
}

type networkCounters struct {
	rxBytes uint64
	txBytes uint64
}

type diskCounters struct {
	readBytes  uint64
	writeBytes uint64
}

type memoryInfo struct {
	usagePct float64
	totalGB  float64
}

type cpuInfo struct {
	model   string
	cores   int
	threads int
}

func NewHostCollector() *HostCollector {
	return &HostCollector{}
}

func (c *HostCollector) Collect(cfg config.Config) (client.ServerSnapshot, []client.MetricPayload, error) {
	collectedAt := time.Now().UTC()
	hostRoot := hostRootPath()
	loadAverage := readLoadAverage(hostRoot)
	memory := readMemoryInfo(hostRoot)
	diskUsage, totalDiskGB := readDiskUsage(hostPath(hostRoot, "/"))
	uptime := readUptime(hostRoot)
	hostname, _ := os.Hostname()
	publicIP := detectPrimaryIP()
	osName := readOSName(hostRoot)
	kernel := readKernelVersion(hostRoot)
	cpu := readCPUInfo(hostRoot)

	cpuUsagePct, networkRXMB, networkTXMB, diskReadMB, diskWriteMB := c.readRates(collectedAt)

	server := client.ServerSnapshot{
		ID:             cfg.ServerID,
		Name:           cfg.ServerName,
		Hostname:       hostname,
		PublicIP:       publicIP,
		AgentVersion:   agentVersion,
		Status:         "up",
		UptimeSeconds:  uptime,
		CPUUsagePct:    cpuUsagePct,
		MemoryUsagePct: memory.usagePct,
		DiskUsagePct:   diskUsage,
		NetworkRXMB:    networkRXMB,
		NetworkTXMB:    networkTXMB,
		LoadAverage:    loadAverage,
		OS:             osName,
		Kernel:         kernel,
		CPUModel:       cpu.model,
		CPUCores:       cpu.cores,
		CPUThreads:     cpu.threads,
		TotalMemoryGB:  memory.totalGB,
		TotalDiskGB:    totalDiskGB,
		CollectedAt:    collectedAt,
	}

	metrics := buildHostMetrics(cfg.ServerID, collectedAt, server, diskReadMB, diskWriteMB)
	return server, metrics, nil
}

func (c *HostCollector) readRates(now time.Time) (float64, float64, float64, float64, float64) {
	hostRoot := hostRootPath()
	currentCPU := readCPUCounters(hostRoot)
	currentNetwork := readNetworkCounters(hostRoot)
	currentDisk := readDiskCounters(hostRoot)

	if c.lastCountersAt.IsZero() {
		c.lastCPU = currentCPU
		c.lastNetwork = currentNetwork
		c.lastDisk = currentDisk
		c.lastCountersAt = now

		time.Sleep(initialSampleDelay)
		now = time.Now().UTC()
		currentCPU = readCPUCounters(hostRoot)
		currentNetwork = readNetworkCounters(hostRoot)
		currentDisk = readDiskCounters(hostRoot)
	}

	elapsedSeconds := now.Sub(c.lastCountersAt).Seconds()
	if elapsedSeconds <= 0 {
		elapsedSeconds = 1
	}

	cpuUsage := calculateCPUUsage(c.lastCPU, currentCPU)
	networkRX := bytesRateToMB(c.lastNetwork.rxBytes, currentNetwork.rxBytes, elapsedSeconds)
	networkTX := bytesRateToMB(c.lastNetwork.txBytes, currentNetwork.txBytes, elapsedSeconds)
	diskRead := bytesRateToMB(c.lastDisk.readBytes, currentDisk.readBytes, elapsedSeconds)
	diskWrite := bytesRateToMB(c.lastDisk.writeBytes, currentDisk.writeBytes, elapsedSeconds)

	c.lastCPU = currentCPU
	c.lastNetwork = currentNetwork
	c.lastDisk = currentDisk
	c.lastCountersAt = now

	return cpuUsage, networkRX, networkTX, diskRead, diskWrite
}

func buildHostMetrics(serverID string, now time.Time, server client.ServerSnapshot, diskReadMB float64, diskWriteMB float64) []client.MetricPayload {
	keys := []struct {
		key   string
		unit  string
		value float64
	}{
		{key: "cpu_usage_pct", unit: "%", value: server.CPUUsagePct},
		{key: "memory_usage_pct", unit: "%", value: server.MemoryUsagePct},
		{key: "disk_usage_pct", unit: "%", value: server.DiskUsagePct},
		{key: "network_rx_mb", unit: "MB/s", value: server.NetworkRXMB},
		{key: "network_tx_mb", unit: "MB/s", value: server.NetworkTXMB},
		{key: "disk_read_mb", unit: "MB/s", value: diskReadMB},
		{key: "disk_write_mb", unit: "MB/s", value: diskWriteMB},
	}

	metrics := make([]client.MetricPayload, 0, len(keys))
	for _, item := range keys {
		metrics = append(metrics, client.MetricPayload{
			ServerID: serverID,
			Key:      item.key,
			Unit:     item.unit,
			Points: []client.MetricPoint{
				{Timestamp: now, Value: item.value},
			},
		})
	}

	return metrics
}

func readLoadAverage(hostRoot string) string {
	content, err := os.ReadFile(hostPath(hostRoot, "/proc/loadavg"))
	if err != nil {
		return "0.00 0.00 0.00"
	}

	parts := strings.Fields(string(content))
	if len(parts) < 3 {
		return "0.00 0.00 0.00"
	}

	return strings.Join(parts[:3], " ")
}

func readMemoryInfo(hostRoot string) memoryInfo {
	content, err := os.ReadFile(hostPath(hostRoot, "/proc/meminfo"))
	if err != nil {
		return memoryInfo{}
	}

	totalKB, availableKB := parseMeminfo(content)
	if totalKB == 0 {
		return memoryInfo{}
	}

	usedKB := totalKB - availableKB
	return memoryInfo{
		usagePct: (usedKB / totalKB) * 100,
		totalGB:  totalKB / (1024 * 1024),
	}
}

func parseMeminfo(content []byte) (float64, float64) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var totalKB float64
	var availableKB float64

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			totalKB, _ = strconv.ParseFloat(fields[1], 64)
		case "MemAvailable:":
			availableKB, _ = strconv.ParseFloat(fields[1], 64)
		}
	}

	return totalKB, availableKB
}

func readDiskUsage(path string) (float64, float64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0
	}

	total := float64(stat.Blocks) * float64(stat.Bsize)
	free := float64(stat.Bavail) * float64(stat.Bsize)
	if total == 0 {
		return 0, 0
	}

	used := total - free
	return (used / total) * 100, total / megabyteDivisor / 1024
}

func readUptime(hostRoot string) int64 {
	content, err := os.ReadFile(hostPath(hostRoot, "/proc/uptime"))
	if err != nil {
		return 0
	}

	parts := strings.Fields(string(content))
	if len(parts) == 0 {
		return 0
	}

	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}

	return int64(value)
}

func readCPUCounters(hostRoot string) cpuCounters {
	content, err := os.ReadFile(hostPath(hostRoot, "/proc/stat"))
	if err != nil {
		return cpuCounters{}
	}

	return parseCPUCounters(content)
}

func parseCPUCounters(content []byte) cpuCounters {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 || fields[0] != "cpu" {
			continue
		}

		var total uint64
		for _, field := range fields[1:] {
			value, err := strconv.ParseUint(field, 10, 64)
			if err != nil {
				continue
			}
			total += value
		}

		idle, _ := strconv.ParseUint(fields[4], 10, 64)
		if len(fields) > 5 {
			iowait, _ := strconv.ParseUint(fields[5], 10, 64)
			idle += iowait
		}

		return cpuCounters{idle: idle, total: total}
	}

	return cpuCounters{}
}

func calculateCPUUsage(previous, current cpuCounters) float64 {
	totalDelta := float64(current.total - previous.total)
	idleDelta := float64(current.idle - previous.idle)
	if totalDelta <= 0 {
		return 0
	}

	return ((totalDelta - idleDelta) / totalDelta) * 100
}

func readNetworkCounters(hostRoot string) networkCounters {
	content, err := os.ReadFile(hostPath(hostRoot, "/proc/net/dev"))
	if err != nil {
		return networkCounters{}
	}

	return parseNetworkCounters(content)
}

func parseNetworkCounters(content []byte) networkCounters {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var total networkCounters

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.Contains(line, ":") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		if name == "lo" {
			continue
		}

		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}

		rx, errRX := strconv.ParseUint(fields[0], 10, 64)
		tx, errTX := strconv.ParseUint(fields[8], 10, 64)
		if errRX != nil || errTX != nil {
			continue
		}

		total.rxBytes += rx
		total.txBytes += tx
	}

	return total
}

func readDiskCounters(hostRoot string) diskCounters {
	content, err := os.ReadFile(hostPath(hostRoot, "/proc/diskstats"))
	if err != nil {
		return diskCounters{}
	}

	devices := blockDevices(hostRoot)
	return parseDiskCounters(content, devices)
}

func blockDevices(hostRoot string) map[string]struct{} {
	entries, err := os.ReadDir(hostPath(hostRoot, "/sys/block"))
	if err != nil {
		return map[string]struct{}{}
	}

	devices := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "fd") || strings.HasPrefix(name, "sr") {
			continue
		}
		devices[name] = struct{}{}
	}

	return devices
}

func parseDiskCounters(content []byte, devices map[string]struct{}) diskCounters {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var total diskCounters

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}

		name := fields[2]
		if _, ok := devices[name]; !ok {
			continue
		}

		readSectors, errRead := strconv.ParseUint(fields[5], 10, 64)
		writeSectors, errWrite := strconv.ParseUint(fields[9], 10, 64)
		if errRead != nil || errWrite != nil {
			continue
		}

		total.readBytes += readSectors * diskSectorSizeBytes
		total.writeBytes += writeSectors * diskSectorSizeBytes
	}

	return total
}

func bytesRateToMB(previous, current uint64, elapsedSeconds float64) float64 {
	if current < previous || elapsedSeconds <= 0 {
		return 0
	}

	return (float64(current-previous) / megabyteDivisor) / elapsedSeconds
}

func readOSName(hostRoot string) string {
	content, err := os.ReadFile(hostPath(hostRoot, "/etc/os-release"))
	if err != nil {
		return runtime.GOOS
	}

	if pretty := parseOSRelease(content, "PRETTY_NAME"); pretty != "" {
		return pretty
	}

	name := parseOSRelease(content, "NAME")
	version := parseOSRelease(content, "VERSION")
	switch {
	case name != "" && version != "":
		return name + " " + version
	case name != "":
		return name
	default:
		return runtime.GOOS
	}
}

func parseOSRelease(content []byte, key string) string {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	prefix := key + "="
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, prefix) {
			continue
		}

		value := strings.TrimPrefix(line, prefix)
		return strings.Trim(value, `"`)
	}

	return ""
}

func readKernelVersion(hostRoot string) string {
	content, err := os.ReadFile(hostPath(hostRoot, "/proc/sys/kernel/osrelease"))
	if err != nil {
		return runtime.GOARCH
	}

	return strings.TrimSpace(string(content))
}

func readCPUInfo(hostRoot string) cpuInfo {
	content, err := os.ReadFile(hostPath(hostRoot, "/proc/cpuinfo"))
	if err != nil {
		return cpuInfo{
			model:   runtime.GOARCH,
			cores:   runtime.NumCPU(),
			threads: runtime.NumCPU(),
		}
	}

	return parseCPUInfo(content)
}

func parseCPUInfo(content []byte) cpuInfo {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	info := cpuInfo{
		model:   runtime.GOARCH,
		cores:   runtime.NumCPU(),
		threads: runtime.NumCPU(),
	}

	threads := 0
	corePairs := map[string]struct{}{}
	var physicalID string
	var coreID string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			if physicalID != "" || coreID != "" {
				corePairs[physicalID+":"+coreID] = struct{}{}
			}
			physicalID = ""
			coreID = ""
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "processor":
			threads++
		case "model name":
			if info.model == runtime.GOARCH {
				info.model = value
			}
		case "Hardware":
			if info.model == runtime.GOARCH {
				info.model = value
			}
		case "physical id":
			physicalID = value
		case "core id":
			coreID = value
		}
	}

	if physicalID != "" || coreID != "" {
		corePairs[physicalID+":"+coreID] = struct{}{}
	}

	if threads > 0 {
		info.threads = threads
	}
	if len(corePairs) > 0 {
		info.cores = len(corePairs)
	} else if info.threads > 0 {
		info.cores = info.threads
	}

	return info
}

func detectPrimaryIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, address := range addresses {
			var ip net.IP
			switch value := address.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4()
			if ip == nil {
				continue
			}

			return ip.String()
		}
	}

	return ""
}
