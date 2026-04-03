package auth

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

func (r *repository) BootstrapStatus(ctx context.Context) (bool, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, r.store.BootstrapStatus)
}

func (r *repository) BootstrapAdmin(ctx context.Context, tenantName, name, email, password string) (User, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Write, func() (User, error) {
		return r.store.BootstrapAdmin(tenantName, name, email, password)
	})
}

func (r *repository) Authenticate(ctx context.Context, email, password string) (User, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Write, func() (User, error) {
		return r.store.Authenticate(email, password)
	})
}

func (r *repository) UserByToken(ctx context.Context, token string) (User, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() (User, error) {
		return r.store.UserByToken(token)
	})
}

func (r *repository) RevokeSession(ctx context.Context, token string) error {
	return shareddb.WithTimeoutVoid(ctx, r.timeouts.Write, func() error {
		return r.store.RevokeSession(token)
	})
}

func (r *repository) UpdateUserName(ctx context.Context, userID, name string) (User, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Write, func() (User, error) {
		return r.store.UpdateUserName(userID, name)
	})
}

func (r *repository) ChangeUserPassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	return shareddb.WithTimeoutVoid(ctx, r.timeouts.Write, func() error {
		return r.store.ChangeUserPassword(userID, currentPassword, newPassword)
	})
}

func (r *repository) InviteByToken(ctx context.Context, token string) (ViewerInvite, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() (ViewerInvite, error) {
		return r.store.InviteByToken(token)
	})
}

func (r *repository) AcceptViewerInvite(ctx context.Context, token, name, password string) (User, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Write, func() (User, error) {
		return r.store.AcceptViewerInvite(token, name, password)
	})
}
