package agent

import (
	"context"

	shareddb "github.com/dipesh/bifrost/backend/internal/shared/database"
)

type repository struct {
	store    Store
	timeouts shareddb.QueryTimeouts
}

func NewRepository(dataStore Store, timeouts shareddb.QueryTimeouts) Repository {
	return &repository{
		store:    dataStore,
		timeouts: timeouts,
	}
}

func (r *repository) AgentByAPIKey(ctx context.Context, apiKey string) (Agent, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() (Agent, error) {
		return r.store.AgentByAPIKey(apiKey)
	})
}

func (r *repository) UpdateAgentLastSeen(ctx context.Context, agentID string) error {
	return shareddb.WithTimeoutVoid(ctx, r.timeouts.Write, func() error {
		return r.store.UpdateAgentLastSeen(agentID)
	})
}

func (r *repository) SelfEnrollPendingAgent(ctx context.Context, agentID, serverID string) (Agent, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Write, func() (Agent, error) {
		return r.store.SelfEnrollPendingAgent(agentID, serverID)
	})
}

func (r *repository) Ingest(ctx context.Context, payload IngestPayload) error {
	return shareddb.WithTimeoutVoid(ctx, r.timeouts.Ingest, func() error {
		return r.store.Ingest(payload)
	})
}
