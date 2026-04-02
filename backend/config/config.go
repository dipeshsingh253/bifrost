package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr              string
	JWTSecret               string
	DatabaseURL             string
	AgentBackendURL         string
	AgentDockerImage        string
	AgentBinaryPath         string
	AgentSourceDir          string
	SeedDemoData            bool
	ServerReadHeaderTimeout time.Duration
	ServerReadTimeout       time.Duration
	ServerWriteTimeout      time.Duration
	ServerIdleTimeout       time.Duration
	ServerMaxHeaderBytes    int
	TrustedProxies          []string
	DBMaxOpenConns          int
	DBMaxIdleConns          int
	DBConnMaxLifetime       time.Duration
	DBConnMaxIdleTime       time.Duration
	DBReadTimeout           time.Duration
	DBWriteTimeout          time.Duration
	DBIngestTimeout         time.Duration
}

func Load() Config {
	loadDotEnvFiles(".env", ".env.local")

	return Config{
		ListenAddr:              env("BIFROST_LISTEN_ADDR", ":8080"),
		JWTSecret:               env("BIFROST_JWT_SECRET", "bifrost-dev-secret"),
		DatabaseURL:             env("BIFROST_DATABASE_URL", "postgres://bifrost:bifrost@127.0.0.1:5433/bifrost?sslmode=disable"),
		AgentBackendURL:         env("BIFROST_AGENT_BACKEND_URL", "http://localhost:8080"),
		AgentDockerImage:        env("BIFROST_AGENT_DOCKER_IMAGE", "bifrost-agent:latest"),
		AgentBinaryPath:         env("BIFROST_AGENT_BINARY_PATH", ""),
		AgentSourceDir:          env("BIFROST_AGENT_SOURCE_DIR", "../agent"),
		SeedDemoData:            envBool("BIFROST_SEED_DEMO_DATA", false),
		ServerReadHeaderTimeout: envDuration("BIFROST_SERVER_READ_HEADER_TIMEOUT", 5*time.Second),
		ServerReadTimeout:       envDuration("BIFROST_SERVER_READ_TIMEOUT", 15*time.Second),
		ServerWriteTimeout:      envDuration("BIFROST_SERVER_WRITE_TIMEOUT", 30*time.Second),
		ServerIdleTimeout:       envDuration("BIFROST_SERVER_IDLE_TIMEOUT", 60*time.Second),
		ServerMaxHeaderBytes:    envInt("BIFROST_SERVER_MAX_HEADER_BYTES", 1<<20),
		TrustedProxies:          envList("BIFROST_TRUSTED_PROXIES"),
		DBMaxOpenConns:          envInt("BIFROST_DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:          envInt("BIFROST_DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime:       envDuration("BIFROST_DB_CONN_MAX_LIFETIME", 30*time.Minute),
		DBConnMaxIdleTime:       envDuration("BIFROST_DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
		DBReadTimeout:           envDuration("BIFROST_DB_READ_TIMEOUT", 3*time.Second),
		DBWriteTimeout:          envDuration("BIFROST_DB_WRITE_TIMEOUT", 5*time.Second),
		DBIngestTimeout:         envDuration("BIFROST_DB_INGEST_TIMEOUT", 10*time.Second),
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

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func envList(key string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		values = append(values, part)
	}
	return values
}
