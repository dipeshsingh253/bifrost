package seed

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dipesh/bifrost/backend/internal/domain"
	"github.com/dipesh/bifrost/backend/internal/store"
)

const (
	TenantIDDemo          = "c4fa4fd6-8101-4c17-a89a-a46ad3caab18"
	UserIDOwner           = "d0ed9f63-3f89-4d83-9df3-18eeb11ec31a"
	ServerIDDevServer     = "9329fc4b-4d0d-4d3a-8fcb-53f87d7d9e0d"
	ServiceIDServiceA     = "9df9d26e-1df6-456d-97b2-c5ecf6ecf18f"
	ServiceIDSearchStack  = "1f4f333d-b869-4dad-bba4-a687afcc89a7"
	ServiceIDEdgeProxy    = "95f6f969-df1f-479d-b5f2-a31d9e9afcb9"
	ContainerIDAPI1       = "34ad2a51-2d8d-4a3c-a05e-a6773d3e4dfc"
	ContainerIDWorker1    = "98ec4147-cf92-4782-8650-6e95bc0d311d"
	ContainerIDWeb1       = "9ea6f294-fb37-4334-b567-a90e358f4965"
	ContainerIDQdrant1    = "c489d2a7-7e2e-473b-9737-c209c6b86e60"
	ContainerIDIngest1    = "530964e8-83fe-457b-a8cf-e721d93288f0"
	ContainerIDEdgeProxy1 = "52b8df81-5b68-4ac5-933a-06ab2d5c0c18"
	AgentIDDemo           = "5b8b7684-f4e3-44a9-8fc6-a58a6110b541"
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

	containerPoints := func(keys []string, base map[string]float64, variance float64) []domain.ContainerMetricPoint {
		result := make([]domain.ContainerMetricPoint, 0, 20)
		for i := 19; i >= 0; i-- {
			values := make(map[string]float64, len(keys))
			for index, key := range keys {
				offset := float64((i+index)%5) * variance
				values[key] = base[key] + offset
			}
			result = append(result, domain.ContainerMetricPoint{
				Timestamp: now.Add(-time.Duration(i) * 3 * time.Minute),
				Values:    values,
			})
		}
		return result
	}

	server := domain.Server{
		ID:             ServerIDDevServer,
		TenantID:       TenantIDDemo,
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
			ID:           ContainerIDAPI1,
			ServiceID:    ServiceIDServiceA,
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
			ID:           ContainerIDWorker1,
			ServiceID:    ServiceIDServiceA,
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
			ID:           ContainerIDWeb1,
			ServiceID:    ServiceIDServiceA,
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
			ID:           ContainerIDQdrant1,
			ServiceID:    ServiceIDSearchStack,
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
			ID:           ContainerIDIngest1,
			ServiceID:    ServiceIDSearchStack,
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
			ID:           ContainerIDEdgeProxy1,
			ServiceID:    ServiceIDEdgeProxy,
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
			ID:               ServiceIDServiceA,
			TenantID:         TenantIDDemo,
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
			ID:               ServiceIDSearchStack,
			TenantID:         TenantIDDemo,
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
			ID:               ServiceIDEdgeProxy,
			TenantID:         TenantIDDemo,
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

	containerIDs := []string{
		ContainerIDAPI1,
		ContainerIDWorker1,
		ContainerIDWeb1,
		ContainerIDQdrant1,
		ContainerIDIngest1,
		ContainerIDEdgeProxy1,
	}

	containerMetrics := map[string]domain.ContainerMetricBundle{
		server.ID: {
			CPU: containerPoints(containerIDs, map[string]float64{
				ContainerIDAPI1:       6.8,
				ContainerIDWorker1:    2.7,
				ContainerIDWeb1:       1.2,
				ContainerIDQdrant1:    3.9,
				ContainerIDIngest1:    10.4,
				ContainerIDEdgeProxy1: 0.5,
			}, 0.35),
			Memory: containerPoints(containerIDs, map[string]float64{
				ContainerIDAPI1:       372,
				ContainerIDWorker1:    208,
				ContainerIDWeb1:       124,
				ContainerIDQdrant1:    624,
				ContainerIDIngest1:    420,
				ContainerIDEdgeProxy1: 48,
			}, 8),
			Network: containerPoints(containerIDs, map[string]float64{
				ContainerIDAPI1:       0.11,
				ContainerIDWorker1:    0.03,
				ContainerIDWeb1:       0.08,
				ContainerIDQdrant1:    0.06,
				ContainerIDIngest1:    0.09,
				ContainerIDEdgeProxy1: 0.04,
			}, 0.01),
		},
	}

	logs := map[string][]domain.LogLine{
		ServiceIDServiceA: {
			log(ServerIDDevServer, ServiceIDServiceA, ContainerIDAPI1, "service-a-backend-1", "backend", "info", "HTTP server started on :8000", now.Add(-5*time.Minute)),
			log(ServerIDDevServer, ServiceIDServiceA, ContainerIDWorker1, "service-a-worker-1", "worker", "warn", "retrying failed webhook delivery", now.Add(-4*time.Minute)),
			log(ServerIDDevServer, ServiceIDServiceA, ContainerIDAPI1, "service-a-backend-1", "backend", "info", "completed GET /api/v1/tasks in 42ms", now.Add(-90*time.Second)),
		},
		ServiceIDSearchStack: {
			log(ServerIDDevServer, ServiceIDSearchStack, ContainerIDIngest1, "search-stack-ingest-1", "ingest", "error", "qdrant index sync exceeded deadline", now.Add(-6*time.Minute)),
			log(ServerIDDevServer, ServiceIDSearchStack, ContainerIDQdrant1, "search-stack-qdrant-1", "qdrant", "info", "snapshot completed successfully", now.Add(-3*time.Minute)),
		},
		ServiceIDEdgeProxy: {
			log(ServerIDDevServer, ServiceIDEdgeProxy, ContainerIDEdgeProxy1, "edge-proxy-1", "edge-proxy-1", "info", "Serving static assets on :80", now.Add(-2*time.Minute)),
		},
	}

	return store.SeedData{
		Users: []domain.User{
			{
				ID:        UserIDOwner,
				TenantID:  TenantIDDemo,
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
				ID:         AgentIDDemo,
				TenantID:   TenantIDDemo,
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
		ID:            mustSeedUUIDString(),
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

func mustSeedUUIDString() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}

	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80

	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(bytes[0:4]),
		hex.EncodeToString(bytes[4:6]),
		hex.EncodeToString(bytes[6:8]),
		hex.EncodeToString(bytes[8:10]),
		hex.EncodeToString(bytes[10:16]),
	)
}
