package admin

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) TenantSummary(ctx context.Context, tenantID string) (TenantSummary, error) {
	return s.repo.TenantSummary(ctx, tenantID)
}

func (s *Service) ViewerAccess(ctx context.Context, tenantID string) (ViewerAccess, error) {
	return s.repo.ViewerAccess(ctx, tenantID)
}

func (s *Service) CreateViewerInvite(ctx context.Context, tenantID, invitedByUserID, email string) (ViewerInvite, error) {
	return s.repo.CreateViewerInvite(ctx, tenantID, invitedByUserID, email)
}

func (s *Service) RevokeViewerInvite(ctx context.Context, tenantID, inviteID string) error {
	return s.repo.RevokeViewerInvite(ctx, tenantID, inviteID)
}

func (s *Service) DisableViewer(ctx context.Context, tenantID, viewerUserID string) error {
	return s.repo.DisableViewer(ctx, tenantID, viewerUserID)
}

func (s *Service) DeleteViewer(ctx context.Context, tenantID, viewerUserID string) error {
	return s.repo.DeleteViewer(ctx, tenantID, viewerUserID)
}
