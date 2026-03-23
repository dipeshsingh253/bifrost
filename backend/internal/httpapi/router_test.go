package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/dipesh/bifrost/backend/internal/config"
	"github.com/dipesh/bifrost/backend/internal/domain"
	"github.com/dipesh/bifrost/backend/internal/seed"
	"github.com/dipesh/bifrost/backend/internal/store"
)

func selfEnrollAgent(t *testing.T, router http.Handler, bootstrapKey, agentID, serverID string) string {
	t.Helper()

	requestBody := `{"agent_id":"` + agentID + `","server_id":"` + serverID + `"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/agents/enroll", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Agent-Key", bootstrapKey)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected agent enroll status 200, got %d", response.Code)
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			APIKey string `json:"api_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode agent enroll: %v", err)
	}
	if payload.Data.APIKey == "" || payload.Data.APIKey == bootstrapKey {
		t.Fatalf("expected enroll to rotate the bootstrap credential")
	}

	return payload.Data.APIKey
}

func requireErrorCode(t *testing.T, response *httptest.ResponseRecorder, expected string) {
	t.Helper()

	var payload struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload.Success {
		t.Fatalf("expected error response, got success payload")
	}
	if payload.Error.Code != expected {
		t.Fatalf("expected error code %q, got %q", expected, payload.Error.Code)
	}
}

func TestBootstrapStatusReportsWhenSetupIsRequired(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(store.SeedData{}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/bootstrap/status", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			NeedsBootstrap bool `json:"needs_bootstrap"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode bootstrap status: %v", err)
	}
	if !payload.Success || !payload.Data.NeedsBootstrap {
		t.Fatalf("expected bootstrap to be required for an empty store")
	}
}

func TestInstallAgentScriptIsPublicAndContainsExpectedSetupSteps(t *testing.T) {
	router := NewRouter(config.Config{
		AgentDockerImage: "bifrost-agent:latest",
	}, store.NewMemoryStore(store.SeedData{}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/install/agent.sh", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected install script status 200, got %d", response.Code)
	}

	body := response.Body.String()
	if !strings.Contains(body, "#!/bin/sh") {
		t.Fatalf("expected shell script header, got %q", body)
	}
	if !strings.Contains(body, `AGENT_BINARY_URL="${BIFROST_AGENT_BINARY_URL:-http://example.com/api/v1/install/agent}"`) {
		t.Fatalf("expected install script to default the binary download url, got %q", body)
	}
	if !strings.Contains(body, "docker cp \"$tmp_container:/usr/local/bin/bifrost-agent\" \"$BINARY_PATH\"") {
		t.Fatalf("expected install script to extract the binary from the docker image")
	}
	if !strings.Contains(body, "if ! docker image inspect \"$AGENT_IMAGE\" >/dev/null 2>&1; then") {
		t.Fatalf("expected install script to reuse a local image before pulling")
	}
	if !strings.Contains(body, "if ! download_binary; then") {
		t.Fatalf("expected install script to prefer downloading the current agent binary")
	}
	if !strings.Contains(body, "EnvironmentFile=$ENV_FILE") || !strings.Contains(body, "ExecStart=$BINARY_PATH") {
		t.Fatalf("expected install script to install a systemd unit, got %q", body)
	}
}

func TestInstallAgentBinaryServesConfiguredBinary(t *testing.T) {
	tempDir := t.TempDir()
	binaryPath := tempDir + "/bifrost-agent"
	if err := os.WriteFile(binaryPath, []byte("binary-payload"), 0o755); err != nil {
		t.Fatalf("write temp binary: %v", err)
	}

	router := NewRouter(config.Config{
		AgentBinaryPath: binaryPath,
	}, store.NewMemoryStore(store.SeedData{}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/install/agent", nil)
	request.Host = "example.com"
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected install binary status 200, got %d", response.Code)
	}
	if response.Body.String() != "binary-payload" {
		t.Fatalf("expected install binary body to match configured file, got %q", response.Body.String())
	}
}

func TestBootstrapLoginSessionAndLogoutFlow(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(store.SeedData{}))

	bootstrapBody := `{"tenant_name":"Bifrost Local","name":"Admin User","email":"admin@example.com","password":"password123"}`
	bootstrapRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/bootstrap", strings.NewReader(bootstrapBody))
	bootstrapRequest.Header.Set("Content-Type", "application/json")
	bootstrapResponse := httptest.NewRecorder()
	router.ServeHTTP(bootstrapResponse, bootstrapRequest)

	if bootstrapResponse.Code != http.StatusCreated {
		t.Fatalf("expected bootstrap status 201, got %d", bootstrapResponse.Code)
	}

	cookies := bootstrapResponse.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected bootstrap to set a session cookie")
	}

	sessionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	sessionRequest.AddCookie(cookies[0])
	sessionResponse := httptest.NewRecorder()
	router.ServeHTTP(sessionResponse, sessionRequest)

	if sessionResponse.Code != http.StatusOK {
		t.Fatalf("expected session status 200 after bootstrap, got %d", sessionResponse.Code)
	}

	var sessionPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		} `json:"data"`
	}
	if err := json.Unmarshal(sessionResponse.Body.Bytes(), &sessionPayload); err != nil {
		t.Fatalf("decode session payload: %v", err)
	}
	if sessionPayload.Data.Email != "admin@example.com" || sessionPayload.Data.Role != "admin" {
		t.Fatalf("expected bootstrapped admin session, got %+v", sessionPayload.Data)
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutRequest.AddCookie(cookies[0])
	logoutResponse := httptest.NewRecorder()
	router.ServeHTTP(logoutResponse, logoutRequest)

	if logoutResponse.Code != http.StatusOK {
		t.Fatalf("expected logout status 200, got %d", logoutResponse.Code)
	}

	postLogoutRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	postLogoutRequest.AddCookie(cookies[0])
	postLogoutResponse := httptest.NewRecorder()
	router.ServeHTTP(postLogoutResponse, postLogoutRequest)

	if postLogoutResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected session status 401 after logout, got %d", postLogoutResponse.Code)
	}
}

func TestLoginSetsCookieForExistingUser(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	loginBody := `{"email":"owner@bifrost.local","password":"bifrost123"}`
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	router.ServeHTTP(loginResponse, loginRequest)

	if loginResponse.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", loginResponse.Code)
	}

	cookies := loginResponse.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected login to set a session cookie")
	}

	sessionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	sessionRequest.AddCookie(cookies[0])
	sessionResponse := httptest.NewRecorder()
	router.ServeHTTP(sessionResponse, sessionRequest)

	if sessionResponse.Code != http.StatusOK {
		t.Fatalf("expected session status 200 after login, got %d", sessionResponse.Code)
	}
}

func TestAdminSummaryAllowsAdminAndBlocksViewer(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seedWithViewer()))

	adminRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/summary", nil)
	adminRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	adminResponse := httptest.NewRecorder()
	router.ServeHTTP(adminResponse, adminRequest)

	if adminResponse.Code != http.StatusOK {
		t.Fatalf("expected admin summary status 200 for admin, got %d", adminResponse.Code)
	}

	var adminPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Tenant struct {
				AdminCount  int `json:"admin_count"`
				ViewerCount int `json:"viewer_count"`
			} `json:"tenant"`
		} `json:"data"`
	}
	if err := json.Unmarshal(adminResponse.Body.Bytes(), &adminPayload); err != nil {
		t.Fatalf("decode admin summary: %v", err)
	}
	if adminPayload.Data.Tenant.AdminCount == 0 || adminPayload.Data.Tenant.ViewerCount == 0 {
		t.Fatalf("expected admin summary to include role counts")
	}

	viewerRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/summary", nil)
	viewerRequest.Header.Set("Authorization", "Bearer viewer-token")
	viewerResponse := httptest.NewRecorder()
	router.ServeHTTP(viewerResponse, viewerRequest)

	if viewerResponse.Code != http.StatusForbidden {
		t.Fatalf("expected admin summary status 403 for viewer, got %d", viewerResponse.Code)
	}
}

func TestViewerRetainsMonitoringReadAccess(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seedWithViewer()))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	request.Header.Set("Authorization", "Bearer viewer-token")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected viewer server list status 200, got %d", response.Code)
	}
}

func TestAdminCanCreateSystemOnboardingWithOpaqueAgentCredentials(t *testing.T) {
	router := NewRouter(config.Config{
		AgentBackendURL: "https://bifrost.example.com",
	}, store.NewMemoryStore(seed.Data()))

	createBody := `{"name":"Production VPS","description":"Primary app node"}`
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/systems", strings.NewReader(createBody))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create system onboarding status 201, got %d", createResponse.Code)
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			ID                    string `json:"id"`
			ServerID              string `json:"server_id"`
			AgentID               string `json:"agent_id"`
			Name                  string `json:"name"`
			Description           string `json:"description"`
			Status                string `json:"status"`
			APIKey                string `json:"api_key"`
			EnrollmentToken       string `json:"enrollment_token"`
			BackendURL            string `json:"backend_url"`
			InstallScriptURL      string `json:"install_script_url"`
			DockerRunCommand      string `json:"docker_run_command"`
			SystemdInstallCommand string `json:"systemd_install_command"`
			ConfigYAML            string `json:"config_yaml"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode create system onboarding: %v", err)
	}

	if payload.Data.ID == "" || payload.Data.ServerID == "" || payload.Data.AgentID == "" || payload.Data.APIKey == "" {
		t.Fatalf("expected onboarding response to include identifiers and api key")
	}
	if !looksLikeUUID(payload.Data.ID) || !looksLikeUUID(payload.Data.ServerID) || !looksLikeUUID(payload.Data.AgentID) {
		t.Fatalf("expected onboarding response to include uuid identifiers, got %+v", payload.Data)
	}
	if payload.Data.APIKey == payload.Data.AgentID+"-key" {
		t.Fatalf("expected api key to be opaque instead of derived from the agent id")
	}
	if payload.Data.Status != "awaiting_connection" {
		t.Fatalf("expected awaiting_connection status, got %q", payload.Data.Status)
	}
	if payload.Data.Name != "Production VPS" || payload.Data.Description != "Primary app node" {
		t.Fatalf("expected onboarding metadata to round-trip, got %+v", payload.Data)
	}
	if payload.Data.BackendURL != "https://bifrost.example.com" {
		t.Fatalf("expected configured backend url, got %q", payload.Data.BackendURL)
	}
	if !strings.Contains(payload.Data.ConfigYAML, `server_id: "`+payload.Data.ServerID+`"`) {
		t.Fatalf("expected config yaml to include server id")
	}
	if !strings.Contains(payload.Data.ConfigYAML, `agent_id: "`+payload.Data.AgentID+`"`) {
		t.Fatalf("expected config yaml to include agent id")
	}
	if payload.Data.EnrollmentToken == "" || payload.Data.EnrollmentToken != payload.Data.APIKey {
		t.Fatalf("expected onboarding response to expose the bootstrap enrollment token")
	}
	if !strings.Contains(payload.Data.ConfigYAML, `enrollment_token: "`+payload.Data.EnrollmentToken+`"`) {
		t.Fatalf("expected config yaml to include the enrollment token")
	}
	if payload.Data.InstallScriptURL != "https://bifrost.example.com/api/v1/install/agent.sh" {
		t.Fatalf("expected install script url, got %q", payload.Data.InstallScriptURL)
	}
	if !strings.Contains(payload.Data.DockerRunCommand, "BIFROST_AGENT_ID='"+payload.Data.AgentID+"'") {
		t.Fatalf("expected docker command to include the agent id env, got %q", payload.Data.DockerRunCommand)
	}
	if !strings.Contains(payload.Data.SystemdInstallCommand, "curl -fsSL 'https://bifrost.example.com/api/v1/install/agent.sh'") {
		t.Fatalf("expected systemd install command to use install script, got %q", payload.Data.SystemdInstallCommand)
	}
}

func TestViewerCannotCreateSystemOnboarding(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seedWithViewer()))

	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/systems", strings.NewReader(`{"name":"Blocked"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Authorization", "Bearer viewer-token")
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusForbidden {
		t.Fatalf("expected viewer system onboarding status 403, got %d", createResponse.Code)
	}
}

func TestSystemOnboardingTransitionsToConnectedAfterFirstIngest(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/systems", strings.NewReader(`{"name":"Fresh VPS","description":"Waiting node"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create system onboarding status 201, got %d", createResponse.Code)
	}

	var createPayload struct {
		Success bool `json:"success"`
		Data    struct {
			ID       string `json:"id"`
			ServerID string `json:"server_id"`
			AgentID  string `json:"agent_id"`
			APIKey   string `json:"api_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create system onboarding: %v", err)
	}

	preEnrollRequest := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/snapshot", strings.NewReader(`{
		"agent_id":"`+createPayload.Data.AgentID+`",
		"server":{"id":"`+createPayload.Data.ServerID+`","name":"Fresh VPS","hostname":"fresh-vps","status":"up","services":[],"collected_at":"2026-03-23T01:55:00Z"},
		"metrics":[],
		"logs":[]
	}`))
	preEnrollRequest.Header.Set("Content-Type", "application/json")
	preEnrollRequest.Header.Set("X-Agent-Key", createPayload.Data.APIKey)
	preEnrollResponse := httptest.NewRecorder()
	router.ServeHTTP(preEnrollResponse, preEnrollRequest)

	if preEnrollResponse.Code != http.StatusConflict {
		t.Fatalf("expected pending bootstrap token ingest to be rejected with 409, got %d", preEnrollResponse.Code)
	}

	enrolledAPIKey := selfEnrollAgent(t, router, createPayload.Data.APIKey, createPayload.Data.AgentID, createPayload.Data.ServerID)

	collectedAt := "2026-03-23T02:00:00Z"
	ingestBody := `{
		"agent_id":"` + createPayload.Data.AgentID + `",
		"server":{
			"id":"` + createPayload.Data.ServerID + `",
			"name":"Fresh VPS",
			"hostname":"fresh-vps",
			"public_ip":"198.51.100.10",
			"agent_version":"0.1.0",
			"status":"up",
			"uptime_seconds":123,
			"cpu_usage_pct":10,
			"memory_usage_pct":20,
			"disk_usage_pct":30,
			"network_rx_mb":1,
			"network_tx_mb":2,
			"load_average":"0.10 0.20 0.30",
			"os":"Ubuntu",
			"kernel":"6.8.0",
			"cpu_model":"AMD EPYC",
			"cpu_cores":2,
			"cpu_threads":4,
			"total_memory_gb":4,
			"total_disk_gb":80,
			"services":[],
			"collected_at":"` + collectedAt + `"
		},
		"metrics":[],
		"logs":[]
	}`

	ingestRequest := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/snapshot", strings.NewReader(ingestBody))
	ingestRequest.Header.Set("Content-Type", "application/json")
	ingestRequest.Header.Set("X-Agent-Key", enrolledAPIKey)
	ingestResponse := httptest.NewRecorder()
	router.ServeHTTP(ingestResponse, ingestRequest)

	if ingestResponse.Code != http.StatusAccepted {
		t.Fatalf("expected ingest status 202, got %d", ingestResponse.Code)
	}

	statusRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/systems/"+createPayload.Data.ID, nil)
	statusRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	statusResponse := httptest.NewRecorder()
	router.ServeHTTP(statusResponse, statusRequest)

	if statusResponse.Code != http.StatusOK {
		t.Fatalf("expected system onboarding detail status 200, got %d", statusResponse.Code)
	}

	var statusPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Status      string  `json:"status"`
			ConnectedAt *string `json:"connected_at"`
			ServerID    string  `json:"server_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(statusResponse.Body.Bytes(), &statusPayload); err != nil {
		t.Fatalf("decode system onboarding detail: %v", err)
	}
	if statusPayload.Data.Status != "connected" {
		t.Fatalf("expected connected status after first ingest, got %q", statusPayload.Data.Status)
	}
	if statusPayload.Data.ConnectedAt == nil || *statusPayload.Data.ConnectedAt != collectedAt {
		t.Fatalf("expected connected_at to match first ingest time, got %+v", statusPayload.Data.ConnectedAt)
	}
	if statusPayload.Data.ServerID != createPayload.Data.ServerID {
		t.Fatalf("expected system onboarding to keep server id %q, got %q", createPayload.Data.ServerID, statusPayload.Data.ServerID)
	}

	secondIngestRequest := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/snapshot", strings.NewReader(strings.Replace(ingestBody, collectedAt, "2026-03-23T03:00:00Z", 1)))
	secondIngestRequest.Header.Set("Content-Type", "application/json")
	secondIngestRequest.Header.Set("X-Agent-Key", enrolledAPIKey)
	secondIngestResponse := httptest.NewRecorder()
	router.ServeHTTP(secondIngestResponse, secondIngestRequest)

	if secondIngestResponse.Code != http.StatusAccepted {
		t.Fatalf("expected second ingest status 202, got %d", secondIngestResponse.Code)
	}

	recheckRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/systems/"+createPayload.Data.ID, nil)
	recheckRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	recheckResponse := httptest.NewRecorder()
	router.ServeHTTP(recheckResponse, recheckRequest)

	if recheckResponse.Code != http.StatusOK {
		t.Fatalf("expected second system onboarding detail status 200, got %d", recheckResponse.Code)
	}

	var recheckPayload struct {
		Success bool `json:"success"`
		Data    struct {
			ConnectedAt *string `json:"connected_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recheckResponse.Body.Bytes(), &recheckPayload); err != nil {
		t.Fatalf("decode repeated system onboarding detail: %v", err)
	}
	if recheckPayload.Data.ConnectedAt == nil || *recheckPayload.Data.ConnectedAt != collectedAt {
		t.Fatalf("expected connected_at to remain pinned to first ingest, got %+v", recheckPayload.Data.ConnectedAt)
	}
}

func TestAdminCanListPendingAndConnectedSystemOnboardings(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	type createdSystem struct {
		ID       string
		ServerID string
		AgentID  string
		APIKey   string
	}

	createSystem := func(name string) createdSystem {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/systems", strings.NewReader(`{"name":"`+name+`"}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer demo-owner-token")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusCreated {
			t.Fatalf("expected create system onboarding status 201, got %d", response.Code)
		}

		var payload struct {
			Success bool `json:"success"`
			Data    struct {
				ID       string `json:"id"`
				ServerID string `json:"server_id"`
				AgentID  string `json:"agent_id"`
				APIKey   string `json:"api_key"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode create system onboarding: %v", err)
		}
		return createdSystem(payload.Data)
	}

	pending := createSystem("Pending VPS")
	connected := createSystem("Connected VPS")
	connectedAPIKey := selfEnrollAgent(t, router, connected.APIKey, connected.AgentID, connected.ServerID)

	ingestBody := `{
		"agent_id":"` + connected.AgentID + `",
		"server":{
			"id":"` + connected.ServerID + `",
			"name":"Connected VPS",
			"hostname":"connected-vps",
			"public_ip":"198.51.100.20",
			"agent_version":"0.1.0",
			"status":"up",
			"uptime_seconds":1,
			"cpu_usage_pct":1,
			"memory_usage_pct":1,
			"disk_usage_pct":1,
			"network_rx_mb":1,
			"network_tx_mb":1,
			"load_average":"0.01 0.01 0.01",
			"os":"Ubuntu",
			"kernel":"6.8.0",
			"cpu_model":"AMD EPYC",
			"cpu_cores":2,
			"cpu_threads":4,
			"total_memory_gb":4,
			"total_disk_gb":80,
			"services":[],
			"collected_at":"2026-03-23T04:00:00Z"
		},
		"metrics":[],
		"logs":[]
	}`
	ingestRequest := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/snapshot", strings.NewReader(ingestBody))
	ingestRequest.Header.Set("Content-Type", "application/json")
	ingestRequest.Header.Set("X-Agent-Key", connectedAPIKey)
	ingestResponse := httptest.NewRecorder()
	router.ServeHTTP(ingestResponse, ingestRequest)

	if ingestResponse.Code != http.StatusAccepted {
		t.Fatalf("expected ingest status 202, got %d", ingestResponse.Code)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/systems", nil)
	listRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)

	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected list system onboardings status 200, got %d", listResponse.Code)
	}

	var listPayload struct {
		Success bool `json:"success"`
		Data    []struct {
			ID          string  `json:"id"`
			ServerID    string  `json:"server_id"`
			Status      string  `json:"status"`
			Name        string  `json:"name"`
			ConnectedAt *string `json:"connected_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode list system onboardings: %v", err)
	}
	if len(listPayload.Data) != 2 {
		t.Fatalf("expected exactly two system onboarding entries, got %d", len(listPayload.Data))
	}
	if listPayload.Data[0].ID != pending.ID || listPayload.Data[0].Status != "awaiting_connection" {
		t.Fatalf("expected pending onboarding to be listed first, got %+v", listPayload.Data[0])
	}
	if listPayload.Data[1].ID != connected.ID || listPayload.Data[1].Status != "connected" {
		t.Fatalf("expected connected onboarding to be listed after pending, got %+v", listPayload.Data[1])
	}
	if listPayload.Data[1].ConnectedAt == nil || *listPayload.Data[1].ConnectedAt != "2026-03-23T04:00:00Z" {
		t.Fatalf("expected connected onboarding to expose its connection timestamp, got %+v", listPayload.Data[1].ConnectedAt)
	}
}

func TestViewerCannotListSystemOnboardings(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seedWithViewer()))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/systems", nil)
	request.Header.Set("Authorization", "Bearer viewer-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected viewer list system onboardings status 403, got %d", response.Code)
	}
}

func TestAdminCanCancelPendingSystemOnboarding(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/systems", strings.NewReader(`{"name":"Leaked VPS"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create system onboarding status 201, got %d", createResponse.Code)
	}

	var createPayload struct {
		Success bool `json:"success"`
		Data    struct {
			ID      string `json:"id"`
			AgentID string `json:"agent_id"`
			APIKey  string `json:"api_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create system onboarding: %v", err)
	}

	cancelRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/systems/"+createPayload.Data.ID+"/cancel", nil)
	cancelRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	cancelResponse := httptest.NewRecorder()
	router.ServeHTTP(cancelResponse, cancelRequest)

	if cancelResponse.Code != http.StatusOK {
		t.Fatalf("expected cancel system onboarding status 200, got %d", cancelResponse.Code)
	}

	detailRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/systems/"+createPayload.Data.ID, nil)
	detailRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	detailResponse := httptest.NewRecorder()
	router.ServeHTTP(detailResponse, detailRequest)

	if detailResponse.Code != http.StatusNotFound {
		t.Fatalf("expected cancelled system onboarding detail status 404, got %d", detailResponse.Code)
	}

	ingestBody := `{
		"agent_id":"` + createPayload.Data.AgentID + `",
		"server":{"id":"8f6a3306-18dc-4a1f-9720-5063f06af4a5","name":"Leaked VPS","hostname":"leaked-vps","status":"up","services":[],"collected_at":"2026-03-23T05:00:00Z"},
		"metrics":[],
		"logs":[]
	}`
	ingestRequest := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/snapshot", strings.NewReader(ingestBody))
	ingestRequest.Header.Set("Content-Type", "application/json")
	ingestRequest.Header.Set("X-Agent-Key", createPayload.Data.APIKey)
	ingestResponse := httptest.NewRecorder()
	router.ServeHTTP(ingestResponse, ingestRequest)

	if ingestResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected cancelled api key to be rejected, got %d", ingestResponse.Code)
	}
}

func TestPendingBootstrapTokenMustSelfEnrollBeforeSnapshotIngest(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/systems", strings.NewReader(`{"name":"Bootstrap VPS"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create system onboarding status 201, got %d", createResponse.Code)
	}

	var createPayload struct {
		Success bool `json:"success"`
		Data    struct {
			ServerID string `json:"server_id"`
			AgentID  string `json:"agent_id"`
			APIKey   string `json:"api_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create system onboarding: %v", err)
	}

	ingestBody := `{
		"agent_id":"` + createPayload.Data.AgentID + `",
		"server":{"id":"` + createPayload.Data.ServerID + `","name":"Bootstrap VPS","hostname":"bootstrap-vps","status":"up","services":[],"collected_at":"2026-03-23T05:30:00Z"},
		"metrics":[],
		"logs":[]
	}`
	preEnrollRequest := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/snapshot", strings.NewReader(ingestBody))
	preEnrollRequest.Header.Set("Content-Type", "application/json")
	preEnrollRequest.Header.Set("X-Agent-Key", createPayload.Data.APIKey)
	preEnrollResponse := httptest.NewRecorder()
	router.ServeHTTP(preEnrollResponse, preEnrollRequest)

	if preEnrollResponse.Code != http.StatusConflict {
		t.Fatalf("expected bootstrap token ingest status 409, got %d", preEnrollResponse.Code)
	}

	realAPIKey := selfEnrollAgent(t, router, createPayload.Data.APIKey, createPayload.Data.AgentID, createPayload.Data.ServerID)
	postEnrollRequest := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/snapshot", strings.NewReader(ingestBody))
	postEnrollRequest.Header.Set("Content-Type", "application/json")
	postEnrollRequest.Header.Set("X-Agent-Key", realAPIKey)
	postEnrollResponse := httptest.NewRecorder()
	router.ServeHTTP(postEnrollResponse, postEnrollRequest)

	if postEnrollResponse.Code != http.StatusAccepted {
		t.Fatalf("expected enrolled api key ingest status 202, got %d", postEnrollResponse.Code)
	}
}

func TestAdminCanReissuePendingSystemOnboardingCredentials(t *testing.T) {
	router := NewRouter(config.Config{
		AgentBackendURL: "https://bifrost.example.com",
	}, store.NewMemoryStore(seed.Data()))

	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/systems", strings.NewReader(`{"name":"Rotated VPS"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create system onboarding status 201, got %d", createResponse.Code)
	}

	var createPayload struct {
		Success bool `json:"success"`
		Data    struct {
			ID       string `json:"id"`
			ServerID string `json:"server_id"`
			AgentID  string `json:"agent_id"`
			APIKey   string `json:"api_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create system onboarding: %v", err)
	}

	reissueRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/systems/"+createPayload.Data.ID+"/reissue", nil)
	reissueRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	reissueResponse := httptest.NewRecorder()
	router.ServeHTTP(reissueResponse, reissueRequest)

	if reissueResponse.Code != http.StatusOK {
		t.Fatalf("expected reissue system onboarding status 200, got %d", reissueResponse.Code)
	}

	var reissuePayload struct {
		Success bool `json:"success"`
		Data    struct {
			ID                    string `json:"id"`
			ServerID              string `json:"server_id"`
			AgentID               string `json:"agent_id"`
			Status                string `json:"status"`
			APIKey                string `json:"api_key"`
			EnrollmentToken       string `json:"enrollment_token"`
			SystemdInstallCommand string `json:"systemd_install_command"`
			ConfigYAML            string `json:"config_yaml"`
		} `json:"data"`
	}
	if err := json.Unmarshal(reissueResponse.Body.Bytes(), &reissuePayload); err != nil {
		t.Fatalf("decode reissue system onboarding: %v", err)
	}

	if reissuePayload.Data.ID != createPayload.Data.ID || reissuePayload.Data.ServerID != createPayload.Data.ServerID || reissuePayload.Data.AgentID != createPayload.Data.AgentID {
		t.Fatalf("expected reissue to preserve onboarding identity, got %+v", reissuePayload.Data)
	}
	if reissuePayload.Data.Status != "awaiting_connection" {
		t.Fatalf("expected reissued onboarding to remain pending, got %q", reissuePayload.Data.Status)
	}
	if reissuePayload.Data.APIKey == "" || reissuePayload.Data.APIKey == createPayload.Data.APIKey {
		t.Fatalf("expected reissue to return a rotated bootstrap credential")
	}
	if reissuePayload.Data.EnrollmentToken == "" || reissuePayload.Data.EnrollmentToken != reissuePayload.Data.APIKey {
		t.Fatalf("expected reissue to expose the rotated enrollment token")
	}
	if !strings.Contains(reissuePayload.Data.ConfigYAML, `enrollment_token: "`+reissuePayload.Data.EnrollmentToken+`"`) {
		t.Fatalf("expected reissued config to include the rotated enrollment token")
	}
	if !strings.Contains(reissuePayload.Data.SystemdInstallCommand, "BIFROST_ENROLLMENT_TOKEN='"+reissuePayload.Data.EnrollmentToken+"'") {
		t.Fatalf("expected reissued quick-install command to include the rotated enrollment token")
	}

	oldKeyIngest := `{
		"agent_id":"` + createPayload.Data.AgentID + `",
		"server":{"id":"` + createPayload.Data.ServerID + `","name":"Rotated VPS","hostname":"rotated-vps","status":"up","services":[],"collected_at":"2026-03-23T06:00:00Z"},
		"metrics":[],
		"logs":[]
	}`
	oldKeyRequest := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/snapshot", strings.NewReader(oldKeyIngest))
	oldKeyRequest.Header.Set("Content-Type", "application/json")
	oldKeyRequest.Header.Set("X-Agent-Key", createPayload.Data.APIKey)
	oldKeyResponse := httptest.NewRecorder()
	router.ServeHTTP(oldKeyResponse, oldKeyRequest)

	if oldKeyResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected old api key to be rejected after reissue, got %d", oldKeyResponse.Code)
	}

	enrolledAPIKey := selfEnrollAgent(t, router, reissuePayload.Data.APIKey, reissuePayload.Data.AgentID, reissuePayload.Data.ServerID)
	newKeyRequest := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/snapshot", strings.NewReader(oldKeyIngest))
	newKeyRequest.Header.Set("Content-Type", "application/json")
	newKeyRequest.Header.Set("X-Agent-Key", enrolledAPIKey)
	newKeyResponse := httptest.NewRecorder()
	router.ServeHTTP(newKeyResponse, newKeyRequest)

	if newKeyResponse.Code != http.StatusAccepted {
		t.Fatalf("expected rotated api key ingest status 202, got %d", newKeyResponse.Code)
	}
}

func TestAdminCannotCancelOrReissueConnectedSystemOnboarding(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/systems", strings.NewReader(`{"name":"Connected VPS"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create system onboarding status 201, got %d", createResponse.Code)
	}

	var createPayload struct {
		Success bool `json:"success"`
		Data    struct {
			ID       string `json:"id"`
			ServerID string `json:"server_id"`
			AgentID  string `json:"agent_id"`
			APIKey   string `json:"api_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create system onboarding: %v", err)
	}
	connectedAPIKey := selfEnrollAgent(t, router, createPayload.Data.APIKey, createPayload.Data.AgentID, createPayload.Data.ServerID)

	ingestBody := `{
		"agent_id":"` + createPayload.Data.AgentID + `",
		"server":{"id":"` + createPayload.Data.ServerID + `","name":"Connected VPS","hostname":"connected-vps","status":"up","services":[],"collected_at":"2026-03-23T07:00:00Z"},
		"metrics":[],
		"logs":[]
	}`
	ingestRequest := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/snapshot", strings.NewReader(ingestBody))
	ingestRequest.Header.Set("Content-Type", "application/json")
	ingestRequest.Header.Set("X-Agent-Key", connectedAPIKey)
	ingestResponse := httptest.NewRecorder()
	router.ServeHTTP(ingestResponse, ingestRequest)

	if ingestResponse.Code != http.StatusAccepted {
		t.Fatalf("expected ingest status 202, got %d", ingestResponse.Code)
	}

	for _, path := range []string{
		"/api/v1/admin/systems/" + createPayload.Data.ID + "/cancel",
		"/api/v1/admin/systems/" + createPayload.Data.ID + "/reissue",
	} {
		request := httptest.NewRequest(http.MethodPost, path, nil)
		request.Header.Set("Authorization", "Bearer demo-owner-token")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusConflict {
			t.Fatalf("expected connected system onboarding action %s to return 409, got %d", path, response.Code)
		}
	}
}

func TestViewerCannotCancelOrReissueSystemOnboarding(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seedWithViewer()))

	for _, path := range []string{
		"/api/v1/admin/systems/8a7d41f0-3ae7-4336-88ef-49f9fb17053e/cancel",
		"/api/v1/admin/systems/8a7d41f0-3ae7-4336-88ef-49f9fb17053e/reissue",
	} {
		request := httptest.NewRequest(http.MethodPost, path, nil)
		request.Header.Set("Authorization", "Bearer viewer-token")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusForbidden {
			t.Fatalf("expected viewer onboarding action %s to return 403, got %d", path, response.Code)
		}
	}
}

func TestAdminCanCreateAcceptDisableAndDeleteViewerAccess(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	createBody := `{"email":"new-viewer@bifrost.local"}`
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/invites", strings.NewReader(createBody))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create invite status 201, got %d", createResponse.Code)
	}

	var invitePayload struct {
		Success bool `json:"success"`
		Data    struct {
			ID          string `json:"id"`
			Email       string `json:"email"`
			InviteToken string `json:"invite_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &invitePayload); err != nil {
		t.Fatalf("decode create invite: %v", err)
	}
	if invitePayload.Data.ID == "" || invitePayload.Data.InviteToken == "" {
		t.Fatalf("expected created invite to include id and token")
	}

	detailRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/invites/"+invitePayload.Data.InviteToken, nil)
	detailResponse := httptest.NewRecorder()
	router.ServeHTTP(detailResponse, detailRequest)

	if detailResponse.Code != http.StatusOK {
		t.Fatalf("expected invite detail status 200, got %d", detailResponse.Code)
	}

	acceptBody := `{"token":"` + invitePayload.Data.InviteToken + `","name":"Viewer User","password":"viewer123"}`
	acceptRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/invites/accept", strings.NewReader(acceptBody))
	acceptRequest.Header.Set("Content-Type", "application/json")
	acceptResponse := httptest.NewRecorder()
	router.ServeHTTP(acceptResponse, acceptRequest)

	if acceptResponse.Code != http.StatusCreated {
		t.Fatalf("expected invite accept status 201, got %d", acceptResponse.Code)
	}

	var acceptPayload struct {
		Success bool `json:"success"`
		Data    struct {
			User struct {
				ID    string `json:"id"`
				Email string `json:"email"`
				Role  string `json:"role"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(acceptResponse.Body.Bytes(), &acceptPayload); err != nil {
		t.Fatalf("decode accepted invite: %v", err)
	}
	if acceptPayload.Data.User.ID == "" || acceptPayload.Data.User.Role != "viewer" {
		t.Fatalf("expected accepted invite to create a viewer user")
	}

	acceptCookies := acceptResponse.Result().Cookies()
	if len(acceptCookies) == 0 {
		t.Fatalf("expected accepted invite to set a session cookie")
	}

	viewerServersRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	viewerServersRequest.AddCookie(acceptCookies[0])
	viewerServersResponse := httptest.NewRecorder()
	router.ServeHTTP(viewerServersResponse, viewerServersRequest)

	if viewerServersResponse.Code != http.StatusOK {
		t.Fatalf("expected accepted viewer to read servers, got %d", viewerServersResponse.Code)
	}

	disableRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/viewers/"+acceptPayload.Data.User.ID+"/disable", nil)
	disableRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	disableResponse := httptest.NewRecorder()
	router.ServeHTTP(disableResponse, disableRequest)

	if disableResponse.Code != http.StatusOK {
		t.Fatalf("expected disable viewer status 200, got %d", disableResponse.Code)
	}

	viewerAfterDisableRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	viewerAfterDisableRequest.AddCookie(acceptCookies[0])
	viewerAfterDisableResponse := httptest.NewRecorder()
	router.ServeHTTP(viewerAfterDisableResponse, viewerAfterDisableRequest)

	if viewerAfterDisableResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected disabled viewer access to be blocked, got %d", viewerAfterDisableResponse.Code)
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/viewers/"+acceptPayload.Data.User.ID, nil)
	deleteRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, deleteRequest)

	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("expected delete viewer status 200, got %d", deleteResponse.Code)
	}
}

func TestRevokedInviteCannotBeAccepted(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	createBody := `{"email":"revoked-viewer@bifrost.local"}`
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/invites", strings.NewReader(createBody))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create invite status 201, got %d", createResponse.Code)
	}

	var invitePayload struct {
		Success bool `json:"success"`
		Data    struct {
			ID          string `json:"id"`
			InviteToken string `json:"invite_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &invitePayload); err != nil {
		t.Fatalf("decode create invite: %v", err)
	}

	revokeRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/invites/"+invitePayload.Data.ID+"/revoke", nil)
	revokeRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	revokeResponse := httptest.NewRecorder()
	router.ServeHTTP(revokeResponse, revokeRequest)

	if revokeResponse.Code != http.StatusOK {
		t.Fatalf("expected revoke invite status 200, got %d", revokeResponse.Code)
	}

	acceptBody := `{"token":"` + invitePayload.Data.InviteToken + `","name":"Viewer User","password":"viewer123"}`
	acceptRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/invites/accept", strings.NewReader(acceptBody))
	acceptRequest.Header.Set("Content-Type", "application/json")
	acceptResponse := httptest.NewRecorder()
	router.ServeHTTP(acceptResponse, acceptRequest)

	if acceptResponse.Code != http.StatusConflict {
		t.Fatalf("expected revoked invite accept status 409, got %d", acceptResponse.Code)
	}
}

func TestServerDetailReturnsFrontendBundleShape(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer, nil)
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
				Containers       []struct {
					ID string `json:"id"`
				} `json:"containers"`
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

	if payload.Data.Server.ID != seed.ServerIDDevServer {
		t.Fatalf("expected server id %s, got %q", seed.ServerIDDevServer, payload.Data.Server.ID)
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
	if payload.Data.Services[0].ID != seed.ServiceIDEdgeProxy && payload.Data.Services[0].ID != seed.ServiceIDSearchStack && payload.Data.Services[0].ID != seed.ServiceIDServiceA {
		t.Fatalf("expected bundled service id to use canonical id, got %q", payload.Data.Services[0].ID)
	}
	if len(payload.Data.Services[0].Containers) == 0 || !looksLikeUUID(payload.Data.Services[0].Containers[0].ID) {
		t.Fatalf("expected bundled container to include a uuid id, got %+v", payload.Data.Services[0].Containers)
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
	if _, ok := payload.Data.ContainerMetrics.CPU[0][seed.ContainerIDAPI1]; !ok {
		t.Fatalf("expected container-id keyed series in cpu metric point")
	}
}

func TestMonitoringRoutesResolveCanonicalIDsAndRejectLegacyReadableIDs(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	serverRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer, nil)
	serverRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	serverResponse := httptest.NewRecorder()
	router.ServeHTTP(serverResponse, serverRequest)

	if serverResponse.Code != http.StatusOK {
		t.Fatalf("expected server detail status 200 for canonical id route, got %d", serverResponse.Code)
	}

	legacyServerRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-dev-server", nil)
	legacyServerRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	legacyServerResponse := httptest.NewRecorder()
	router.ServeHTTP(legacyServerResponse, legacyServerRequest)

	if legacyServerResponse.Code != http.StatusNotFound {
		t.Fatalf("expected legacy server id route to return 404, got %d", legacyServerResponse.Code)
	}
	requireErrorCode(t, legacyServerResponse, "SERVER_NOT_FOUND")

	projectRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/projects/"+seed.ServiceIDServiceA, nil)
	projectRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	projectResponse := httptest.NewRecorder()
	router.ServeHTTP(projectResponse, projectRequest)

	if projectResponse.Code != http.StatusOK {
		t.Fatalf("expected project detail status 200 for canonical id route, got %d", projectResponse.Code)
	}

	legacyProjectRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/projects/svc-service-a", nil)
	legacyProjectRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	legacyProjectResponse := httptest.NewRecorder()
	router.ServeHTTP(legacyProjectResponse, legacyProjectRequest)

	if legacyProjectResponse.Code != http.StatusNotFound {
		t.Fatalf("expected legacy project id route to return 404, got %d", legacyProjectResponse.Code)
	}
	requireErrorCode(t, legacyProjectResponse, "PROJECT_NOT_FOUND")

	containerRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/containers/"+seed.ContainerIDAPI1, nil)
	containerRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	containerResponse := httptest.NewRecorder()
	router.ServeHTTP(containerResponse, containerRequest)

	if containerResponse.Code != http.StatusOK {
		t.Fatalf("expected container detail status 200 for canonical id route, got %d", containerResponse.Code)
	}

	legacyContainerRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/containers/ctr-api-1", nil)
	legacyContainerRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	legacyContainerResponse := httptest.NewRecorder()
	router.ServeHTTP(legacyContainerResponse, legacyContainerRequest)

	if legacyContainerResponse.Code != http.StatusNotFound {
		t.Fatalf("expected legacy container id route to return 404, got %d", legacyContainerResponse.Code)
	}
	requireErrorCode(t, legacyContainerResponse, "CONTAINER_NOT_FOUND")
}

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func looksLikeUUID(value string) bool {
	return uuidPattern.MatchString(value)
}

func TestIngestSnapshotReplacesAndClearsDockerServiceState(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))
	replacementProjectID := "8bd11f62-ab9e-4e55-9068-8fd9f7498d6d"
	replacementContainerID := "adf4f2a7-d645-4862-b8e2-1f53fef4d37b"

	ingestSnapshot := func(body string) {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/snapshot", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Agent-Key", "demo-agent-key")

		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusAccepted {
			t.Fatalf("expected ingest status 202, got %d", response.Code)
		}
	}

	loadServerBundle := func() struct {
		Success bool `json:"success"`
		Data    struct {
			Services []struct {
				ID string `json:"id"`
			} `json:"services"`
			ContainerMetrics struct {
				CPU []map[string]any `json:"cpu"`
			} `json:"containerMetrics"`
		} `json:"data"`
	} {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer, nil)
		request.Header.Set("Authorization", "Bearer demo-owner-token")

		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("expected server detail status 200, got %d", response.Code)
		}

		var payload struct {
			Success bool `json:"success"`
			Data    struct {
				Services []struct {
					ID string `json:"id"`
				} `json:"services"`
				ContainerMetrics struct {
					CPU []map[string]any `json:"cpu"`
				} `json:"containerMetrics"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode server detail: %v", err)
		}
		return payload
	}

	loadProjects := func() struct {
		Success bool `json:"success"`
		Data    struct {
			Projects []struct {
				ID string `json:"id"`
			} `json:"projects"`
		} `json:"data"`
	} {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/projects", nil)
		request.Header.Set("Authorization", "Bearer demo-owner-token")

		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("expected project list status 200, got %d", response.Code)
		}

		var payload struct {
			Success bool `json:"success"`
			Data    struct {
				Projects []struct {
					ID string `json:"id"`
				} `json:"projects"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode project list: %v", err)
		}
		return payload
	}

	loadStandaloneContainers := func() struct {
		Success bool `json:"success"`
		Data    struct {
			Containers []struct {
				ID string `json:"id"`
			} `json:"containers"`
		} `json:"data"`
	} {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/containers?standalone=true", nil)
		request.Header.Set("Authorization", "Bearer demo-owner-token")

		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("expected standalone container list status 200, got %d", response.Code)
		}

		var payload struct {
			Success bool `json:"success"`
			Data    struct {
				Containers []struct {
					ID string `json:"id"`
				} `json:"containers"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode standalone container list: %v", err)
		}
		return payload
	}

	replacementIngest := `{
		"agent_id":"` + seed.AgentIDDemo + `",
		"server":{
			"id":"` + seed.ServerIDDevServer + `",
			"name":"Dev Server",
			"hostname":"devbox",
			"public_ip":"198.51.100.10",
			"agent_version":"0.1.0",
			"status":"up",
			"uptime_seconds":4000,
			"cpu_usage_pct":12,
			"memory_usage_pct":24,
			"disk_usage_pct":36,
			"network_rx_mb":1.5,
			"network_tx_mb":2.5,
			"load_average":"0.11 0.22 0.33",
			"os":"Ubuntu",
			"kernel":"6.8.0",
			"cpu_model":"AMD EPYC",
			"cpu_cores":4,
			"cpu_threads":8,
			"total_memory_gb":16,
			"total_disk_gb":128,
			"collected_at":"2026-03-23T04:00:00Z",
			"services":[
				{
					"id":"` + replacementProjectID + `",
					"name":"fresh-stack",
					"compose_project":"fresh-stack",
					"status":"running",
					"published_ports":["8088:8080"],
					"containers":[
						{
							"id":"` + replacementContainerID + `",
							"name":"fresh-stack-api-1",
							"image":"fresh-api:latest",
							"status":"running",
							"health":"healthy",
							"cpu_usage_pct":8,
							"memory_mb":256,
							"network_mb":4,
							"restart_count":0,
							"uptime":"10m",
							"ports":["8088:8080"],
							"command":"./fresh-api",
							"last_seen_at":"2026-03-23T04:00:00Z"
						}
					]
				}
			]
		},
		"metrics":[],
		"logs":[]
	}`

	ingestSnapshot(replacementIngest)

	bundleAfterReplacement := loadServerBundle()
	if len(bundleAfterReplacement.Data.Services) != 1 {
		t.Fatalf("expected replaced snapshot to leave only fresh-stack service, got %+v", bundleAfterReplacement.Data.Services)
	}
	if len(bundleAfterReplacement.Data.ContainerMetrics.CPU) == 0 {
		t.Fatalf("expected container metrics after replacement ingest")
	}
	foundReplacementSeries := false
	for _, point := range bundleAfterReplacement.Data.ContainerMetrics.CPU {
		for key := range point {
			if key == "timestamp" {
				continue
			}
			if key == replacementContainerID {
				foundReplacementSeries = true
			}
		}
		if _, ok := point[seed.ContainerIDAPI1]; ok {
			t.Fatalf("expected stale seeded project container metric series to be pruned")
		}
		if _, ok := point[seed.ContainerIDEdgeProxy1]; ok {
			t.Fatalf("expected stale standalone container metric series to be pruned")
		}
	}
	if !foundReplacementSeries {
		t.Fatalf("expected replacement container metric series %q to be present", replacementContainerID)
	}

	projectsAfterReplacement := loadProjects()
	if len(projectsAfterReplacement.Data.Projects) != 1 {
		t.Fatalf("expected replaced snapshot project list to contain only fresh-stack, got %+v", projectsAfterReplacement.Data.Projects)
	}
	freshProjectID := projectsAfterReplacement.Data.Projects[0].ID
	if freshProjectID != replacementProjectID {
		t.Fatalf("expected replacement project id %q, got %q", replacementProjectID, freshProjectID)
	}

	staleProjectRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/projects/"+seed.ServiceIDServiceA, nil)
	staleProjectRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	staleProjectResponse := httptest.NewRecorder()
	router.ServeHTTP(staleProjectResponse, staleProjectRequest)

	if staleProjectResponse.Code != http.StatusNotFound {
		t.Fatalf("expected stale project detail to return 404 after replacement ingest, got %d", staleProjectResponse.Code)
	}

	freshProjectRequestBeforeEmpty := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/projects/"+replacementProjectID, nil)
	freshProjectRequestBeforeEmpty.Header.Set("Authorization", "Bearer demo-owner-token")
	freshProjectResponseBeforeEmpty := httptest.NewRecorder()
	router.ServeHTTP(freshProjectResponseBeforeEmpty, freshProjectRequestBeforeEmpty)

	if freshProjectResponseBeforeEmpty.Code != http.StatusOK {
		t.Fatalf("expected replacement project detail to return 200 before empty ingest, got %d", freshProjectResponseBeforeEmpty.Code)
	}

	standaloneAfterReplacement := loadStandaloneContainers()
	if len(standaloneAfterReplacement.Data.Containers) != 0 {
		t.Fatalf("expected replacement ingest with no standalone containers to clear stale standalone list, got %+v", standaloneAfterReplacement.Data.Containers)
	}

	emptyIngest := `{
		"agent_id":"` + seed.AgentIDDemo + `",
		"server":{
			"id":"` + seed.ServerIDDevServer + `",
			"name":"Dev Server",
			"hostname":"devbox",
			"public_ip":"198.51.100.10",
			"agent_version":"0.1.0",
			"status":"up",
			"uptime_seconds":5000,
			"cpu_usage_pct":10,
			"memory_usage_pct":20,
			"disk_usage_pct":30,
			"network_rx_mb":1,
			"network_tx_mb":2,
			"load_average":"0.10 0.20 0.30",
			"os":"Ubuntu",
			"kernel":"6.8.0",
			"cpu_model":"AMD EPYC",
			"cpu_cores":4,
			"cpu_threads":8,
			"total_memory_gb":16,
			"total_disk_gb":128,
			"collected_at":"2026-03-23T05:00:00Z",
			"services":[]
		},
		"metrics":[],
		"logs":[]
	}`

	ingestSnapshot(emptyIngest)

	bundleAfterEmpty := loadServerBundle()
	if len(bundleAfterEmpty.Data.Services) != 0 {
		t.Fatalf("expected empty snapshot to clear all services, got %+v", bundleAfterEmpty.Data.Services)
	}
	if len(bundleAfterEmpty.Data.ContainerMetrics.CPU) == 0 {
		t.Fatalf("expected container metrics points to remain readable after empty ingest")
	}
	for _, point := range bundleAfterEmpty.Data.ContainerMetrics.CPU {
		for key := range point {
			if key != "timestamp" {
				t.Fatalf("expected empty snapshot to remove all container metric series, found %q", key)
			}
		}
	}

	projectsAfterEmpty := loadProjects()
	if len(projectsAfterEmpty.Data.Projects) != 0 {
		t.Fatalf("expected empty snapshot to clear project list, got %+v", projectsAfterEmpty.Data.Projects)
	}

	freshProjectRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/projects/"+freshProjectID, nil)
	freshProjectRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	freshProjectResponse := httptest.NewRecorder()
	router.ServeHTTP(freshProjectResponse, freshProjectRequest)

	if freshProjectResponse.Code != http.StatusNotFound {
		t.Fatalf("expected cleared project detail to return 404 after empty ingest, got %d", freshProjectResponse.Code)
	}

	standaloneAfterEmpty := loadStandaloneContainers()
	if len(standaloneAfterEmpty.Data.Containers) != 0 {
		t.Fatalf("expected empty snapshot to keep standalone container list empty, got %+v", standaloneAfterEmpty.Data.Containers)
	}
}

func TestServiceLogsIncludeFrontendLogFields(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/services/"+seed.ServiceIDServiceA+"/logs", nil)
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

	projectDetailRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/projects/"+seed.ServiceIDServiceA, nil)
	projectDetailRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	projectDetailResponse := httptest.NewRecorder()
	router.ServeHTTP(projectDetailResponse, projectDetailRequest)

	if projectDetailResponse.Code != http.StatusOK {
		t.Fatalf("expected project detail status 200, got %d", projectDetailResponse.Code)
	}

	projectMetricsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/projects/"+seed.ServiceIDServiceA+"/metrics", nil)
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
	if _, ok := metricsPayload.Data.Metrics.CPU[0][seed.ContainerIDAPI1]; !ok {
		t.Fatalf("expected project metrics to preserve container-id keys")
	}
	if _, ok := metricsPayload.Data.Metrics.CPU[0][seed.ContainerIDQdrant1]; ok {
		t.Fatalf("expected project metrics to exclude containers from other projects")
	}

	projectLogsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/projects/"+seed.ServiceIDServiceA+"/logs?search=webhook", nil)
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

	projectEventsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/projects/"+seed.ServiceIDSearchStack+"/events", nil)
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
		t.Fatalf("expected persisted project events")
	}
}

func TestContainerEndpointsReturnStandaloneListDetailMetricsLogsEventsAndEnv(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	containersRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/containers?standalone=true", nil)
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
	if len(containerListPayload.Data.Containers) == 0 || containerListPayload.Data.Containers[0].ID != seed.ContainerIDEdgeProxy1 {
		t.Fatalf("expected seeded standalone container in list")
	}

	containerDetailRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/containers/"+seed.ContainerIDAPI1, nil)
	containerDetailRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	containerDetailResponse := httptest.NewRecorder()
	router.ServeHTTP(containerDetailResponse, containerDetailRequest)

	if containerDetailResponse.Code != http.StatusOK {
		t.Fatalf("expected container detail status 200, got %d", containerDetailResponse.Code)
	}

	containerMetricsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/containers/"+seed.ContainerIDAPI1+"/metrics", nil)
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

	containerLogsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/containers/"+seed.ContainerIDAPI1+"/logs?limit=1", nil)
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
	if len(containerLogsPayload.Data.Logs) != 1 || containerLogsPayload.Data.Logs[0].ContainerID != seed.ContainerIDAPI1 {
		t.Fatalf("expected limited container log response")
	}

	containerEventsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/containers/"+seed.ContainerIDIngest1+"/events", nil)
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
		t.Fatalf("expected persisted container events")
	}

	containerEnvRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/containers/"+seed.ContainerIDEdgeProxy1+"/env", nil)
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

func TestContainerEventsPersistAcrossStateTransitions(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	ingestBody := `{
		"agent_id":"` + seed.AgentIDDemo + `",
		"server":{
			"id":"` + seed.ServerIDDevServer + `",
			"name":"dev-server",
			"hostname":"ubuntu-bifrost",
			"public_ip":"203.0.113.42",
			"agent_version":"0.1.0",
			"status":"up",
			"uptime_seconds":100,
			"cpu_usage_pct":10,
			"memory_usage_pct":20,
			"disk_usage_pct":30,
			"network_rx_mb":1,
			"network_tx_mb":1,
			"load_average":"0.10 0.10 0.10",
			"os":"Ubuntu",
			"kernel":"6.8.0",
			"cpu_model":"AMD EPYC",
			"cpu_cores":4,
			"cpu_threads":8,
			"total_memory_gb":16,
			"total_disk_gb":256,
			"collected_at":"2026-03-23T06:00:00Z",
			"services":[
				{
					"id":"` + seed.ServiceIDServiceA + `",
					"name":"service-a",
					"compose_project":"service-a",
					"status":"degraded",
					"published_ports":["8000:8000"],
					"containers":[
						{
							"id":"` + seed.ContainerIDAPI1 + `",
							"name":"service-a-backend-1",
							"image":"ghcr.io/acme/service-a-backend:latest",
							"status":"exited",
							"health":"unhealthy",
							"cpu_usage_pct":0,
							"memory_mb":0,
							"network_mb":0,
							"restart_count":2,
							"uptime":"0m",
							"ports":["8000:8000"],
							"command":"bin/api",
							"last_seen_at":"2026-03-23T06:00:00Z"
						},
						{
							"id":"` + seed.ContainerIDWorker1 + `",
							"name":"service-a-worker-1",
							"image":"ghcr.io/acme/service-a-worker:latest",
							"status":"running",
							"health":"healthy",
							"cpu_usage_pct":1,
							"memory_mb":128,
							"network_mb":0.1,
							"restart_count":1,
							"uptime":"1m",
							"ports":[],
							"command":"bin/worker",
							"last_seen_at":"2026-03-23T06:00:00Z"
						},
						{
							"id":"` + seed.ContainerIDWeb1 + `",
							"name":"service-a-frontend-1",
							"image":"ghcr.io/acme/service-a-frontend:latest",
							"status":"running",
							"health":"healthy",
							"cpu_usage_pct":1,
							"memory_mb":128,
							"network_mb":0.1,
							"restart_count":0,
							"uptime":"1m",
							"ports":["3000:3000"],
							"command":"node server.js",
							"last_seen_at":"2026-03-23T06:00:00Z"
						}
					]
				},
				{
					"id":"` + seed.ServiceIDSearchStack + `",
					"name":"search-stack",
					"compose_project":"search-stack",
					"status":"running",
					"published_ports":["6333:6333"],
					"containers":[
						{
							"id":"` + seed.ContainerIDQdrant1 + `",
							"name":"search-stack-qdrant-1",
							"image":"qdrant/qdrant:v1.13.4",
							"status":"running",
							"health":"healthy",
							"cpu_usage_pct":1,
							"memory_mb":128,
							"network_mb":0.1,
							"restart_count":0,
							"uptime":"1m",
							"ports":["6333:6333"],
							"command":"./entrypoint.sh",
							"last_seen_at":"2026-03-23T06:00:00Z"
						},
						{
							"id":"` + seed.ContainerIDIngest1 + `",
							"name":"search-stack-ingest-1",
							"image":"ghcr.io/acme/search-ingest:latest",
							"status":"running",
							"health":"degraded",
							"cpu_usage_pct":1,
							"memory_mb":128,
							"network_mb":0.1,
							"restart_count":2,
							"uptime":"1m",
							"ports":[],
							"command":"bin/ingest",
							"last_seen_at":"2026-03-23T06:00:00Z"
						}
					]
				},
				{
					"id":"` + seed.ServiceIDEdgeProxy + `",
					"name":"edge-proxy-1",
					"compose_project":"",
					"status":"running",
					"published_ports":["8080:80"],
					"containers":[
						{
							"id":"` + seed.ContainerIDEdgeProxy1 + `",
							"name":"edge-proxy-1",
							"image":"nginx:1.27-alpine",
							"status":"running",
							"health":"healthy",
							"cpu_usage_pct":1,
							"memory_mb":64,
							"network_mb":0.1,
							"restart_count":0,
							"uptime":"1m",
							"ports":["8080:80"],
							"command":"nginx -g 'daemon off;'",
							"last_seen_at":"2026-03-23T06:00:00Z"
						}
					]
				}
			]
		},
		"metrics":[],
		"logs":[]
	}`

	ingestRequest := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/snapshot", strings.NewReader(ingestBody))
	ingestRequest.Header.Set("Content-Type", "application/json")
	ingestRequest.Header.Set("X-Agent-Key", "demo-agent-key")
	ingestResponse := httptest.NewRecorder()
	router.ServeHTTP(ingestResponse, ingestRequest)

	if ingestResponse.Code != http.StatusAccepted {
		t.Fatalf("expected ingest status 202, got %d", ingestResponse.Code)
	}

	eventsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer+"/containers/"+seed.ContainerIDAPI1+"/events", nil)
	eventsRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	eventsResponse := httptest.NewRecorder()
	router.ServeHTTP(eventsResponse, eventsRequest)

	if eventsResponse.Code != http.StatusOK {
		t.Fatalf("expected container events status 200, got %d", eventsResponse.Code)
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Events []struct {
				Type string `json:"type"`
			} `json:"events"`
		} `json:"data"`
	}
	if err := json.Unmarshal(eventsResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode transitioned container events: %v", err)
	}

	foundStart := false
	foundStop := false
	foundRestart := false
	for _, event := range payload.Data.Events {
		switch event.Type {
		case "start":
			foundStart = true
		case "stop":
			foundStop = true
		case "restart":
			foundRestart = true
		}
	}
	if !foundStart || !foundStop || !foundRestart {
		t.Fatalf("expected persisted events to include prior start plus new stop/restart, got %+v", payload.Data.Events)
	}
}

func TestIngestRewritesLegacyRuntimeMonitoringIDsToCanonicalUUIDs(t *testing.T) {
	router := NewRouter(config.Config{}, store.NewMemoryStore(seed.Data()))

	const legacyServiceID = "svc-zhiro"
	const legacyContainerID = "2263a588fa73c3b8e1efda820b74a0374dd872bd2cca887045b40e6e1b10c35d"

	ingestBody := `{
		"agent_id":"` + seed.AgentIDDemo + `",
		"server":{
			"id":"` + seed.ServerIDDevServer + `",
			"name":"dev-server",
			"hostname":"ubuntu-bifrost",
			"public_ip":"203.0.113.42",
			"agent_version":"0.1.0",
			"status":"up",
			"uptime_seconds":100,
			"cpu_usage_pct":10,
			"memory_usage_pct":20,
			"disk_usage_pct":30,
			"network_rx_mb":1,
			"network_tx_mb":1,
			"load_average":"0.10 0.10 0.10",
			"os":"Ubuntu",
			"kernel":"6.8.0",
			"cpu_model":"AMD EPYC",
			"cpu_cores":4,
			"cpu_threads":8,
			"total_memory_gb":16,
			"total_disk_gb":256,
			"collected_at":"2026-03-23T06:00:00Z",
			"services":[
				{
					"id":"` + legacyServiceID + `",
					"name":"zhiro",
					"compose_project":"zhiro",
					"status":"running",
					"published_ports":["8000:8000"],
					"containers":[
						{
							"id":"` + legacyContainerID + `",
							"name":"zhiro-app",
							"image":"zhiro-app",
							"status":"running",
							"health":"healthy",
							"cpu_usage_pct":0.15,
							"memory_mb":92.46,
							"network_mb":0.03,
							"restart_count":0,
							"uptime":"1h",
							"ports":["8000:8000"],
							"command":"uvicorn src.main:app --host 0.0.0.0 --port 8000 --workers 1",
							"last_seen_at":"2026-03-23T06:00:00Z"
						}
					]
				}
			]
		},
		"metrics":[],
		"logs":[
			{
				"server_id":"` + seed.ServerIDDevServer + `",
				"service_id":"` + legacyServiceID + `",
				"container_id":"` + legacyContainerID + `",
				"level":"info",
				"message":"app boot complete",
				"timestamp":"2026-03-23T06:00:00Z"
			}
		]
	}`

	ingestRequest := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/snapshot", strings.NewReader(ingestBody))
	ingestRequest.Header.Set("Content-Type", "application/json")
	ingestRequest.Header.Set("X-Agent-Key", "demo-agent-key")
	ingestResponse := httptest.NewRecorder()
	router.ServeHTTP(ingestResponse, ingestRequest)

	if ingestResponse.Code != http.StatusAccepted {
		t.Fatalf("expected ingest status 202, got %d", ingestResponse.Code)
	}

	serverRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+seed.ServerIDDevServer, nil)
	serverRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	serverResponse := httptest.NewRecorder()
	router.ServeHTTP(serverResponse, serverRequest)

	if serverResponse.Code != http.StatusOK {
		t.Fatalf("expected server detail status 200, got %d", serverResponse.Code)
	}

	var serverPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Services []struct {
				ID             string `json:"id"`
				ComposeProject string `json:"compose_project"`
				Containers     []struct {
					ID string `json:"id"`
				} `json:"containers"`
			} `json:"services"`
			ContainerMetrics struct {
				CPU []map[string]any `json:"cpu"`
			} `json:"containerMetrics"`
		} `json:"data"`
	}
	if err := json.Unmarshal(serverResponse.Body.Bytes(), &serverPayload); err != nil {
		t.Fatalf("decode server payload: %v", err)
	}
	if len(serverPayload.Data.Services) != 1 {
		t.Fatalf("expected a single live service after ingest, got %d", len(serverPayload.Data.Services))
	}

	service := serverPayload.Data.Services[0]
	if service.ComposeProject != "zhiro" {
		t.Fatalf("expected zhiro compose project, got %+v", service)
	}
	if service.ID == legacyServiceID || !looksLikeUUID(service.ID) {
		t.Fatalf("expected canonical uuid service id instead of %q", service.ID)
	}
	if len(service.Containers) != 1 {
		t.Fatalf("expected a single live container after ingest, got %d", len(service.Containers))
	}

	containerID := service.Containers[0].ID
	if containerID == legacyContainerID || !looksLikeUUID(containerID) {
		t.Fatalf("expected canonical uuid container id instead of %q", containerID)
	}

	if len(serverPayload.Data.ContainerMetrics.CPU) == 0 {
		t.Fatalf("expected container metric points to be present")
	}
	lastPoint := serverPayload.Data.ContainerMetrics.CPU[len(serverPayload.Data.ContainerMetrics.CPU)-1]
	if _, ok := lastPoint[containerID]; !ok {
		t.Fatalf("expected container metrics to use canonical container id %q", containerID)
	}
	if _, ok := lastPoint[legacyContainerID]; ok {
		t.Fatalf("expected raw docker container id %q to be absent from container metrics", legacyContainerID)
	}

	serviceLogsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/services/"+service.ID+"/logs", nil)
	serviceLogsRequest.Header.Set("Authorization", "Bearer demo-owner-token")
	serviceLogsResponse := httptest.NewRecorder()
	router.ServeHTTP(serviceLogsResponse, serviceLogsRequest)

	if serviceLogsResponse.Code != http.StatusOK {
		t.Fatalf("expected service logs status 200, got %d", serviceLogsResponse.Code)
	}

	var logsPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Logs []struct {
				ServiceID   string `json:"service_id"`
				ContainerID string `json:"container_id"`
				Message     string `json:"message"`
			} `json:"logs"`
		} `json:"data"`
	}
	if err := json.Unmarshal(serviceLogsResponse.Body.Bytes(), &logsPayload); err != nil {
		t.Fatalf("decode service logs payload: %v", err)
	}
	if len(logsPayload.Data.Logs) != 1 {
		t.Fatalf("expected one log line, got %d", len(logsPayload.Data.Logs))
	}
	if logsPayload.Data.Logs[0].ServiceID != service.ID || logsPayload.Data.Logs[0].ContainerID != containerID {
		t.Fatalf("expected logs to be remapped to canonical ids, got %+v", logsPayload.Data.Logs[0])
	}
}

func seedWithViewer() store.SeedData {
	data := seed.Data()
	data.Users = append(data.Users, domain.User{
		ID:        "a27aa815-4c28-4f75-a31a-4e9fe10e7f93",
		TenantID:  seed.TenantIDDemo,
		Email:     "viewer@bifrost.local",
		Name:      "Bifrost Viewer",
		Password:  "viewer123",
		Role:      domain.RoleViewer,
		AuthToken: "viewer-token",
	})
	return data
}
