package auth

import "github.com/dipesh/bifrost/backend/internal/domain"

type User = domain.User
type UserRole = domain.UserRole
type ViewerInvite = domain.ViewerInvite

const (
	RoleOwner  = domain.RoleOwner
	RoleAdmin  = domain.RoleAdmin
	RoleMember = domain.RoleMember
	RoleViewer = domain.RoleViewer
)
