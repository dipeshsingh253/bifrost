package config

import "os"

type Config struct {
	ListenAddr string
	JWTSecret  string
}

func Load() Config {
	return Config{
		ListenAddr: env("BIFROST_LISTEN_ADDR", ":8080"),
		JWTSecret:  env("BIFROST_JWT_SECRET", "bifrost-dev-secret"),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
