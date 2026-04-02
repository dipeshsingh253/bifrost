package onboarding

import (
	"context"

	"github.com/dipesh/bifrost/backend/internal/domain"
)

type Store interface {
	CreateSystemOnboarding(tenantID, createdByUserID, name, description string) (domain.SystemOnboarding, error)
	ListSystemOnboardings(tenantID string) ([]domain.SystemOnboarding, error)
	SystemOnboardingByID(tenantID, onboardingID string) (domain.SystemOnboarding, error)
	CancelSystemOnboarding(tenantID, onboardingID string) error
	ReissueSystemOnboardingCredentials(tenantID, onboardingID string) (domain.SystemOnboarding, error)
}

type Repository interface {
	CreateSystemOnboarding(ctx context.Context, tenantID, createdByUserID, name, description string) (domain.SystemOnboarding, error)
	ListSystemOnboardings(ctx context.Context, tenantID string) ([]domain.SystemOnboarding, error)
	SystemOnboardingByID(ctx context.Context, tenantID, onboardingID string) (domain.SystemOnboarding, error)
	CancelSystemOnboarding(ctx context.Context, tenantID, onboardingID string) error
	ReissueSystemOnboardingCredentials(ctx context.Context, tenantID, onboardingID string) (domain.SystemOnboarding, error)
}
