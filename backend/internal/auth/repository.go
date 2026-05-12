package auth

import "context"

type Store interface {
	BootstrapStatus() (bool, error)
	BootstrapAdmin(tenantName, name, email, password string) (User, error)
	Authenticate(email, password string) (User, error)
	UserByToken(token string) (User, error)
	RevokeSession(token string) error
	UpdateUserName(userID, name string) (User, error)
	ChangeUserPassword(userID, currentPassword, newPassword string) error
	InviteByToken(token string) (ViewerInvite, error)
	AcceptViewerInvite(token, name, password string) (User, error)
}

type Repository interface {
	BootstrapStatus(ctx context.Context) (bool, error)
	BootstrapAdmin(ctx context.Context, tenantName, name, email, password string) (User, error)
	Authenticate(ctx context.Context, email, password string) (User, error)
	UserByToken(ctx context.Context, token string) (User, error)
	RevokeSession(ctx context.Context, token string) error
	UpdateUserName(ctx context.Context, userID, name string) (User, error)
	ChangeUserPassword(ctx context.Context, userID, currentPassword, newPassword string) error
	InviteByToken(ctx context.Context, token string) (ViewerInvite, error)
	AcceptViewerInvite(ctx context.Context, token, name, password string) (User, error)
}
