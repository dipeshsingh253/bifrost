package seed

import (
	"fmt"
	"time"

	"github.com/dipesh/bifrost/backend/internal/domain"
	"github.com/dipesh/bifrost/backend/internal/store"
)

func Data() store.SeedData {
	now := time.Now().UTC()
	points := func(base float64, variance float64) []domain.MetricPoint {
		result := make([]domain.MetricPoint, 0, 20)
		for i := 19; i >= 0; i-- {
			offset := float64((i % 5)) * variance
			result = append(result, domain.MetricPoint{
				Timestamp: now.Add(-time.Duration(i) * 3 * time.Minute),
				Value:     base + offset,
			})
		}
		return result
	}

	containerPoints := func(names []string, base map[string]float64, variance float64) []domain.ContainerMetricPoint {
		result := make([]domain.ContainerMetricPoint, 0, 20)
		for i := 19; i >= 0; i-- {
			values := make(map[string]float64, len(names))
			for index, name := range names {
				offset := float64((i+index)%5) * variance
				values[name] = base[name] + offset
			}
			result = append(result, domain.ContainerMetricPoint{
				Timestamp: now.Add(-time.Duration(i) * 3 * time.Minute),
				Values:    values,
			})
		}
		return result
	}

	server := domain.Server{
		ID:             "srv-dev-server",
		TenantID:       "tenant-demo",
		Name:           "dev-server",
		Hostname:       "ubuntu-bifrost",
		PublicIP:       "203.0.113.42",
		AgentVersion:   "0.1.0",
		Status:         "up",
		LastSeenAt:     now,
		UptimeSeconds:  662 * 24 * 60 * 60,
		CPUUsagePct:    16.3,
		MemoryUsagePct: 58.1,
		DiskUsagePct:   42.7,
		NetworkRXMB:    0.84,
		NetworkTXMB:    0.27,
		LoadAverage:    "0.24 0.18 0.12",
		OS:             "Ubuntu 24.04.2 LTS",
		Kernel:         "6.8.0-59-generic",
		CPUModel:       "AMD EPYC Processor",
		CPUCores:       4,
		CPUThreads:     8,
		TotalMemoryGB:  16,
		TotalDiskGB:    256,
	}

	apiContainers := []domain.Container{
		{
			ID:           "ctr-api-1",
			ServiceID:    "svc-service-a",
			Name:         "service-a-backend-1",
			Image:        "ghcr.io/acme/service-a-backend:latest",
			Status:       "running",
			Health:       "healthy",
			CPUUsagePct:  7.8,
			MemoryMB:     384,
			RestartCount: 0,
			Uptime:       "6d 02h",
			Ports:        []string{"8000:8000"},
			Command:      "bin/api",
			LastSeenAt:   now,
		},
		{
			ID:           "ctr-worker-1",
			ServiceID:    "svc-service-a",
			Name:         "service-a-worker-1",
			Image:        "ghcr.io/acme/service-a-worker:latest",
			Status:       "running",
			Health:       "healthy",
			CPUUsagePct:  3.1,
			MemoryMB:     216,
			RestartCount: 1,
			Uptime:       "5d 19h",
			Ports:        []string{},
			Command:      "bin/worker",
			LastSeenAt:   now,
		},
		{
			ID:           "ctr-web-1",
			ServiceID:    "svc-service-a",
			Name:         "service-a-frontend-1",
			Image:        "ghcr.io/acme/service-a-frontend:latest",
			Status:       "running",
			Health:       "healthy",
			CPUUsagePct:  1.4,
			MemoryMB:     128,
			RestartCount: 0,
			Uptime:       "6d 03h",
			Ports:        []string{"3000:3000"},
			Command:      "node server.js",
			LastSeenAt:   now,
		},
	}

	searchContainers := []domain.Container{
		{
			ID:           "ctr-qdrant-1",
			ServiceID:    "svc-search-stack",
			Name:         "search-stack-qdrant-1",
			Image:        "qdrant/qdrant:v1.13.4",
			Status:       "running",
			Health:       "healthy",
			CPUUsagePct:  4.3,
			MemoryMB:     640,
			RestartCount: 0,
			Uptime:       "14d 08h",
			Ports:        []string{"6333:6333"},
			Command:      "./entrypoint.sh",
			LastSeenAt:   now,
		},
		{
			ID:           "ctr-ingest-1",
			ServiceID:    "svc-search-stack",
			Name:         "search-stack-ingest-1",
			Image:        "ghcr.io/acme/search-ingest:latest",
			Status:       "running",
			Health:       "degraded",
			CPUUsagePct:  12.2,
			MemoryMB:     432,
			RestartCount: 2,
			Uptime:       "1d 12h",
			Ports:        []string{},
			Command:      "bin/ingest",
			LastSeenAt:   now,
		},
	}

	standaloneContainers := []domain.Container{
		{
			ID:           "ctr-edge-proxy-1",
			ServiceID:    "svc-edge-proxy",
			Name:         "edge-proxy-1",
			Image:        "nginx:1.27-alpine",
			Status:       "running",
			Health:       "healthy",
			CPUUsagePct:  0.7,
			MemoryMB:     48,
			RestartCount: 0,
			Uptime:       "9d 04h",
			Ports:        []string{"8080:80"},
			Command:      "nginx -g 'daemon off;'",
			LastSeenAt:   now,
		},
	}

	services := []domain.Service{
		{
			ID:               "svc-service-a",
			TenantID:         "tenant-demo",
			ServerID:         server.ID,
			Name:             "service-a",
			ComposeProject:   "service-a",
			Status:           "running",
			ContainerCount:   len(apiContainers),
			RestartCount:     1,
			PublishedPorts:   []string{"3000", "8000"},
			Containers:       apiContainers,
			LastLogTimestamp: now.Add(-30 * time.Second),
		},
		{
			ID:               "svc-search-stack",
			TenantID:         "tenant-demo",
			ServerID:         server.ID,
			Name:             "search-stack",
			ComposeProject:   "search-stack",
			Status:           "degraded",
			ContainerCount:   len(searchContainers),
			RestartCount:     2,
			PublishedPorts:   []string{"6333"},
			Containers:       searchContainers,
			LastLogTimestamp: now.Add(-90 * time.Second),
		},
		{
			ID:               "svc-edge-proxy",
			TenantID:         "tenant-demo",
			ServerID:         server.ID,
			Name:             "edge-proxy-1",
			ComposeProject:   "",
			Status:           "running",
			ContainerCount:   len(standaloneContainers),
			RestartCount:     0,
			PublishedPorts:   []string{"8080"},
			Containers:       standaloneContainers,
			LastLogTimestamp: now.Add(-45 * time.Second),
		},
	}

	metrics := map[string][]domain.MetricSeries{
		server.ID: {
			{Key: "cpu_usage_pct", Unit: "%", Points: points(14, 1.6)},
			{Key: "memory_usage_pct", Unit: "%", Points: points(56, 0.5)},
			{Key: "disk_usage_pct", Unit: "%", Points: points(41, 0.4)},
			{Key: "network_rx_mb", Unit: "MB/s", Points: points(0.22, 0.15)},
			{Key: "network_tx_mb", Unit: "MB/s", Points: points(0.11, 0.08)},
			{Key: "disk_read_mb", Unit: "MB/s", Points: points(0.34, 0.06)},
			{Key: "disk_write_mb", Unit: "MB/s", Points: points(0.19, 0.04)},
		},
	}

	containerNames := []string{
		"service-a-backend-1",
		"service-a-worker-1",
		"service-a-frontend-1",
		"search-stack-qdrant-1",
		"search-stack-ingest-1",
		"edge-proxy-1",
	}

	containerMetrics := map[string]domain.ContainerMetricBundle{
		server.ID: {
			CPU: containerPoints(containerNames, map[string]float64{
				"service-a-backend-1":   6.8,
				"service-a-worker-1":    2.7,
				"service-a-frontend-1":  1.2,
				"search-stack-qdrant-1": 3.9,
				"search-stack-ingest-1": 10.4,
				"edge-proxy-1":          0.5,
			}, 0.35),
			Memory: containerPoints(containerNames, map[string]float64{
				"service-a-backend-1":   372,
				"service-a-worker-1":    208,
				"service-a-frontend-1":  124,
				"search-stack-qdrant-1": 624,
				"search-stack-ingest-1": 420,
				"edge-proxy-1":          48,
			}, 8),
			Network: containerPoints(containerNames, map[string]float64{
				"service-a-backend-1":   0.11,
				"service-a-worker-1":    0.03,
				"service-a-frontend-1":  0.08,
				"search-stack-qdrant-1": 0.06,
				"search-stack-ingest-1": 0.09,
				"edge-proxy-1":          0.04,
			}, 0.01),
		},
	}

	logs := map[string][]domain.LogLine{
		"svc-service-a": {
			log("srv-dev-server", "svc-service-a", "ctr-api-1", "service-a-backend-1", "backend", "info", "HTTP server started on :8000", now.Add(-5*time.Minute)),
			log("srv-dev-server", "svc-service-a", "ctr-worker-1", "service-a-worker-1", "worker", "warn", "retrying failed webhook delivery", now.Add(-4*time.Minute)),
			log("srv-dev-server", "svc-service-a", "ctr-api-1", "service-a-backend-1", "backend", "info", "completed GET /api/v1/tasks in 42ms", now.Add(-90*time.Second)),
		},
		"svc-search-stack": {
			log("srv-dev-server", "svc-search-stack", "ctr-ingest-1", "search-stack-ingest-1", "ingest", "error", "qdrant index sync exceeded deadline", now.Add(-6*time.Minute)),
			log("srv-dev-server", "svc-search-stack", "ctr-qdrant-1", "search-stack-qdrant-1", "qdrant", "info", "snapshot completed successfully", now.Add(-3*time.Minute)),
		},
		"svc-edge-proxy": {
			log("srv-dev-server", "svc-edge-proxy", "ctr-edge-proxy-1", "edge-proxy-1", "edge-proxy-1", "info", "Serving static assets on :80", now.Add(-2*time.Minute)),
		},
	}

	return store.SeedData{
		Users: []domain.User{
			{
				ID:        "usr-owner",
				TenantID:  "tenant-demo",
				Email:     "owner@bifrost.local",
				Name:      "Bifrost Owner",
				Password:  "bifrost123",
				Role:      domain.RoleOwner,
				AuthToken: "demo-owner-token",
			},
		},
		Servers:          []domain.Server{server},
		Services:         services,
		Metrics:          metrics,
		ContainerMetrics: containerMetrics,
		Logs:             logs,
		Agents: []domain.Agent{
			{
				ID:         "agt-demo",
				TenantID:   "tenant-demo",
				ServerID:   server.ID,
				Name:       "demo-agent",
				APIKey:     "demo-agent-key",
				Version:    "0.1.0",
				LastSeenAt: now,
				EnrolledAt: now.Add(-24 * time.Hour),
				ServerName: server.Name,
				Hostname:   server.Hostname,
			},
		},
	}
}

func log(serverID, serviceID, containerID, containerName, serviceTag, level, message string, timestamp time.Time) domain.LogLine {
	return domain.LogLine{
		ID:            fmt.Sprintf("%s-%d", containerID, timestamp.Unix()),
		ServerID:      serverID,
		ServiceID:     serviceID,
		ContainerID:   containerID,
		ContainerName: containerName,
		ServiceTag:    serviceTag,
		Level:         level,
		Message:       message,
		Timestamp:     timestamp,
	}
}
