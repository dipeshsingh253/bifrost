package monitoring

import (
	"context"

	shareddb "github.com/dipesh/bifrost/backend/internal/shared/database"
)

type repository struct {
	store    Store
	timeouts shareddb.QueryTimeouts
}

func NewRepository(dataStore Store, timeouts shareddb.QueryTimeouts) Repository {
	return &repository{
		store:    dataStore,
		timeouts: timeouts,
	}
}

func (r *repository) ListServers(ctx context.Context, tenantID string) ([]Server, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() ([]Server, error) {
		return r.store.ListServers(tenantID), nil
	})
}

func (r *repository) ServerByID(ctx context.Context, tenantID, serverID string) (Server, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() (Server, error) {
		return r.store.ServerByID(tenantID, serverID)
	})
}

func (r *repository) ServicesByServer(ctx context.Context, tenantID, serverID string) ([]MonitoredService, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() ([]MonitoredService, error) {
		return r.store.ServicesByServer(tenantID, serverID), nil
	})
}

func (r *repository) ServiceByID(ctx context.Context, tenantID, serviceID string) (MonitoredService, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() (MonitoredService, error) {
		return r.store.ServiceByID(tenantID, serviceID)
	})
}

func (r *repository) ProjectByID(ctx context.Context, tenantID, serverID, projectID string) (MonitoredService, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() (MonitoredService, error) {
		return r.store.ProjectByID(tenantID, serverID, projectID)
	})
}

func (r *repository) ProjectsByServer(ctx context.Context, tenantID, serverID string) ([]MonitoredService, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() ([]MonitoredService, error) {
		return r.store.ProjectsByServer(tenantID, serverID), nil
	})
}

func (r *repository) StandaloneContainersByServer(ctx context.Context, tenantID, serverID string) ([]Container, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() ([]Container, error) {
		return r.store.StandaloneContainersByServer(tenantID, serverID), nil
	})
}

func (r *repository) MetricsByServer(ctx context.Context, serverID string) ([]MetricSeries, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() ([]MetricSeries, error) {
		return r.store.MetricsByServer(serverID), nil
	})
}

func (r *repository) LogsByService(ctx context.Context, serviceID string) ([]LogLine, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() ([]LogLine, error) {
		return r.store.LogsByService(serviceID), nil
	})
}

func (r *repository) LogsByContainer(ctx context.Context, serviceID, containerID string) ([]LogLine, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() ([]LogLine, error) {
		return r.store.LogsByContainer(serviceID, containerID), nil
	})
}

func (r *repository) ServerBundle(ctx context.Context, tenantID, serverID string) (ServerBundle, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() (ServerBundle, error) {
		return r.store.ServerBundle(tenantID, serverID)
	})
}

func (r *repository) ContainerByID(ctx context.Context, tenantID, serverID, containerID string) (Container, MonitoredService, error) {
	type result struct {
		container Container
		service   MonitoredService
	}
	value, err := shareddb.WithTimeout(ctx, r.timeouts.Read, func() (result, error) {
		container, service, err := r.store.ContainerByID(tenantID, serverID, containerID)
		return result{container: container, service: service}, err
	})
	return value.container, value.service, err
}

func (r *repository) ProjectMetrics(ctx context.Context, tenantID, serverID, projectID string) (ContainerMetricBundle, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() (ContainerMetricBundle, error) {
		return r.store.ProjectMetrics(tenantID, serverID, projectID)
	})
}

func (r *repository) ContainerMetrics(ctx context.Context, tenantID, serverID, containerID string) (ContainerMetricHistory, Container, MonitoredService, error) {
	type result struct {
		metrics   ContainerMetricHistory
		container Container
		service   MonitoredService
	}
	value, err := shareddb.WithTimeout(ctx, r.timeouts.Read, func() (result, error) {
		metrics, container, service, err := r.store.ContainerMetrics(tenantID, serverID, containerID)
		return result{metrics: metrics, container: container, service: service}, err
	})
	return value.metrics, value.container, value.service, err
}

func (r *repository) ProjectEvents(ctx context.Context, tenantID, serverID, projectID string) ([]EventLog, MonitoredService, error) {
	type result struct {
		events  []EventLog
		service MonitoredService
	}
	value, err := shareddb.WithTimeout(ctx, r.timeouts.Read, func() (result, error) {
		events, service, err := r.store.ProjectEvents(tenantID, serverID, projectID)
		return result{events: events, service: service}, err
	})
	return value.events, value.service, err
}

func (r *repository) ContainerEvents(ctx context.Context, tenantID, serverID, containerID string) ([]EventLog, Container, MonitoredService, error) {
	type result struct {
		events    []EventLog
		container Container
		service   MonitoredService
	}
	value, err := shareddb.WithTimeout(ctx, r.timeouts.Read, func() (result, error) {
		events, container, service, err := r.store.ContainerEvents(tenantID, serverID, containerID)
		return result{events: events, container: container, service: service}, err
	})
	return value.events, value.container, value.service, err
}

func (r *repository) ContainerEnv(ctx context.Context, tenantID, serverID, containerID string) (map[string]string, Container, MonitoredService, error) {
	type result struct {
		env       map[string]string
		container Container
		service   MonitoredService
	}
	value, err := shareddb.WithTimeout(ctx, r.timeouts.Read, func() (result, error) {
		env, container, service, err := r.store.ContainerEnv(tenantID, serverID, containerID)
		return result{env: env, container: container, service: service}, err
	})
	return value.env, value.container, value.service, err
}
