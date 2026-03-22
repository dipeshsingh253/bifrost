package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServerID            string          `yaml:"server_id"`
	ServerName          string          `yaml:"server_name"`
	TenantID            string          `yaml:"tenant_id"`
	BackendURL          string          `yaml:"backend_url"`
	APIKey              string          `yaml:"api_key"`
	PollIntervalSeconds int             `yaml:"poll_interval_seconds"`
	Collectors          CollectorConfig `yaml:"collectors"`
	Docker              DockerConfig    `yaml:"docker"`
}

type CollectorConfig struct {
	Host   bool `yaml:"host"`
	Docker bool `yaml:"docker"`
	Logs   bool `yaml:"logs"`
}

type DockerConfig struct {
	IncludeProjects   []string `yaml:"include_projects"`
	ExcludeContainers []string `yaml:"exclude_containers"`
}

func Load(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return Config{}, err
	}

	if cfg.PollIntervalSeconds == 0 {
		cfg.PollIntervalSeconds = 10
	}

	return cfg, nil
}
