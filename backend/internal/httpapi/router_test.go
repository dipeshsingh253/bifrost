package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dipesh/bifrost/backend/internal/config"
	"github.com/dipesh/bifrost/backend/internal/seed"
	"github.com/dipesh/bifrost/backend/internal/store"
)

func TestServerDetailReturnsFrontendBundleShape(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-dev-server", nil)
	request.Header.Set("Authorization", "Bearer demo-owner-token")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Server struct {
				ID            string  `json:"id"`
				Status        string  `json:"status"`
				CPUCores      int     `json:"cpu_cores"`
				CPUThreads    int     `json:"cpu_threads"`
				TotalMemoryGB float64 `json:"total_memory_gb"`
				TotalDiskGB   float64 `json:"total_disk_gb"`
			} `json:"server"`
			Services []struct {
				ID               string `json:"id"`
				Status           string `json:"status"`
				LastLogTimestamp string `json:"last_log_timestamp"`
			} `json:"services"`
			Metrics []struct {
				Key string `json:"key"`
			} `json:"metrics"`
			ContainerMetrics struct {
				CPU     []map[string]any `json:"cpu"`
				Memory  []map[string]any `json:"memory"`
				Network []map[string]any `json:"network"`
			} `json:"containerMetrics"`
		} `json:"data"`
	}

	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !payload.Success {
		t.Fatalf("expected success response")
	}

	if payload.Data.Server.ID != "srv-dev-server" {
		t.Fatalf("expected server id srv-dev-server, got %q", payload.Data.Server.ID)
	}
	if payload.Data.Server.Status != "up" {
		t.Fatalf("expected normalized server status up, got %q", payload.Data.Server.Status)
	}
	if payload.Data.Server.CPUCores == 0 || payload.Data.Server.CPUThreads == 0 {
		t.Fatalf("expected cpu capacity fields to be populated")
	}
	if payload.Data.Server.TotalMemoryGB == 0 || payload.Data.Server.TotalDiskGB == 0 {
		t.Fatalf("expected server capacity totals to be populated")
	}

	if len(payload.Data.Services) == 0 {
		t.Fatalf("expected services in server bundle")
	}
	if payload.Data.Services[0].LastLogTimestamp == "" {
		t.Fatalf("expected service last_log_timestamp to be present")
	}

	foundDiskRead := false
	for _, metric := range payload.Data.Metrics {
		if metric.Key == "disk_read_mb" {
			foundDiskRead = true
			break
		}
	}
	if !foundDiskRead {
		t.Fatalf("expected disk_read_mb metric in bundled metrics")
	}

	if len(payload.Data.ContainerMetrics.CPU) == 0 {
		t.Fatalf("expected container cpu metrics")
	}
	if _, ok := payload.Data.ContainerMetrics.CPU[0]["timestamp"]; !ok {
		t.Fatalf("expected timestamp key in container cpu metric point")
	}
	if _, ok := payload.Data.ContainerMetrics.CPU[0]["service-a-backend-1"]; !ok {
		t.Fatalf("expected container-name keyed series in cpu metric point")
	}
}

func TestServiceLogsIncludeFrontendLogFields(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/services/svc-service-a/logs", nil)
	request.Header.Set("Authorization", "Bearer demo-owner-token")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Logs []struct {
				ContainerID   string `json:"container_id"`
				ContainerName string `json:"containerName"`
				ServiceTag    string `json:"serviceTag"`
			} `json:"logs"`
		} `json:"data"`
	}

	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(payload.Data.Logs) == 0 {
		t.Fatalf("expected logs in service log response")
	}
	if payload.Data.Logs[0].ContainerID == "" || payload.Data.Logs[0].ContainerName == "" || payload.Data.Logs[0].ServiceTag == "" {
		t.Fatalf("expected enriched log fields for frontend log viewer")
	}
}

func TestProjectEndpointsReturnProjectDetailMetricsLogsAndEvents(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	projectDetailRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-dev-server/projects/svc-service-a", nil)
	projectDetailRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	projectDetailResponse := httptest.NewRecorder()
	router.ServeHTTP(projectDetailResponse, projectDetailRequest)

	if projectDetailResponse.Code != http.StatusOK {
		t.Fatalf("expected project detail status 200, got %d", projectDetailResponse.Code)
	}

	projectMetricsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-dev-server/projects/svc-service-a/metrics", nil)
	projectMetricsRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	projectMetricsResponse := httptest.NewRecorder()
	router.ServeHTTP(projectMetricsResponse, projectMetricsRequest)

	if projectMetricsResponse.Code != http.StatusOK {
		t.Fatalf("expected project metrics status 200, got %d", projectMetricsResponse.Code)
	}

	var metricsPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Metrics struct {
				CPU []map[string]any `json:"cpu"`
			} `json:"metrics"`
		} `json:"data"`
	}
	if err := json.Unmarshal(projectMetricsResponse.Body.Bytes(), &metricsPayload); err != nil {
		t.Fatalf("decode project metrics: %v", err)
	}
	if len(metricsPayload.Data.Metrics.CPU) == 0 {
		t.Fatalf("expected project cpu metrics")
	}
	if _, ok := metricsPayload.Data.Metrics.CPU[0]["service-a-backend-1"]; !ok {
		t.Fatalf("expected project metrics to preserve container-name keys")
	}
	if _, ok := metricsPayload.Data.Metrics.CPU[0]["search-stack-qdrant-1"]; ok {
		t.Fatalf("expected project metrics to exclude containers from other projects")
	}

	projectLogsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-dev-server/projects/svc-service-a/logs?search=webhook", nil)
	projectLogsRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	projectLogsResponse := httptest.NewRecorder()
	router.ServeHTTP(projectLogsResponse, projectLogsRequest)

	if projectLogsResponse.Code != http.StatusOK {
		t.Fatalf("expected project logs status 200, got %d", projectLogsResponse.Code)
	}

	var logsPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Logs []struct {
				Message string `json:"message"`
			} `json:"logs"`
		} `json:"data"`
	}
	if err := json.Unmarshal(projectLogsResponse.Body.Bytes(), &logsPayload); err != nil {
		t.Fatalf("decode project logs: %v", err)
	}
	if len(logsPayload.Data.Logs) != 1 || logsPayload.Data.Logs[0].Message == "" {
		t.Fatalf("expected filtered project logs")
	}

	projectEventsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-dev-server/projects/svc-search-stack/events", nil)
	projectEventsRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	projectEventsResponse := httptest.NewRecorder()
	router.ServeHTTP(projectEventsResponse, projectEventsRequest)

	if projectEventsResponse.Code != http.StatusOK {
		t.Fatalf("expected project events status 200, got %d", projectEventsResponse.Code)
	}

	var eventsPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Events []struct {
				Type string `json:"type"`
			} `json:"events"`
		} `json:"data"`
	}
	if err := json.Unmarshal(projectEventsResponse.Body.Bytes(), &eventsPayload); err != nil {
		t.Fatalf("decode project events: %v", err)
	}
	if len(eventsPayload.Data.Events) == 0 {
		t.Fatalf("expected derived project events")
	}
}

func TestContainerEndpointsReturnStandaloneListDetailMetricsLogsEventsAndEnv(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	containersRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-dev-server/containers?standalone=true", nil)
	containersRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	containersResponse := httptest.NewRecorder()
	router.ServeHTTP(containersResponse, containersRequest)

	if containersResponse.Code != http.StatusOK {
		t.Fatalf("expected standalone container list status 200, got %d", containersResponse.Code)
	}

	var containerListPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Containers []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"containers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(containersResponse.Body.Bytes(), &containerListPayload); err != nil {
		t.Fatalf("decode standalone containers: %v", err)
	}
	if len(containerListPayload.Data.Containers) == 0 || containerListPayload.Data.Containers[0].ID != "ctr-edge-proxy-1" {
		t.Fatalf("expected seeded standalone container in list")
	}

	containerDetailRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-dev-server/containers/ctr-api-1", nil)
	containerDetailRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	containerDetailResponse := httptest.NewRecorder()
	router.ServeHTTP(containerDetailResponse, containerDetailRequest)

	if containerDetailResponse.Code != http.StatusOK {
		t.Fatalf("expected container detail status 200, got %d", containerDetailResponse.Code)
	}

	containerMetricsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-dev-server/containers/ctr-api-1/metrics", nil)
	containerMetricsRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	containerMetricsResponse := httptest.NewRecorder()
	router.ServeHTTP(containerMetricsResponse, containerMetricsRequest)

	if containerMetricsResponse.Code != http.StatusOK {
		t.Fatalf("expected container metrics status 200, got %d", containerMetricsResponse.Code)
	}

	var containerMetricsPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Metrics struct {
				CPU []struct {
					Value float64 `json:"value"`
				} `json:"cpu"`
			} `json:"metrics"`
		} `json:"data"`
	}
	if err := json.Unmarshal(containerMetricsResponse.Body.Bytes(), &containerMetricsPayload); err != nil {
		t.Fatalf("decode container metrics: %v", err)
	}
	if len(containerMetricsPayload.Data.Metrics.CPU) == 0 {
		t.Fatalf("expected container cpu history")
	}

	containerLogsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-dev-server/containers/ctr-api-1/logs?limit=1", nil)
	containerLogsRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	containerLogsResponse := httptest.NewRecorder()
	router.ServeHTTP(containerLogsResponse, containerLogsRequest)

	if containerLogsResponse.Code != http.StatusOK {
		t.Fatalf("expected container logs status 200, got %d", containerLogsResponse.Code)
	}

	var containerLogsPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Logs []struct {
				ContainerID string `json:"container_id"`
			} `json:"logs"`
		} `json:"data"`
	}
	if err := json.Unmarshal(containerLogsResponse.Body.Bytes(), &containerLogsPayload); err != nil {
		t.Fatalf("decode container logs: %v", err)
	}
	if len(containerLogsPayload.Data.Logs) != 1 || containerLogsPayload.Data.Logs[0].ContainerID != "ctr-api-1" {
		t.Fatalf("expected limited container log response")
	}

	containerEventsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-dev-server/containers/ctr-ingest-1/events", nil)
	containerEventsRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	containerEventsResponse := httptest.NewRecorder()
	router.ServeHTTP(containerEventsResponse, containerEventsRequest)

	if containerEventsResponse.Code != http.StatusOK {
		t.Fatalf("expected container events status 200, got %d", containerEventsResponse.Code)
	}

	var containerEventsPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Events []struct {
				Type string `json:"type"`
			} `json:"events"`
		} `json:"data"`
	}
	if err := json.Unmarshal(containerEventsResponse.Body.Bytes(), &containerEventsPayload); err != nil {
		t.Fatalf("decode container events: %v", err)
	}
	if len(containerEventsPayload.Data.Events) == 0 {
		t.Fatalf("expected derived container events")
	}

	containerEnvRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-dev-server/containers/ctr-edge-proxy-1/env", nil)
	containerEnvRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	containerEnvResponse := httptest.NewRecorder()
	router.ServeHTTP(containerEnvResponse, containerEnvRequest)

	if containerEnvResponse.Code != http.StatusOK {
		t.Fatalf("expected container env status 200, got %d", containerEnvResponse.Code)
	}

	var containerEnvPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Env map[string]string `json:"env"`
		} `json:"data"`
	}
	if err := json.Unmarshal(containerEnvResponse.Body.Bytes(), &containerEnvPayload); err != nil {
		t.Fatalf("decode container env: %v", err)
	}
	if containerEnvPayload.Data.Env["CONTAINER_NAME"] != "edge-proxy-1" || containerEnvPayload.Data.Env["PORT"] == "" {
		t.Fatalf("expected derived container environment payload")
	}
}
