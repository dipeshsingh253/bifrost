package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	shareddb "github.com/dipesh/bifrost/backend/internal/shared/database"
)

type blockingStore struct{}

func (blockingStore) AgentByAPIKey(apiKey string) (Agent, error) {
	time.Sleep(50 * time.Millisecond)
	return Agent{}, nil
}

func (blockingStore) UpdateAgentLastSeen(agentID string) error {
	time.Sleep(50 * time.Millisecond)
	return nil
}

func (blockingStore) SelfEnrollPendingAgent(agentID, serverID string) (Agent, error) {
	time.Sleep(50 * time.Millisecond)
	return Agent{}, nil
}

func (blockingStore) Ingest(payload IngestPayload) error {
	time.Sleep(50 * time.Millisecond)
	return nil
}

func TestRepositoryIngestTimeoutReturnsDeadlineExceeded(t *testing.T) {
	repo := NewRepository(blockingStore{}, shareddb.QueryTimeouts{
		Read:   time.Second,
		Write:  time.Second,
		Ingest: 10 * time.Millisecond,
	})

	err := repo.Ingest(context.Background(), IngestPayload{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}
