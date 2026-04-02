package auth

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) BootstrapStatus(ctx context.Context) (bool, error) {
	return s.repo.BootstrapStatus(ctx)
}

func (s *Service) BootstrapAdmin(ctx context.Context, tenantName, name, email, password string) (User, error) {
	return s.repo.BootstrapAdmin(ctx, tenantName, name, email, password)
}

func (s *Service) Authenticate(ctx context.Context, email, password string) (User, error) {
	return s.repo.Authenticate(ctx, email, password)
}

func (s *Service) UserByToken(ctx context.Context, token string) (User, error) {
	return s.repo.UserByToken(ctx, token)
}

func (s *Service) RevokeSession(ctx context.Context, token string) error {
	return s.repo.RevokeSession(ctx, token)
}

func (s *Service) UpdateUserName(ctx context.Context, userID, name string) (User, error) {
	return s.repo.UpdateUserName(ctx, userID, name)
}

func (s *Service) ChangeUserPassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	return s.repo.ChangeUserPassword(ctx, userID, currentPassword, newPassword)
}

func (s *Service) InviteByToken(ctx context.Context, token string) (ViewerInvite, error) {
	return s.repo.InviteByToken(ctx, token)
}

func (s *Service) AcceptViewerInvite(ctx context.Context, token, name, password string) (User, error) {
	return s.repo.AcceptViewerInvite(ctx, token, name, password)
}
