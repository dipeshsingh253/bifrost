package config

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AgentID             string          `yaml:"agent_id"`
	ServerID            string          `yaml:"server_id"`
	ServerName          string          `yaml:"server_name"`
	TenantID            string          `yaml:"tenant_id"`
	BackendURL          string          `yaml:"backend_url"`
	APIKey              string          `yaml:"api_key"`
	EnrollmentToken     string          `yaml:"enrollment_token"`
	PollIntervalSeconds int             `yaml:"poll_interval_seconds"`
	Collectors          CollectorConfig `yaml:"collectors"`
	Docker              DockerConfig    `yaml:"docker"`
	Logs                LogsConfig      `yaml:"logs"`
}

type CollectorConfig struct {
	Host   bool `yaml:"host"`
	Docker bool `yaml:"docker"`
	Logs   bool `yaml:"logs"`
}

type DockerConfig struct {
	IncludeAll        bool     `yaml:"include_all"`
	IncludeProjects   []string `yaml:"include_projects"`
	IncludeContainers []string `yaml:"include_containers"`
	ExcludeProjects   []string `yaml:"exclude_projects"`
	ExcludeContainers []string `yaml:"exclude_containers"`
}

type LogsConfig struct {
	MaxLinesPerFetch int `yaml:"max_lines_per_fetch"`
}

func Load(path string) (Config, error) {
	cfg := Config{
		PollIntervalSeconds: 10,
		Collectors: CollectorConfig{
			Host:   true,
			Docker: true,
			Logs:   true,
		},
		Docker: DockerConfig{
			IncludeAll: true,
		},
		Logs: LogsConfig{
			MaxLinesPerFetch: 200,
		},
	}

	content, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := yaml.Unmarshal(content, &cfg); err != nil {
			return Config{}, err
		}
	case errors.Is(err, os.ErrNotExist):
	default:
		return Config{}, err
	}

	applyEnvOverrides(&cfg)

	return cfg, nil
}

func (cfg Config) Validate() error {
	switch {
	case cfg.AgentID == "":
		return errInvalidConfig("agent_id is required")
	case cfg.ServerID == "":
		return errInvalidConfig("server_id is required")
	case cfg.BackendURL == "":
		return errInvalidConfig("backend_url is required")
	case cfg.APIKey == "" && cfg.EnrollmentToken == "":
		return errInvalidConfig("api_key or enrollment_token is required")
	case cfg.PollIntervalSeconds <= 0:
		return errInvalidConfig("poll_interval_seconds must be greater than 0")
	case cfg.Logs.MaxLinesPerFetch <= 0:
		return errInvalidConfig("logs.max_lines_per_fetch must be greater than 0")
	default:
		return nil
	}
}

func Save(path string, cfg Config) error {
	content, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dirForPath(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o600)
}

type invalidConfigError string

func (e invalidConfigError) Error() string {
	return string(e)
}

func errInvalidConfig(message string) error {
	return invalidConfigError(message)
}

func applyEnvOverrides(cfg *Config) {
	setString := func(envValue string, current *string) {
		if value := strings.TrimSpace(os.Getenv(envValue)); value != "" {
			*current = value
		}
	}

	setBool := func(envValue string, current *bool) {
		value := strings.TrimSpace(strings.ToLower(os.Getenv(envValue)))
		switch value {
		case "1", "true", "yes", "on":
			*current = true
		case "0", "false", "no", "off":
			*current = false
		}
	}

	setList := func(envValue string, current *[]string) {
		value := strings.TrimSpace(os.Getenv(envValue))
		if value == "" {
			return
		}
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
		*current = result
	}

	setString("BIFROST_AGENT_ID", &cfg.AgentID)
	setString("BIFROST_SERVER_ID", &cfg.ServerID)
	setString("BIFROST_SERVER_NAME", &cfg.ServerName)
	setString("BIFROST_TENANT_ID", &cfg.TenantID)
	setString("BIFROST_BACKEND_URL", &cfg.BackendURL)
	setString("BIFROST_API_KEY", &cfg.APIKey)
	setString("BIFROST_ENROLLMENT_TOKEN", &cfg.EnrollmentToken)
	if value := strings.TrimSpace(os.Getenv("BIFROST_POLL_INTERVAL_SECONDS")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.PollIntervalSeconds = parsed
		}
	}

	setBool("BIFROST_COLLECT_HOST", &cfg.Collectors.Host)
	setBool("BIFROST_COLLECT_DOCKER", &cfg.Collectors.Docker)
	setBool("BIFROST_COLLECT_LOGS", &cfg.Collectors.Logs)
	setBool("BIFROST_DOCKER_INCLUDE_ALL", &cfg.Docker.IncludeAll)
	setList("BIFROST_DOCKER_INCLUDE_PROJECTS", &cfg.Docker.IncludeProjects)
	setList("BIFROST_DOCKER_INCLUDE_CONTAINERS", &cfg.Docker.IncludeContainers)
	setList("BIFROST_DOCKER_EXCLUDE_PROJECTS", &cfg.Docker.ExcludeProjects)
	setList("BIFROST_DOCKER_EXCLUDE_CONTAINERS", &cfg.Docker.ExcludeContainers)
	if value := strings.TrimSpace(os.Getenv("BIFROST_LOGS_MAX_LINES_PER_FETCH")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.Logs.MaxLinesPerFetch = parsed
		}
	}
}

func dirForPath(path string) string {
	index := strings.LastIndex(path, "/")
	if index <= 0 {
		return "."
	}
	return path[:index]
}
