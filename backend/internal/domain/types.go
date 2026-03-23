package domain

import (
	"encoding/json"
	"time"
)

type UserRole string

const (
	RoleOwner  UserRole = "owner"
	RoleAdmin  UserRole = "admin"
	RoleMember UserRole = "member"
	RoleViewer UserRole = "viewer"
)

type User struct {
	ID        string   `json:"id"`
	TenantID  string   `json:"tenant_id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Password  string   `json:"-"`
	Role      UserRole `json:"role"`
	AuthToken string   `json:"-"`
}

type TenantSummary struct {
	TenantID    string `json:"tenant_id"`
	TenantName  string `json:"tenant_name"`
	AdminCount  int    `json:"admin_count"`
	ViewerCount int    `json:"viewer_count"`
}

type ViewerInvite struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	Email           string     `json:"email"`
	Role            UserRole   `json:"role"`
	InvitedByUserID string     `json:"invited_by_user_id"`
	ExpiresAt       time.Time  `json:"expires_at"`
	CreatedAt       time.Time  `json:"created_at"`
	AcceptedAt      *time.Time `json:"accepted_at,omitempty"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
	InviteToken     string     `json:"invite_token,omitempty"`
	Status          string     `json:"status"`
}

type ViewerAccount struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Email      string     `json:"email"`
	Name       string     `json:"name"`
	Role       UserRole   `json:"role"`
	Status     string     `json:"status"`
	DisabledAt *time.Time `json:"disabled_at,omitempty"`
}

type ViewerAccess struct {
	Viewers []ViewerAccount `json:"viewers"`
	Invites []ViewerInvite  `json:"invites"`
}

type SystemOnboarding struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	ServerID        string     `json:"server_id"`
	AgentID         string     `json:"agent_id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	Status          string     `json:"status"`
	CreatedByUserID string     `json:"created_by_user_id"`
	CreatedAt       time.Time  `json:"created_at"`
	ConnectedAt     *time.Time `json:"connected_at,omitempty"`
	APIKey          string     `json:"api_key,omitempty"`
}

type Server struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	Name           string    `json:"name"`
	Hostname       string    `json:"hostname"`
	PublicIP       string    `json:"public_ip"`
	AgentVersion   string    `json:"agent_version"`
	Status         string    `json:"status"`
	LastSeenAt     time.Time `json:"last_seen_at"`
	UptimeSeconds  int64     `json:"uptime_seconds"`
	CPUUsagePct    float64   `json:"cpu_usage_pct"`
	MemoryUsagePct float64   `json:"memory_usage_pct"`
	DiskUsagePct   float64   `json:"disk_usage_pct"`
	NetworkRXMB    float64   `json:"network_rx_mb"`
	NetworkTXMB    float64   `json:"network_tx_mb"`
	LoadAverage    string    `json:"load_average"`
	OS             string    `json:"os"`
	Kernel         string    `json:"kernel"`
	CPUModel       string    `json:"cpu_model"`
	CPUCores       int       `json:"cpu_cores"`
	CPUThreads     int       `json:"cpu_threads"`
	TotalMemoryGB  float64   `json:"total_memory_gb"`
	TotalDiskGB    float64   `json:"total_disk_gb"`
}

type Service struct {
	ID               string      `json:"id"`
	TenantID         string      `json:"tenant_id"`
	ServerID         string      `json:"server_id"`
	Name             string      `json:"name"`
	ComposeProject   string      `json:"compose_project"`
	Status           string      `json:"status"`
	ContainerCount   int         `json:"container_count"`
	RestartCount     int         `json:"restart_count"`
	PublishedPorts   []string    `json:"published_ports"`
	Containers       []Container `json:"containers"`
	LastLogTimestamp time.Time   `json:"last_log_timestamp"`
}

type Container struct {
	ID           string    `json:"id"`
	ServiceID    string    `json:"service_id"`
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

type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type MetricSeries struct {
	Key    string        `json:"key"`
	Unit   string        `json:"unit"`
	Points []MetricPoint `json:"points"`
}

type ContainerMetricPoint struct {
	Timestamp time.Time          `json:"timestamp"`
	Values    map[string]float64 `json:"-"`
}

func (p ContainerMetricPoint) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"timestamp": p.Timestamp,
	}

	for name, value := range p.Values {
		payload[name] = value
	}

	return json.Marshal(payload)
}

type ContainerMetricBundle struct {
	CPU     []ContainerMetricPoint `json:"cpu"`
	Memory  []ContainerMetricPoint `json:"memory"`
	Network []ContainerMetricPoint `json:"network"`
}

type ContainerMetricHistory struct {
	CPU     []MetricPoint `json:"cpu"`
	Memory  []MetricPoint `json:"memory"`
	Network []MetricPoint `json:"network"`
}

type ServerBundle struct {
	Server           Server                `json:"server"`
	Services         []Service             `json:"services"`
	Metrics          []MetricSeries        `json:"metrics"`
	ContainerMetrics ContainerMetricBundle `json:"containerMetrics"`
}

type EventLog struct {
	ServiceID   string    `json:"-"`
	ContainerID string    `json:"-"`
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	EntityName  string    `json:"entityName"`
}

type LogLine struct {
	ID            string    `json:"id"`
	ServerID      string    `json:"server_id"`
	ServiceID     string    `json:"service_id"`
	ContainerID   string    `json:"container_id"`
	ContainerName string    `json:"containerName"`
	ServiceTag    string    `json:"serviceTag"`
	Level         string    `json:"level"`
	Message       string    `json:"message"`
	Timestamp     time.Time `json:"timestamp"`
}

type Agent struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	ServerID    string    `json:"server_id"`
	Name        string    `json:"name"`
	APIKey      string    `json:"api_key"`
	Version     string    `json:"version"`
	LastSeenAt  time.Time `json:"last_seen_at"`
	EnrolledAt  time.Time `json:"enrolled_at"`
	ServerName  string    `json:"server_name"`
	Hostname    string    `json:"hostname"`
	Description string    `json:"description"`
}

type IngestPayload struct {
	AgentID string          `json:"agent_id"`
	Server  ServerSnapshot  `json:"server"`
	Metrics []MetricPayload `json:"metrics"`
	Logs    []LogPayload    `json:"logs"`
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

type ServiceSnapshot struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	ComposeProject string              `json:"compose_project"`
	Status         string              `json:"status"`
	PublishedPorts []string            `json:"published_ports"`
	Containers     []ContainerSnapshot `json:"containers"`
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

type MetricPayload struct {
	ServerID  string        `json:"server_id"`
	ServiceID string        `json:"service_id"`
	Key       string        `json:"key"`
	Unit      string        `json:"unit"`
	Points    []MetricPoint `json:"points"`
}

type LogPayload struct {
	ServerID    string    `json:"server_id"`
	ServiceID   string    `json:"service_id"`
	ContainerID string    `json:"container_id"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
}
