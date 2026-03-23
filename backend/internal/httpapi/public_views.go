package httpapi

import (
	"time"

	"github.com/dipesh/bifrost/backend/internal/domain"
)

type serverView struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	Name           string    `json:"name"`
	Hostname       string    `json:"hostname"`
	PublicIP       string    `json:"public_ip"`
	AgentVersion   string    `json:"agent_version"`
	Status         string    `json:"status"`
	LastSeenAt     time.Time `json:"last_seen_at"`
	UptimeSeconds  int64     `json:"uptime_seconds"`
	CPUUsagePct    float64   `json:"cpu_usage_pct"`
	MemoryUsagePct float64   `json:"memory_usage_pct"`
	DiskUsagePct   float64   `json:"disk_usage_pct"`
	NetworkRXMB    float64   `json:"network_rx_mb"`
	NetworkTXMB    float64   `json:"network_tx_mb"`
	LoadAverage    string    `json:"load_average"`
	OS             string    `json:"os"`
	Kernel         string    `json:"kernel"`
	CPUModel       string    `json:"cpu_model"`
	CPUCores       int       `json:"cpu_cores"`
	CPUThreads     int       `json:"cpu_threads"`
	TotalMemoryGB  float64   `json:"total_memory_gb"`
	TotalDiskGB    float64   `json:"total_disk_gb"`
}

type serviceView struct {
	ID               string          `json:"id"`
	TenantID         string          `json:"tenant_id"`
	ServerID         string          `json:"server_id"`
	Name             string          `json:"name"`
	ComposeProject   string          `json:"compose_project"`
	Status           string          `json:"status"`
	ContainerCount   int             `json:"container_count"`
	RestartCount     int             `json:"restart_count"`
	PublishedPorts   []string        `json:"published_ports"`
	Containers       []containerView `json:"containers"`
	LastLogTimestamp time.Time       `json:"last_log_timestamp"`
}

type containerView struct {
	ID           string    `json:"id"`
	ServiceID    string    `json:"service_id"`
	Name         string    `json:"name"`
	Image        string    `json:"image"`
	Status       string    `json:"status"`
	Health       string    `json:"health"`
	CPUUsagePct  float64   `json:"cpu_usage_pct"`
	MemoryMB     float64   `json:"memory_mb"`
	NetworkMB    float64   `json:"network_mb"`
	RestartCount int       `json:"restart_count"`
	Uptime       string    `json:"uptime"`
	Ports        []string  `json:"ports"`
	Command      string    `json:"command"`
	LastSeenAt   time.Time `json:"last_seen_at"`
}

type logLineView struct {
	ID            string    `json:"id"`
	ServerID      string    `json:"server_id"`
	ServiceID     string    `json:"service_id"`
	ContainerID   string    `json:"container_id"`
	ContainerName string    `json:"containerName"`
	ServiceTag    string    `json:"serviceTag"`
	Level         string    `json:"level"`
	Message       string    `json:"message"`
	Timestamp     time.Time `json:"timestamp"`
}

func newServerView(server domain.Server) serverView {
	return serverView{
		ID:             server.ID,
		TenantID:       server.TenantID,
		Name:           server.Name,
		Hostname:       server.Hostname,
		PublicIP:       server.PublicIP,
		AgentVersion:   server.AgentVersion,
		Status:         server.Status,
		LastSeenAt:     server.LastSeenAt,
		UptimeSeconds:  server.UptimeSeconds,
		CPUUsagePct:    server.CPUUsagePct,
		MemoryUsagePct: server.MemoryUsagePct,
		DiskUsagePct:   server.DiskUsagePct,
		NetworkRXMB:    server.NetworkRXMB,
		NetworkTXMB:    server.NetworkTXMB,
		LoadAverage:    server.LoadAverage,
		OS:             server.OS,
		Kernel:         server.Kernel,
		CPUModel:       server.CPUModel,
		CPUCores:       server.CPUCores,
		CPUThreads:     server.CPUThreads,
		TotalMemoryGB:  server.TotalMemoryGB,
		TotalDiskGB:    server.TotalDiskGB,
	}
}

func newServiceView(service domain.Service) serviceView {
	containers := make([]containerView, 0, len(service.Containers))
	for _, container := range service.Containers {
		containers = append(containers, newContainerView(container))
	}

	return serviceView{
		ID:               service.ID,
		TenantID:         service.TenantID,
		ServerID:         service.ServerID,
		Name:             service.Name,
		ComposeProject:   service.ComposeProject,
		Status:           service.Status,
		ContainerCount:   service.ContainerCount,
		RestartCount:     service.RestartCount,
		PublishedPorts:   append([]string(nil), service.PublishedPorts...),
		Containers:       containers,
		LastLogTimestamp: service.LastLogTimestamp,
	}
}

func newContainerView(container domain.Container) containerView {
	return containerView{
		ID:           container.ID,
		ServiceID:    container.ServiceID,
		Name:         container.Name,
		Image:        container.Image,
		Status:       container.Status,
		Health:       container.Health,
		CPUUsagePct:  container.CPUUsagePct,
		MemoryMB:     container.MemoryMB,
		NetworkMB:    container.NetworkMB,
		RestartCount: container.RestartCount,
		Uptime:       container.Uptime,
		Ports:        append([]string(nil), container.Ports...),
		Command:      container.Command,
		LastSeenAt:   container.LastSeenAt,
	}
}

func newServicesView(services []domain.Service) []serviceView {
	result := make([]serviceView, 0, len(services))
	for _, service := range services {
		result = append(result, newServiceView(service))
	}
	return result
}

func newStandaloneContainersView(services []domain.Service) []containerView {
	result := make([]containerView, 0)
	for _, service := range services {
		for _, container := range service.Containers {
			result = append(result, newContainerView(container))
		}
	}
	return result
}

func remapContainerMetricBundle(bundle domain.ContainerMetricBundle, services []domain.Service) domain.ContainerMetricBundle {
	return domain.ContainerMetricBundle{
		CPU:     remapContainerMetricPoints(bundle.CPU),
		Memory:  remapContainerMetricPoints(bundle.Memory),
		Network: remapContainerMetricPoints(bundle.Network),
	}
}

func remapContainerMetricPoints(points []domain.ContainerMetricPoint) []domain.ContainerMetricPoint {
	result := make([]domain.ContainerMetricPoint, 0, len(points))
	for _, point := range points {
		values := make(map[string]float64, len(point.Values))
		for key, value := range point.Values {
			values[key] = value
		}
		result = append(result, domain.ContainerMetricPoint{
			Timestamp: point.Timestamp,
			Values:    values,
		})
	}
	return result
}

func newLogLinesView(lines []domain.LogLine) []logLineView {
	result := make([]logLineView, 0, len(lines))
	for _, line := range lines {
		result = append(result, logLineView{
			ID:            line.ID,
			ServerID:      line.ServerID,
			ServiceID:     line.ServiceID,
			ContainerID:   line.ContainerID,
			ContainerName: line.ContainerName,
			ServiceTag:    line.ServiceTag,
			Level:         line.Level,
			Message:       line.Message,
			Timestamp:     line.Timestamp,
		})
	}
	return result
}
