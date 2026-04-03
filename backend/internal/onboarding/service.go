package onboarding

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/dipesh/bifrost/backend/internal/domain"
)

var ErrInvalidName = errors.New("system name is required")
var ErrOnboardingNotPending = errors.New("system onboarding is no longer awaiting connection")

type Service struct {
	repo             Repository
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
	TenantID    string     `json:"tenant_id"`
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

func NewService(repo Repository, agentBackendURL, agentDockerImage string) *Service {
	return &Service{
		repo:             repo,
		agentBackendURL:  agentBackendURL,
		agentDockerImage: agentDockerImage,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (CreatedView, error) {
	name := strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)
	if name == "" {
		return CreatedView{}, ErrInvalidName
	}

	onboarding, err := s.repo.CreateSystemOnboarding(ctx, input.TenantID, input.CreatedByUserID, name, description)
	if err != nil {
		return CreatedView{}, err
	}

	return s.createdView(onboarding), nil
}

func (s *Service) Get(ctx context.Context, tenantID, onboardingID string) (View, error) {
	onboarding, err := s.repo.SystemOnboardingByID(ctx, tenantID, onboardingID)
	if err != nil {
		return View{}, err
	}

	return viewFromDomain(onboarding), nil
}

func (s *Service) List(ctx context.Context, tenantID string) ([]View, error) {
	onboardings, err := s.repo.ListSystemOnboardings(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	views := make([]View, 0, len(onboardings))
	for _, onboarding := range onboardings {
		views = append(views, viewFromDomain(onboarding))
	}

	return views, nil
}

func (s *Service) Cancel(ctx context.Context, tenantID, onboardingID string) error {
	onboarding, err := s.repo.SystemOnboardingByID(ctx, tenantID, onboardingID)
	if err != nil {
		return err
	}
	if onboarding.Status != "awaiting_connection" {
		return ErrOnboardingNotPending
	}

	return s.repo.CancelSystemOnboarding(ctx, tenantID, onboardingID)
}

func (s *Service) Reissue(ctx context.Context, tenantID, onboardingID string) (CreatedView, error) {
	onboarding, err := s.repo.SystemOnboardingByID(ctx, tenantID, onboardingID)
	if err != nil {
		return CreatedView{}, err
	}
	if onboarding.Status != "awaiting_connection" {
		return CreatedView{}, ErrOnboardingNotPending
	}

	reissued, err := s.repo.ReissueSystemOnboardingCredentials(ctx, tenantID, onboardingID)
	if err != nil {
		return CreatedView{}, err
	}

	return s.createdView(reissued), nil
}

func (s *Service) createdView(onboarding domain.SystemOnboarding) CreatedView {
	return CreatedView{
		View:                  viewFromDomain(onboarding),
		APIKey:                onboarding.APIKey,
		EnrollmentToken:       onboarding.APIKey,
		BackendURL:            s.agentBackendURL,
		InstallScriptURL:      installScriptURL(s.agentBackendURL),
		DockerRunCommand:      renderDockerRunCommand(onboarding, s.agentBackendURL, s.agentDockerImage),
		SystemdInstallCommand: renderSystemdInstallCommand(onboarding, s.agentBackendURL),
		ConfigYAML:            renderAgentConfigYAML(onboarding, s.agentBackendURL),
	}
}

func viewFromDomain(onboarding domain.SystemOnboarding) View {
	return View{
		ID:          onboarding.ID,
		TenantID:    onboarding.TenantID,
		ServerID:    onboarding.ServerID,
		AgentID:     onboarding.AgentID,
		Name:        onboarding.Name,
		Description: onboarding.Description,
		Status:      onboarding.Status,
		CreatedAt:   onboarding.CreatedAt,
		ConnectedAt: onboarding.ConnectedAt,
	}
}

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
		"  exclude_containers: []",
		"",
		"logs:",
		"  max_lines_per_fetch: 200",
	}
	return strings.Join(lines, "\n") + "\n"
}

func installScriptURL(backendURL string) string {
	return strings.TrimRight(strings.TrimSpace(backendURL), "/") + "/api/v1/agent/install.sh"
}

func renderDockerRunCommand(onboarding domain.SystemOnboarding, backendURL, dockerImage string) string {
	return strings.Join([]string{
		"docker run -d \\",
		"  --name bifrost-agent \\",
		"  --restart unless-stopped \\",
		"  --network host \\",
		"  --pid host \\",
		"  --uts host \\",
		"  -v /:/hostfs:ro \\",
		"  -v bifrost-agent-data:/var/lib/bifrost-agent \\",
		"  -v /var/run/docker.sock:/var/run/docker.sock:ro \\",
		"  -e BIFROST_CONFIG_PATH=" + shellQuote("/var/lib/bifrost-agent/config.yaml") + " \\",
		"  -e BIFROST_HOST_ROOT=" + shellQuote("/hostfs") + " \\",
		"  -e BIFROST_AGENT_ID=" + shellQuote(onboarding.AgentID) + " \\",
		"  -e BIFROST_SERVER_ID=" + shellQuote(onboarding.ServerID) + " \\",
		"  -e BIFROST_SERVER_NAME=" + shellQuote(onboarding.Name) + " \\",
		"  -e BIFROST_TENANT_ID=" + shellQuote(onboarding.TenantID) + " \\",
		"  -e BIFROST_BACKEND_URL=" + shellQuote(backendURL) + " \\",
		"  -e BIFROST_ENROLLMENT_TOKEN=" + shellQuote(onboarding.APIKey) + " \\",
		"  -e BIFROST_COLLECT_HOST='true' \\",
		"  -e BIFROST_COLLECT_DOCKER='true' \\",
		"  -e BIFROST_COLLECT_LOGS='true' \\",
		"  -e BIFROST_DOCKER_INCLUDE_ALL='true' \\",
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
		"  BIFROST_COLLECT_HOST='true' \\",
		"  BIFROST_COLLECT_DOCKER='true' \\",
		"  BIFROST_COLLECT_LOGS='true' \\",
		"  BIFROST_DOCKER_INCLUDE_ALL='true' \\",
		"  sh",
	}, "\n")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
