package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSupportsEnvOnlyConfigurationWhenFileIsMissing(t *testing.T) {
	t.Setenv("BIFROST_AGENT_ID", "2f838072-1c13-4b25-978d-aadfa9edf66d")
	t.Setenv("BIFROST_SERVER_ID", "a0ad9ea6-5d4d-41bf-babc-a540ca2575be")
	t.Setenv("BIFROST_SERVER_NAME", "Validation VPS")
	t.Setenv("BIFROST_TENANT_ID", "15d0a6bd-c94e-46cf-830c-ba71af57b524")
	t.Setenv("BIFROST_BACKEND_URL", "https://bifrost.example.com")
	t.Setenv("BIFROST_ENROLLMENT_TOKEN", "bootstrap-token")
	t.Setenv("BIFROST_COLLECT_HOST", "true")
	t.Setenv("BIFROST_COLLECT_DOCKER", "true")
	t.Setenv("BIFROST_COLLECT_LOGS", "false")
	t.Setenv("BIFROST_DOCKER_INCLUDE_ALL", "false")
	t.Setenv("BIFROST_DOCKER_INCLUDE_PROJECTS", "zhiro,bifrost")
	t.Setenv("BIFROST_DOCKER_INCLUDE_CONTAINERS", "api,worker")
	t.Setenv("BIFROST_DOCKER_EXCLUDE_PROJECTS", "legacy")
	t.Setenv("BIFROST_DOCKER_EXCLUDE_CONTAINERS", "redis,postgres")
	t.Setenv("BIFROST_LOGS_MAX_LINES_PER_FETCH", "300")

	cfg, err := Load(filepath.Join(t.TempDir(), "missing-config.yaml"))
	if err != nil {
		t.Fatalf("load env-only config: %v", err)
	}

	if cfg.AgentID != "2f838072-1c13-4b25-978d-aadfa9edf66d" || cfg.ServerID != "a0ad9ea6-5d4d-41bf-babc-a540ca2575be" {
		t.Fatalf("expected env ids to load, got %+v", cfg)
	}
	if cfg.EnrollmentToken != "bootstrap-token" || cfg.BackendURL != "https://bifrost.example.com" {
		t.Fatalf("expected env credentials to load, got %+v", cfg)
	}
	if !cfg.Collectors.Host || !cfg.Collectors.Docker || cfg.Collectors.Logs {
		t.Fatalf("expected collector flags from env, got %+v", cfg.Collectors)
	}
	if cfg.Docker.IncludeAll {
		t.Fatalf("expected include_all override from env, got %+v", cfg.Docker)
	}
	if len(cfg.Docker.IncludeProjects) != 2 || cfg.Docker.IncludeProjects[0] != "zhiro" {
		t.Fatalf("expected include projects from env, got %+v", cfg.Docker.IncludeProjects)
	}
	if len(cfg.Docker.IncludeContainers) != 2 || cfg.Docker.IncludeContainers[1] != "worker" {
		t.Fatalf("expected include containers from env, got %+v", cfg.Docker.IncludeContainers)
	}
	if len(cfg.Docker.ExcludeProjects) != 1 || cfg.Docker.ExcludeProjects[0] != "legacy" {
		t.Fatalf("expected exclude projects from env, got %+v", cfg.Docker.ExcludeProjects)
	}
	if len(cfg.Docker.ExcludeContainers) != 2 || cfg.Docker.ExcludeContainers[1] != "postgres" {
		t.Fatalf("expected exclude containers from env, got %+v", cfg.Docker.ExcludeContainers)
	}
	if cfg.Logs.MaxLinesPerFetch != 300 {
		t.Fatalf("expected logs.max_lines_per_fetch from env, got %+v", cfg.Logs)
	}
}

func TestLoadPreservesSavedAPIKeyWhenOnlyBootstrapEnvVarsRemain(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(`
agent_id: 2f838072-1c13-4b25-978d-aadfa9edf66d
server_id: a0ad9ea6-5d4d-41bf-babc-a540ca2575be
server_name: Validation VPS
tenant_id: 15d0a6bd-c94e-46cf-830c-ba71af57b524
backend_url: https://bifrost.example.com
api_key: long-lived-api-key
poll_interval_seconds: 10
`), 0o600); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	t.Setenv("BIFROST_ENROLLMENT_TOKEN", "bootstrap-token")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load mixed config: %v", err)
	}

	if cfg.APIKey != "long-lived-api-key" {
		t.Fatalf("expected saved api key to win, got %+v", cfg)
	}
	if cfg.EnrollmentToken != "bootstrap-token" {
		t.Fatalf("expected bootstrap token to remain available, got %+v", cfg)
	}
}

func TestLoadAppliesDockerAndLogsDefaults(t *testing.T) {
	t.Setenv("BIFROST_AGENT_ID", "2f838072-1c13-4b25-978d-aadfa9edf66d")
	t.Setenv("BIFROST_SERVER_ID", "a0ad9ea6-5d4d-41bf-babc-a540ca2575be")
	t.Setenv("BIFROST_SERVER_NAME", "Validation VPS")
	t.Setenv("BIFROST_TENANT_ID", "15d0a6bd-c94e-46cf-830c-ba71af57b524")
	t.Setenv("BIFROST_BACKEND_URL", "https://bifrost.example.com")
	t.Setenv("BIFROST_ENROLLMENT_TOKEN", "bootstrap-token")

	cfg, err := Load(filepath.Join(t.TempDir(), "missing-config.yaml"))
	if err != nil {
		t.Fatalf("load defaulted config: %v", err)
	}

	if !cfg.Docker.IncludeAll {
		t.Fatalf("expected docker.include_all to default true, got %+v", cfg.Docker)
	}
	if !cfg.Collectors.Host || !cfg.Collectors.Docker || !cfg.Collectors.Logs {
		t.Fatalf("expected collectors to default on, got %+v", cfg.Collectors)
	}
	if cfg.Logs.MaxLinesPerFetch != 200 {
		t.Fatalf("expected logs.max_lines_per_fetch to default 200, got %+v", cfg.Logs)
	}
	if cfg.PollIntervalSeconds != 10 {
		t.Fatalf("expected poll interval default 10, got %d", cfg.PollIntervalSeconds)
	}
}
