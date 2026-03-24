package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsDotEnvFile(t *testing.T) {
	workdir := t.TempDir()
	writeTestFile(t, filepath.Join(workdir, ".env"), `
BIFROST_LISTEN_ADDR=:18080
BIFROST_JWT_SECRET=env-secret
BIFROST_DATABASE_URL=postgres://env-user:env-pass@127.0.0.1:6432/env-db?sslmode=disable
`)

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(previous); chdirErr != nil {
			t.Fatalf("restore cwd: %v", chdirErr)
		}
	}()

	if err := os.Chdir(workdir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	t.Setenv("BIFROST_LISTEN_ADDR", "")
	t.Setenv("BIFROST_JWT_SECRET", "")
	t.Setenv("BIFROST_DATABASE_URL", "")

	cfg := Load()

	if cfg.ListenAddr != ":18080" {
		t.Fatalf("expected listen addr from .env, got %q", cfg.ListenAddr)
	}
	if cfg.JWTSecret != "env-secret" {
		t.Fatalf("expected jwt secret from .env, got %q", cfg.JWTSecret)
	}
	if cfg.DatabaseURL != "postgres://env-user:env-pass@127.0.0.1:6432/env-db?sslmode=disable" {
		t.Fatalf("expected database url from .env, got %q", cfg.DatabaseURL)
	}
}

func TestLoadPrefersProcessEnvAndDotEnvLocal(t *testing.T) {
	workdir := t.TempDir()
	writeTestFile(t, filepath.Join(workdir, ".env"), `
BIFROST_LISTEN_ADDR=:18080
BIFROST_AGENT_BACKEND_URL=http://from-env-file:8080
`)
	writeTestFile(t, filepath.Join(workdir, ".env.local"), `
BIFROST_LISTEN_ADDR=:19090
BIFROST_AGENT_BACKEND_URL=http://from-env-local:8080
`)

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(previous); chdirErr != nil {
			t.Fatalf("restore cwd: %v", chdirErr)
		}
	}()

	if err := os.Chdir(workdir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	t.Setenv("BIFROST_LISTEN_ADDR", ":17070")
	t.Setenv("BIFROST_AGENT_BACKEND_URL", "")

	cfg := Load()

	if cfg.ListenAddr != ":17070" {
		t.Fatalf("expected process env to win, got %q", cfg.ListenAddr)
	}
	if cfg.AgentBackendURL != "http://from-env-local:8080" {
		t.Fatalf("expected .env.local to override .env, got %q", cfg.AgentBackendURL)
	}
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
