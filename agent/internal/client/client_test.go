package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnrollReturnsRotatedAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST enroll request, got %s", r.Method)
		}
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

	apiKey, err := New(server.URL, "bootstrap-token").Enroll("cb15f927-6dbb-4c17-bd6b-d5a65f0e7d5b", "97f40c28-84d8-4d22-bd2a-70116182b318")
	if err != nil {
		t.Fatalf("enroll agent: %v", err)
	}
	if apiKey != "real-agent-key" {
		t.Fatalf("expected rotated api key, got %q", apiKey)
	}
}

func TestPushSnapshotUsesConfiguredAgentID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST snapshot request, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/agent/snapshot" {
			t.Fatalf("expected ingest path, got %s", r.URL.Path)
		}

		var payload struct {
			AgentID string `json:"agent_id"`
			Server  struct {
				ID string `json:"id"`
			} `json:"server"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode ingest payload: %v", err)
		}
		if payload.AgentID != "cb15f927-6dbb-4c17-bd6b-d5a65f0e7d5b" {
			t.Fatalf("expected configured agent id in payload, got %q", payload.AgentID)
		}
		if payload.Server.ID != "97f40c28-84d8-4d22-bd2a-70116182b318" {
			t.Fatalf("expected canonical server id in payload, got %q", payload.Server.ID)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	err := New(server.URL, "real-agent-key").PushSnapshot(
		"cb15f927-6dbb-4c17-bd6b-d5a65f0e7d5b",
		ServerSnapshot{ID: "97f40c28-84d8-4d22-bd2a-70116182b318"},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("push snapshot: %v", err)
	}
}
