package agent

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) AgentByAPIKey(ctx context.Context, apiKey string) (Agent, error) {
	return s.repo.AgentByAPIKey(ctx, apiKey)
}

func (s *Service) UpdateAgentLastSeen(ctx context.Context, agentID string) error {
	return s.repo.UpdateAgentLastSeen(ctx, agentID)
}

func (s *Service) SelfEnrollPendingAgent(ctx context.Context, agentID, serverID string) (Agent, error) {
	return s.repo.SelfEnrollPendingAgent(ctx, agentID, serverID)
}

func (s *Service) Ingest(ctx context.Context, payload IngestPayload) error {
	return s.repo.Ingest(ctx, payload)
}
