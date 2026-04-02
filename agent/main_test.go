package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/dipesh/bifrost/agent/internal/config"
)

func TestEnsureEnrollmentPersistsRotatedAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/agent/enroll" {
			t.Fatalf("expected enroll path, got %s", r.URL.Path)
		}
		if r.Header.Get("X-Agent-Key") != "bootstrap-token" {
			t.Fatalf("expected bootstrap token header")
		}
		var payload struct {
			AgentID  string `json:"agent_id"`
			ServerID string `json:"server_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode enroll request: %v", err)
		}
		if payload.AgentID != "cb15f927-6dbb-4c17-bd6b-d5a65f0e7d5b" || payload.ServerID != "97f40c28-84d8-4d22-bd2a-70116182b318" {
			t.Fatalf("unexpected enroll payload: %+v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"agent_id":"cb15f927-6dbb-4c17-bd6b-d5a65f0e7d5b","server_id":"97f40c28-84d8-4d22-bd2a-70116182b318","api_key":"real-agent-key"}}`))
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := config.Config{
		AgentID:             "cb15f927-6dbb-4c17-bd6b-d5a65f0e7d5b",
		ServerID:            "97f40c28-84d8-4d22-bd2a-70116182b318",
		ServerName:          "Validation VPS",
		BackendURL:          server.URL,
		EnrollmentToken:     "bootstrap-token",
		PollIntervalSeconds: 10,
	}
	if err := config.Save(configPath, cfg); err != nil {
		t.Fatalf("save bootstrap config: %v", err)
	}

	if err := ensureEnrollment(configPath, &cfg); err != nil {
		t.Fatalf("ensure enrollment: %v", err)
	}
	if cfg.APIKey != "real-agent-key" {
		t.Fatalf("expected rotated api key in memory, got %q", cfg.APIKey)
	}
	if cfg.EnrollmentToken != "" {
		t.Fatalf("expected enrollment token to be cleared after self-enrollment")
	}

	saved, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("reload saved config: %v", err)
	}
	if saved.APIKey != "real-agent-key" || saved.EnrollmentToken != "" {
		t.Fatalf("expected persisted api key and cleared enrollment token, got %+v", saved)
	}
}
