package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type enrollResponse struct {
	Success bool `json:"success"`
	Data    struct {
		AgentID  string `json:"agent_id"`
		ServerID string `json:"server_id"`
		APIKey   string `json:"api_key"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type MetricPayload struct {
	ServerID  string        `json:"server_id"`
	ServiceID string        `json:"service_id"`
	Key       string        `json:"key"`
	Unit      string        `json:"unit"`
	Points    []MetricPoint `json:"points"`
}

type ContainerSnapshot struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Image        string    `json:"image"`
	Status       string    `json:"status"`
	Health       string    `json:"health"`
	CPUUsagePct  float64   `json:"cpu_usage_pct"`
	MemoryMB     float64   `json:"memory_mb"`
	NetworkMB    float64   `json:"network_mb"`
	RestartCount int       `json:"restart_count"`
	Uptime       string    `json:"uptime"`
	Ports        []string  `json:"ports"`
	Command      string    `json:"command"`
	LastSeenAt   time.Time `json:"last_seen_at"`
}

type ServiceSnapshot struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	ComposeProject string              `json:"compose_project"`
	Status         string              `json:"status"`
	PublishedPorts []string            `json:"published_ports"`
	Containers     []ContainerSnapshot `json:"containers"`
}

type ServerSnapshot struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Hostname       string            `json:"hostname"`
	PublicIP       string            `json:"public_ip"`
	AgentVersion   string            `json:"agent_version"`
	Status         string            `json:"status"`
	UptimeSeconds  int64             `json:"uptime_seconds"`
	CPUUsagePct    float64           `json:"cpu_usage_pct"`
	MemoryUsagePct float64           `json:"memory_usage_pct"`
	DiskUsagePct   float64           `json:"disk_usage_pct"`
	NetworkRXMB    float64           `json:"network_rx_mb"`
	NetworkTXMB    float64           `json:"network_tx_mb"`
	LoadAverage    string            `json:"load_average"`
	OS             string            `json:"os"`
	Kernel         string            `json:"kernel"`
	CPUModel       string            `json:"cpu_model"`
	CPUCores       int               `json:"cpu_cores"`
	CPUThreads     int               `json:"cpu_threads"`
	TotalMemoryGB  float64           `json:"total_memory_gb"`
	TotalDiskGB    float64           `json:"total_disk_gb"`
	Services       []ServiceSnapshot `json:"services"`
	CollectedAt    time.Time         `json:"collected_at"`
}

type LogPayload struct {
	ServerID    string    `json:"server_id"`
	ServiceID   string    `json:"service_id"`
	ContainerID string    `json:"container_id"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) Enroll(agentID, serverID string) (string, error) {
	payload := map[string]string{
		"agent_id":  agentID,
		"server_id": serverID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/agents/enroll", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Agent-Key", c.apiKey)

	response, err := c.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	var payloadResponse enrollResponse
	if err := json.NewDecoder(response.Body).Decode(&payloadResponse); err != nil {
		return "", err
	}
	if response.StatusCode >= 300 || !payloadResponse.Success || payloadResponse.Data.APIKey == "" {
		if payloadResponse.Error != nil && payloadResponse.Error.Message != "" {
			return "", fmt.Errorf("%s", payloadResponse.Error.Message)
		}
		return "", fmt.Errorf("unexpected status %d", response.StatusCode)
	}
	if payloadResponse.Data.AgentID != agentID || payloadResponse.Data.ServerID != serverID {
		return "", fmt.Errorf("enrollment response identity mismatch")
	}

	return payloadResponse.Data.APIKey, nil
}

func (c *Client) PushSnapshot(agentID string, server ServerSnapshot, metrics []MetricPayload, logs []LogPayload) error {
	payload := map[string]any{
		"agent_id": agentID,
		"server":   server,
		"metrics":  metrics,
		"logs":     logs,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/ingest/snapshot", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Agent-Key", c.apiKey)

	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", response.StatusCode)
	}

	return nil
}
