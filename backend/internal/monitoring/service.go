package monitoring

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListServers(ctx context.Context, tenantID string) ([]Server, error) {
	return s.repo.ListServers(ctx, tenantID)
}

func (s *Service) ServerByID(ctx context.Context, tenantID, serverID string) (Server, error) {
	return s.repo.ServerByID(ctx, tenantID, serverID)
}

func (s *Service) ServicesByServer(ctx context.Context, tenantID, serverID string) ([]MonitoredService, error) {
	return s.repo.ServicesByServer(ctx, tenantID, serverID)
}

func (s *Service) ServiceByID(ctx context.Context, tenantID, serviceID string) (MonitoredService, error) {
	return s.repo.ServiceByID(ctx, tenantID, serviceID)
}

func (s *Service) ProjectByID(ctx context.Context, tenantID, serverID, projectID string) (MonitoredService, error) {
	return s.repo.ProjectByID(ctx, tenantID, serverID, projectID)
}

func (s *Service) ProjectsByServer(ctx context.Context, tenantID, serverID string) ([]MonitoredService, error) {
	return s.repo.ProjectsByServer(ctx, tenantID, serverID)
}

func (s *Service) StandaloneContainersByServer(ctx context.Context, tenantID, serverID string) ([]Container, error) {
	return s.repo.StandaloneContainersByServer(ctx, tenantID, serverID)
}

func (s *Service) MetricsByServer(ctx context.Context, serverID string) ([]MetricSeries, error) {
	return s.repo.MetricsByServer(ctx, serverID)
}

func (s *Service) LogsByService(ctx context.Context, serviceID string) ([]LogLine, error) {
	return s.repo.LogsByService(ctx, serviceID)
}

func (s *Service) LogsByContainer(ctx context.Context, serviceID, containerID string) ([]LogLine, error) {
	return s.repo.LogsByContainer(ctx, serviceID, containerID)
}

func (s *Service) ServerBundle(ctx context.Context, tenantID, serverID string) (ServerBundle, error) {
	return s.repo.ServerBundle(ctx, tenantID, serverID)
}

func (s *Service) ContainerByID(ctx context.Context, tenantID, serverID, containerID string) (Container, MonitoredService, error) {
	return s.repo.ContainerByID(ctx, tenantID, serverID, containerID)
}

func (s *Service) ProjectMetrics(ctx context.Context, tenantID, serverID, projectID string) (ContainerMetricBundle, error) {
	return s.repo.ProjectMetrics(ctx, tenantID, serverID, projectID)
}

func (s *Service) ContainerMetrics(ctx context.Context, tenantID, serverID, containerID string) (ContainerMetricHistory, Container, MonitoredService, error) {
	return s.repo.ContainerMetrics(ctx, tenantID, serverID, containerID)
}

func (s *Service) ProjectEvents(ctx context.Context, tenantID, serverID, projectID string) ([]EventLog, MonitoredService, error) {
	return s.repo.ProjectEvents(ctx, tenantID, serverID, projectID)
}

func (s *Service) ContainerEvents(ctx context.Context, tenantID, serverID, containerID string) ([]EventLog, Container, MonitoredService, error) {
	return s.repo.ContainerEvents(ctx, tenantID, serverID, containerID)
}

func (s *Service) ContainerEnv(ctx context.Context, tenantID, serverID, containerID string) (map[string]string, Container, MonitoredService, error) {
	return s.repo.ContainerEnv(ctx, tenantID, serverID, containerID)
}
