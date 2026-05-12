package install

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	sharedhttp "github.com/dipesh/bifrost/backend/internal/shared/http"
)

type Service struct {
	agentDockerImage string
	agentBinaryPath  string
	inspector        *sharedhttp.RequestInspector
}

func NewService(cfg Config, inspector *sharedhttp.RequestInspector) *Service {
	return &Service{
		agentDockerImage: cfg.AgentDockerImage,
		agentBinaryPath:  cfg.AgentBinaryPath,
		inspector:        inspector,
	}
}

func (s *Service) AgentDockerImage() string {
	return s.agentDockerImage
}

func (s *Service) DefaultBackendURL(request *http.Request) string {
	return s.inspector.DefaultBackendURL(request)
}

func (s *Service) InstallAgentBinaryURL(backendURL string) string {
	return strings.TrimRight(strings.TrimSpace(backendURL), "/") + "/api/v1/agent/install"
}

func (s *Service) BinaryDownloadEnabled() bool {
	return strings.TrimSpace(s.agentBinaryPath) != ""
}

func (s *Service) ResolveAgentBinaryPath() (string, error) {
	if binaryPath := strings.TrimSpace(s.agentBinaryPath); binaryPath != "" {
		if _, err := os.Stat(binaryPath); err != nil {
			return "", fmt.Errorf("read configured agent binary: %w", err)
		}
		return binaryPath, nil
	}
	return "", fmt.Errorf("agent binary serving is disabled")
}
