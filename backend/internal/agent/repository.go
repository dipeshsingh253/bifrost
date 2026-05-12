package agent

import "context"

type Store interface {
	AgentByAPIKey(apiKey string) (Agent, error)
	UpdateAgentLastSeen(agentID string) error
	SelfEnrollPendingAgent(agentID, serverID string) (Agent, error)
	Ingest(payload IngestPayload) error
}

type Repository interface {
	AgentByAPIKey(ctx context.Context, apiKey string) (Agent, error)
	UpdateAgentLastSeen(ctx context.Context, agentID string) error
	SelfEnrollPendingAgent(ctx context.Context, agentID, serverID string) (Agent, error)
	Ingest(ctx context.Context, payload IngestPayload) error
}
