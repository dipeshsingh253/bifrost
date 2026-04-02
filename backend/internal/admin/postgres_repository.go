package admin

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

func (r *repository) TenantSummary(ctx context.Context, tenantID string) (TenantSummary, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() (TenantSummary, error) {
		return r.store.TenantSummary(tenantID)
	})
}

func (r *repository) ViewerAccess(ctx context.Context, tenantID string) (ViewerAccess, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() (ViewerAccess, error) {
		return r.store.ViewerAccess(tenantID)
	})
}

func (r *repository) CreateViewerInvite(ctx context.Context, tenantID, invitedByUserID, email string) (ViewerInvite, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Write, func() (ViewerInvite, error) {
		return r.store.CreateViewerInvite(tenantID, invitedByUserID, email)
	})
}

func (r *repository) RevokeViewerInvite(ctx context.Context, tenantID, inviteID string) error {
	return shareddb.WithTimeoutVoid(ctx, r.timeouts.Write, func() error {
		return r.store.RevokeViewerInvite(tenantID, inviteID)
	})
}

func (r *repository) DisableViewer(ctx context.Context, tenantID, viewerUserID string) error {
	return shareddb.WithTimeoutVoid(ctx, r.timeouts.Write, func() error {
		return r.store.DisableViewer(tenantID, viewerUserID)
	})
}

func (r *repository) DeleteViewer(ctx context.Context, tenantID, viewerUserID string) error {
	return shareddb.WithTimeoutVoid(ctx, r.timeouts.Write, func() error {
		return r.store.DeleteViewer(tenantID, viewerUserID)
	})
}
