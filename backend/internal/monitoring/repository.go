package monitoring

import "context"

type Store interface {
	ListServers(tenantID string) []Server
	ServerByID(tenantID, serverID string) (Server, error)
	ServicesByServer(tenantID, serverID string) []MonitoredService
	ServiceByID(tenantID, serviceID string) (MonitoredService, error)
	ProjectByID(tenantID, serverID, projectID string) (MonitoredService, error)
	ProjectsByServer(tenantID, serverID string) []MonitoredService
	StandaloneContainersByServer(tenantID, serverID string) []Container
	MetricsByServer(serverID string) []MetricSeries
	LogsByService(serviceID string) []LogLine
	LogsByContainer(serviceID, containerID string) []LogLine
	ServerBundle(tenantID, serverID string) (ServerBundle, error)
	ContainerByID(tenantID, serverID, containerID string) (Container, MonitoredService, error)
	ProjectMetrics(tenantID, serverID, projectID string) (ContainerMetricBundle, error)
	ContainerMetrics(tenantID, serverID, containerID string) (ContainerMetricHistory, Container, MonitoredService, error)
	ProjectEvents(tenantID, serverID, projectID string) ([]EventLog, MonitoredService, error)
	ContainerEvents(tenantID, serverID, containerID string) ([]EventLog, Container, MonitoredService, error)
	ContainerEnv(tenantID, serverID, containerID string) (map[string]string, Container, MonitoredService, error)
}

type Repository interface {
	ListServers(ctx context.Context, tenantID string) ([]Server, error)
	ServerByID(ctx context.Context, tenantID, serverID string) (Server, error)
	ServicesByServer(ctx context.Context, tenantID, serverID string) ([]MonitoredService, error)
	ServiceByID(ctx context.Context, tenantID, serviceID string) (MonitoredService, error)
	ProjectByID(ctx context.Context, tenantID, serverID, projectID string) (MonitoredService, error)
	ProjectsByServer(ctx context.Context, tenantID, serverID string) ([]MonitoredService, error)
	StandaloneContainersByServer(ctx context.Context, tenantID, serverID string) ([]Container, error)
	MetricsByServer(ctx context.Context, serverID string) ([]MetricSeries, error)
	LogsByService(ctx context.Context, serviceID string) ([]LogLine, error)
	LogsByContainer(ctx context.Context, serviceID, containerID string) ([]LogLine, error)
	ServerBundle(ctx context.Context, tenantID, serverID string) (ServerBundle, error)
	ContainerByID(ctx context.Context, tenantID, serverID, containerID string) (Container, MonitoredService, error)
	ProjectMetrics(ctx context.Context, tenantID, serverID, projectID string) (ContainerMetricBundle, error)
	ContainerMetrics(ctx context.Context, tenantID, serverID, containerID string) (ContainerMetricHistory, Container, MonitoredService, error)
	ProjectEvents(ctx context.Context, tenantID, serverID, projectID string) ([]EventLog, MonitoredService, error)
	ContainerEvents(ctx context.Context, tenantID, serverID, containerID string) ([]EventLog, Container, MonitoredService, error)
	ContainerEnv(ctx context.Context, tenantID, serverID, containerID string) (map[string]string, Container, MonitoredService, error)
}
