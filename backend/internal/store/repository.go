package store

import "github.com/dipesh/bifrost/backend/internal/domain"

type Repository interface {
	BootstrapStatus() (bool, error)
	BootstrapAdmin(tenantName, name, email, password string) (domain.User, error)
	Authenticate(email, password string) (domain.User, error)
	UserByToken(token string) (domain.User, error)
	RevokeSession(token string) error
	UpdateUserName(userID, name string) (domain.User, error)
	ChangeUserPassword(userID, currentPassword, newPassword string) error
	TenantSummary(tenantID string) (domain.TenantSummary, error)
	ViewerAccess(tenantID string) (domain.ViewerAccess, error)
	CreateViewerInvite(tenantID, invitedByUserID, email string) (domain.ViewerInvite, error)
	CreateSystemOnboarding(tenantID, createdByUserID, name, description string) (domain.SystemOnboarding, error)
	ListSystemOnboardings(tenantID string) ([]domain.SystemOnboarding, error)
	SystemOnboardingByID(tenantID, onboardingID string) (domain.SystemOnboarding, error)
	CancelSystemOnboarding(tenantID, onboardingID string) error
	ReissueSystemOnboardingCredentials(tenantID, onboardingID string) (domain.SystemOnboarding, error)
	SelfEnrollPendingAgent(agentID, serverID string) (domain.Agent, error)
	InviteByToken(token string) (domain.ViewerInvite, error)
	AcceptViewerInvite(token, name, password string) (domain.User, error)
	RevokeViewerInvite(tenantID, inviteID string) error
	DisableViewer(tenantID, viewerUserID string) error
	DeleteViewer(tenantID, viewerUserID string) error
	ListServers(tenantID string) []domain.Server
	ServerByID(tenantID, serverID string) (domain.Server, error)
	ServicesByServer(tenantID, serverID string) []domain.Service
	ServiceByID(tenantID, serviceID string) (domain.Service, error)
	ProjectByID(tenantID, serverID, projectID string) (domain.Service, error)
	ProjectsByServer(tenantID, serverID string) []domain.Service
	StandaloneContainersByServer(tenantID, serverID string) []domain.Container
	MetricsByServer(serverID string) []domain.MetricSeries
	ContainerMetricsByServer(serverID string) domain.ContainerMetricBundle
	LogsByService(serviceID string) []domain.LogLine
	LogsByContainer(serviceID, containerID string) []domain.LogLine
	ServerBundle(tenantID, serverID string) (domain.ServerBundle, error)
	ContainerByID(tenantID, serverID, containerID string) (domain.Container, domain.Service, error)
	ProjectMetrics(tenantID, serverID, projectID string) (domain.ContainerMetricBundle, error)
	ContainerMetrics(tenantID, serverID, containerID string) (domain.ContainerMetricHistory, domain.Container, domain.Service, error)
	ProjectEvents(tenantID, serverID, projectID string) ([]domain.EventLog, domain.Service, error)
	ContainerEvents(tenantID, serverID, containerID string) ([]domain.EventLog, domain.Container, domain.Service, error)
	ContainerEnv(tenantID, serverID, containerID string) (map[string]string, domain.Container, domain.Service, error)
	EnrollAgent(agent domain.Agent) (domain.Agent, error)
	AgentByAPIKey(apiKey string) (domain.Agent, error)
	UpdateAgentLastSeen(agentID string) error
	Ingest(payload domain.IngestPayload) error
}
