package systemonboarding

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/dipesh/bifrost/backend/internal/domain"
)

var ErrInvalidName = errors.New("system name is required")
var ErrOnboardingNotPending = errors.New("system onboarding is no longer awaiting connection")

type repository interface {
	CreateSystemOnboarding(tenantID, createdByUserID, name, description string) (domain.SystemOnboarding, error)
	ListSystemOnboardings(tenantID string) ([]domain.SystemOnboarding, error)
	SystemOnboardingByID(tenantID, onboardingID string) (domain.SystemOnboarding, error)
	CancelSystemOnboarding(tenantID, onboardingID string) error
	ReissueSystemOnboardingCredentials(tenantID, onboardingID string) (domain.SystemOnboarding, error)
}

type Service struct {
	repo             repository
	agentBackendURL  string
	agentDockerImage string
}

type CreateInput struct {
	TenantID        string
	CreatedByUserID string
	Name            string
	Description     string
}

type View struct {
	ID          string     `json:"id"`
	ServerID    string     `json:"server_id"`
	AgentID     string     `json:"agent_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	ConnectedAt *time.Time `json:"connected_at,omitempty"`
}

type CreatedView struct {
	View
	APIKey                string `json:"api_key"`
	EnrollmentToken       string `json:"enrollment_token,omitempty"`
	BackendURL            string `json:"backend_url"`
	InstallScriptURL      string `json:"install_script_url"`
	DockerRunCommand      string `json:"docker_run_command"`
	SystemdInstallCommand string `json:"systemd_install_command"`
	ConfigYAML            string `json:"config_yaml"`
}

func New(repo repository, agentBackendURL, agentDockerImage string) *Service {
	return &Service{
		repo:             repo,
		agentBackendURL:  agentBackendURL,
		agentDockerImage: agentDockerImage,
	}
}

func (s *Service) Create(input CreateInput) (CreatedView, error) {
	name := strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)
	if name == "" {
		return CreatedView{}, ErrInvalidName
	}

	onboarding, err := s.repo.CreateSystemOnboarding(input.TenantID, input.CreatedByUserID, name, description)
	if err != nil {
		return CreatedView{}, err
	}

	return CreatedView{
		View:                  viewFromDomain(onboarding),
		APIKey:                onboarding.APIKey,
		EnrollmentToken:       onboarding.APIKey,
		BackendURL:            s.agentBackendURL,
		InstallScriptURL:      installScriptURL(s.agentBackendURL),
		DockerRunCommand:      renderDockerRunCommand(onboarding, s.agentBackendURL, s.agentDockerImage),
		SystemdInstallCommand: renderSystemdInstallCommand(onboarding, s.agentBackendURL),
		ConfigYAML:            renderAgentConfigYAML(onboarding, s.agentBackendURL),
	}, nil
}

func (s *Service) Get(tenantID, onboardingID string) (View, error) {
	onboarding, err := s.repo.SystemOnboardingByID(tenantID, onboardingID)
	if err != nil {
		return View{}, err
	}

	return viewFromDomain(onboarding), nil
}

func (s *Service) List(tenantID string) ([]View, error) {
	onboardings, err := s.repo.ListSystemOnboardings(tenantID)
	if err != nil {
		return nil, err
	}

	views := make([]View, 0, len(onboardings))
	for _, onboarding := range onboardings {
		views = append(views, viewFromDomain(onboarding))
	}

	return views, nil
}

func (s *Service) Cancel(tenantID, onboardingID string) error {
	onboarding, err := s.repo.SystemOnboardingByID(tenantID, onboardingID)
	if err != nil {
		return err
	}
	if onboarding.Status != "awaiting_connection" {
		return ErrOnboardingNotPending
	}

	return s.repo.CancelSystemOnboarding(tenantID, onboardingID)
}

func (s *Service) Reissue(tenantID, onboardingID string) (CreatedView, error) {
	onboarding, err := s.repo.SystemOnboardingByID(tenantID, onboardingID)
	if err != nil {
		return CreatedView{}, err
	}
	if onboarding.Status != "awaiting_connection" {
		return CreatedView{}, ErrOnboardingNotPending
	}

	reissued, err := s.repo.ReissueSystemOnboardingCredentials(tenantID, onboardingID)
	if err != nil {
		return CreatedView{}, err
	}

	return CreatedView{
		View:                  viewFromDomain(reissued),
		APIKey:                reissued.APIKey,
		EnrollmentToken:       reissued.APIKey,
		BackendURL:            s.agentBackendURL,
		InstallScriptURL:      installScriptURL(s.agentBackendURL),
		DockerRunCommand:      renderDockerRunCommand(reissued, s.agentBackendURL, s.agentDockerImage),
		SystemdInstallCommand: renderSystemdInstallCommand(reissued, s.agentBackendURL),
		ConfigYAML:            renderAgentConfigYAML(reissued, s.agentBackendURL),
	}, nil
}

func viewFromDomain(onboarding domain.SystemOnboarding) View {
	return View{
		ID:          onboarding.ID,
		ServerID:    onboarding.ServerID,
		AgentID:     onboarding.AgentID,
		Name:        onboarding.Name,
		Description: onboarding.Description,
		Status:      onboarding.Status,
		CreatedAt:   onboarding.CreatedAt,
		ConnectedAt: onboarding.ConnectedAt,
	}
}

// Keep the generated config aligned with the current manual agent install flow
// so the frontend can render static install steps without duplicating backend rules.
func renderAgentConfigYAML(onboarding domain.SystemOnboarding, backendURL string) string {
	lines := []string{
		"agent_id: " + strconv.Quote(onboarding.AgentID),
		"server_id: " + strconv.Quote(onboarding.ServerID),
		"server_name: " + strconv.Quote(onboarding.Name),
		"tenant_id: " + strconv.Quote(onboarding.TenantID),
		"backend_url: " + strconv.Quote(backendURL),
		"enrollment_token: " + strconv.Quote(onboarding.APIKey),
		"poll_interval_seconds: 10",
		"",
		"collectors:",
		"  host: true",
		"  docker: true",
		"  logs: true",
		"",
		"docker:",
		"  include_all: true",
		"  include_projects: []",
		"  include_containers: []",
		"  exclude_projects: []",
		"  exclude_containers:",
		"    - redis",
		"",
		"logs:",
		"  max_lines_per_fetch: 200",
	}
	return strings.Join(lines, "\n") + "\n"
}

func installScriptURL(backendURL string) string {
	return strings.TrimRight(strings.TrimSpace(backendURL), "/") + "/api/v1/install/agent.sh"
}

func renderDockerRunCommand(onboarding domain.SystemOnboarding, backendURL, dockerImage string) string {
	return strings.Join([]string{
		"docker run -d \\",
		"  --name bifrost-agent \\",
		"  --restart unless-stopped \\",
		"  --network host \\",
		"  -v bifrost-agent-data:/var/lib/bifrost-agent \\",
		"  -v /var/run/docker.sock:/var/run/docker.sock:ro \\",
		"  -e BIFROST_CONFIG_PATH=" + shellQuote("/var/lib/bifrost-agent/config.yaml") + " \\",
		"  -e BIFROST_AGENT_ID=" + shellQuote(onboarding.AgentID) + " \\",
		"  -e BIFROST_SERVER_ID=" + shellQuote(onboarding.ServerID) + " \\",
		"  -e BIFROST_SERVER_NAME=" + shellQuote(onboarding.Name) + " \\",
		"  -e BIFROST_TENANT_ID=" + shellQuote(onboarding.TenantID) + " \\",
		"  -e BIFROST_BACKEND_URL=" + shellQuote(backendURL) + " \\",
		"  -e BIFROST_ENROLLMENT_TOKEN=" + shellQuote(onboarding.APIKey) + " \\",
		"  " + shellQuote(dockerImage),
	}, "\n")
}

func renderSystemdInstallCommand(onboarding domain.SystemOnboarding, backendURL string) string {
	return strings.Join([]string{
		"curl -fsSL " + shellQuote(installScriptURL(backendURL)) + " | sudo env \\",
		"  BIFROST_AGENT_ID=" + shellQuote(onboarding.AgentID) + " \\",
		"  BIFROST_SERVER_ID=" + shellQuote(onboarding.ServerID) + " \\",
		"  BIFROST_SERVER_NAME=" + shellQuote(onboarding.Name) + " \\",
		"  BIFROST_TENANT_ID=" + shellQuote(onboarding.TenantID) + " \\",
		"  BIFROST_BACKEND_URL=" + shellQuote(backendURL) + " \\",
		"  BIFROST_ENROLLMENT_TOKEN=" + shellQuote(onboarding.APIKey) + " \\",
		"  sh",
	}, "\n")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
