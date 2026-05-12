package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	shareddb "github.com/dipesh/bifrost/backend/internal/shared/database"
)

type blockingStore struct{}

func (blockingStore) BootstrapStatus() (bool, error) {
	time.Sleep(50 * time.Millisecond)
	return false, nil
}

func (blockingStore) BootstrapAdmin(tenantName, name, email, password string) (User, error) {
	time.Sleep(50 * time.Millisecond)
	return User{}, nil
}

func (blockingStore) Authenticate(email, password string) (User, error) {
	time.Sleep(50 * time.Millisecond)
	return User{}, nil
}

func (blockingStore) UserByToken(token string) (User, error) {
	time.Sleep(50 * time.Millisecond)
	return User{}, nil
}

func (blockingStore) RevokeSession(token string) error {
	time.Sleep(50 * time.Millisecond)
	return nil
}

func (blockingStore) UpdateUserName(userID, name string) (User, error) {
	time.Sleep(50 * time.Millisecond)
	return User{}, nil
}

func (blockingStore) ChangeUserPassword(userID, currentPassword, newPassword string) error {
	time.Sleep(50 * time.Millisecond)
	return nil
}

func (blockingStore) InviteByToken(token string) (ViewerInvite, error) {
	time.Sleep(50 * time.Millisecond)
	return ViewerInvite{}, nil
}

func (blockingStore) AcceptViewerInvite(token, name, password string) (User, error) {
	time.Sleep(50 * time.Millisecond)
	return User{}, nil
}

func TestRepositoryReturnsContextCancellationImmediately(t *testing.T) {
	repo := NewRepository(blockingStore{}, shareddb.QueryTimeouts{
		Read:  time.Second,
		Write: time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := repo.BootstrapStatus(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestRepositoryReadTimeoutReturnsDeadlineExceeded(t *testing.T) {
	repo := NewRepository(blockingStore{}, shareddb.QueryTimeouts{
		Read:  10 * time.Millisecond,
		Write: time.Second,
	})

	_, err := repo.BootstrapStatus(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}
