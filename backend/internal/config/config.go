package config

import (
	"os"
	"strings"
)

type Config struct {
	ListenAddr       string
	JWTSecret        string
	DatabaseURL      string
	AgentBackendURL  string
	AgentDockerImage string
	AgentBinaryPath  string
	AgentSourceDir   string
	SeedDemoData     bool
}

func Load() Config {
	loadDotEnvFiles(".env", ".env.local")

	return Config{
		ListenAddr:       env("BIFROST_LISTEN_ADDR", ":8080"),
		JWTSecret:        env("BIFROST_JWT_SECRET", "bifrost-dev-secret"),
		DatabaseURL:      env("BIFROST_DATABASE_URL", "postgres://bifrost:bifrost@127.0.0.1:5433/bifrost?sslmode=disable"),
		AgentBackendURL:  env("BIFROST_AGENT_BACKEND_URL", "http://localhost:8080"),
		AgentDockerImage: env("BIFROST_AGENT_DOCKER_IMAGE", "bifrost-agent:latest"),
		AgentBinaryPath:  env("BIFROST_AGENT_BINARY_PATH", ""),
		AgentSourceDir:   env("BIFROST_AGENT_SOURCE_DIR", "../agent"),
		SeedDemoData:     envBool("BIFROST_SEED_DEMO_DATA", false),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}

	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
