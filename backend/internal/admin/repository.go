package admin

import "context"

type Store interface {
	TenantSummary(tenantID string) (TenantSummary, error)
	ViewerAccess(tenantID string) (ViewerAccess, error)
	CreateViewerInvite(tenantID, invitedByUserID, email string) (ViewerInvite, error)
	RevokeViewerInvite(tenantID, inviteID string) error
	DisableViewer(tenantID, viewerUserID string) error
	DeleteViewer(tenantID, viewerUserID string) error
}

type Repository interface {
	TenantSummary(ctx context.Context, tenantID string) (TenantSummary, error)
	ViewerAccess(ctx context.Context, tenantID string) (ViewerAccess, error)
	CreateViewerInvite(ctx context.Context, tenantID, invitedByUserID, email string) (ViewerInvite, error)
	RevokeViewerInvite(ctx context.Context, tenantID, inviteID string) error
	DisableViewer(ctx context.Context, tenantID, viewerUserID string) error
	DeleteViewer(ctx context.Context, tenantID, viewerUserID string) error
}
