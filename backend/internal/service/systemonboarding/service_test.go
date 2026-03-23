package systemonboarding

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/dipesh/bifrost/backend/internal/domain"
	"github.com/dipesh/bifrost/backend/internal/seed"
	"github.com/dipesh/bifrost/backend/internal/store"
)

func TestCreateRejectsBlankName(t *testing.T) {
	service := New(store.NewMemoryStore(seed.Data()), "https://bifrost.example.com", "bifrost-agent:latest")

	_, err := service.Create(CreateInput{
		TenantID:        seed.TenantIDDemo,
		CreatedByUserID: seed.UserIDOwner,
		Name:            "   ",
	})
	if !errors.Is(err, ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
}

func TestCreateGetAndListRoundTripSystemOnboardingView(t *testing.T) {
	service := New(store.NewMemoryStore(seed.Data()), "https://bifrost.example.com", "bifrost-agent:latest")

	created, err := service.Create(CreateInput{
		TenantID:        seed.TenantIDDemo,
		CreatedByUserID: seed.UserIDOwner,
		Name:            "  Production VPS  ",
		Description:     "  Primary app node  ",
	})
	if err != nil {
		t.Fatalf("create system onboarding: %v", err)
	}

	if created.Name != "Production VPS" || created.Description != "Primary app node" {
		t.Fatalf("expected trimmed onboarding metadata, got %+v", created.View)
	}
	if created.Status != "awaiting_connection" {
		t.Fatalf("expected awaiting_connection status, got %q", created.Status)
	}
	if created.BackendURL != "https://bifrost.example.com" {
		t.Fatalf("expected configured backend url, got %q", created.BackendURL)
	}
	if created.APIKey == "" || created.EnrollmentToken == "" {
		t.Fatalf("expected bootstrap credentials to be present")
	}
	if !looksLikeUUID(created.ID) || !looksLikeUUID(created.ServerID) || !looksLikeUUID(created.AgentID) {
		t.Fatalf("expected created onboarding to expose uuid identifiers, got %+v", created.View)
	}
	if !strings.Contains(created.ConfigYAML, `server_id: "`+created.ServerID+`"`) {
		t.Fatalf("expected config yaml to include server id")
	}
	if !strings.Contains(created.ConfigYAML, `agent_id: "`+created.AgentID+`"`) {
		t.Fatalf("expected config yaml to include agent id")
	}
	if !strings.Contains(created.ConfigYAML, `enrollment_token: "`+created.EnrollmentToken+`"`) {
		t.Fatalf("expected config yaml to include enrollment token")
	}
	if created.InstallScriptURL != "https://bifrost.example.com/api/v1/install/agent.sh" {
		t.Fatalf("expected install script url, got %q", created.InstallScriptURL)
	}
	if !strings.Contains(created.DockerRunCommand, "BIFROST_AGENT_ID='"+created.AgentID+"'") {
		t.Fatalf("expected docker command to include agent env")
	}
	if !strings.Contains(created.DockerRunCommand, "--pid host") || !strings.Contains(created.DockerRunCommand, "--uts host") {
		t.Fatalf("expected docker command to join the host pid and uts namespaces, got %q", created.DockerRunCommand)
	}
	if !strings.Contains(created.DockerRunCommand, "-v /:/hostfs:ro") || !strings.Contains(created.DockerRunCommand, "BIFROST_HOST_ROOT='/hostfs'") {
		t.Fatalf("expected docker command to mount the host filesystem for host metrics, got %q", created.DockerRunCommand)
	}
	if !strings.Contains(created.DockerRunCommand, "'bifrost-agent:latest'") {
		t.Fatalf("expected docker command to use the configured agent image, got %q", created.DockerRunCommand)
	}
	if !strings.Contains(created.SystemdInstallCommand, "curl -fsSL 'https://bifrost.example.com/api/v1/install/agent.sh'") {
		t.Fatalf("expected systemd command to use install script, got %q", created.SystemdInstallCommand)
	}
	if strings.Contains(created.SystemdInstallCommand, "BIFROST_AGENT_IMAGE=") {
		t.Fatalf("expected systemd command to stop requiring an explicit image")
	}

	detail, err := service.Get(seed.TenantIDDemo, created.ID)
	if err != nil {
		t.Fatalf("get system onboarding: %v", err)
	}
	if detail.ID != created.ID || detail.AgentID != created.AgentID {
		t.Fatalf("expected detail view to match created onboarding, got %+v", detail)
	}
	if !looksLikeUUID(detail.ID) || detail.ServerID != created.ServerID {
		t.Fatalf("expected detail view to preserve canonical ids, got %+v", detail)
	}
	if detail.ConnectedAt != nil {
		t.Fatalf("expected pending onboarding to have nil connected_at, got %+v", detail.ConnectedAt)
	}

	list, err := service.List(seed.TenantIDDemo)
	if err != nil {
		t.Fatalf("list system onboardings: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one onboarding in list, got %d", len(list))
	}
	if list[0].ID != created.ID || list[0].Name != "Production VPS" {
		t.Fatalf("expected list view to include created onboarding, got %+v", list[0])
	}
	if !looksLikeUUID(list[0].ID) || list[0].ServerID != created.ServerID {
		t.Fatalf("expected list view to include canonical ids, got %+v", list[0])
	}
}

func TestReissueKeepsPendingOnboardingIdentityAndRotatesCredentials(t *testing.T) {
	service := New(store.NewMemoryStore(seed.Data()), "https://bifrost.example.com", "bifrost-agent:latest")

	created, err := service.Create(CreateInput{
		TenantID:        seed.TenantIDDemo,
		CreatedByUserID: seed.UserIDOwner,
		Name:            "Validation VPS",
		Description:     "Pending node",
	})
	if err != nil {
		t.Fatalf("create system onboarding: %v", err)
	}

	reissued, err := service.Reissue(seed.TenantIDDemo, created.ID)
	if err != nil {
		t.Fatalf("reissue system onboarding: %v", err)
	}

	if reissued.ID != created.ID || reissued.ServerID != created.ServerID || reissued.AgentID != created.AgentID {
		t.Fatalf("expected reissue to keep onboarding identity stable, got %+v", reissued.View)
	}
	if reissued.APIKey == "" || reissued.EnrollmentToken == "" || reissued.APIKey == created.APIKey {
		t.Fatalf("expected reissue to rotate the bootstrap credential")
	}
	if !strings.Contains(reissued.ConfigYAML, `enrollment_token: "`+reissued.EnrollmentToken+`"`) {
		t.Fatalf("expected config yaml to include the rotated enrollment token")
	}
	if !strings.Contains(reissued.ConfigYAML, `agent_id: "`+reissued.AgentID+`"`) {
		t.Fatalf("expected config yaml to preserve the canonical agent id")
	}
	if !strings.Contains(reissued.SystemdInstallCommand, "BIFROST_ENROLLMENT_TOKEN='"+reissued.EnrollmentToken+"'") {
		t.Fatalf("expected systemd install command to include the rotated bootstrap token")
	}
}

func TestCancelRejectsConnectedOnboarding(t *testing.T) {
	store := store.NewMemoryStore(seed.Data())
	service := New(store, "https://bifrost.example.com", "bifrost-agent:latest")

	created, err := service.Create(CreateInput{
		TenantID:        seed.TenantIDDemo,
		CreatedByUserID: seed.UserIDOwner,
		Name:            "Connected VPS",
	})
	if err != nil {
		t.Fatalf("create system onboarding: %v", err)
	}

	if err := store.Ingest(domain.IngestPayload{
		AgentID: created.AgentID,
		Server: domain.ServerSnapshot{
			ID:          created.ServerID,
			Name:        "Connected VPS",
			Hostname:    "connected-vps",
			Status:      "up",
			CollectedAt: time.Date(2026, 3, 23, 8, 0, 0, 0, time.UTC),
		},
	}); err != nil {
		t.Fatalf("ingest system onboarding snapshot: %v", err)
	}

	err = service.Cancel(seed.TenantIDDemo, created.ID)
	if !errors.Is(err, ErrOnboardingNotPending) {
		t.Fatalf("expected ErrOnboardingNotPending, got %v", err)
	}
}

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func looksLikeUUID(value string) bool {
	return uuidPattern.MatchString(value)
}
