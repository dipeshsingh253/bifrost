package onboarding

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

func (r *repository) CreateSystemOnboarding(ctx context.Context, tenantID, createdByUserID, name, description string) (SystemOnboarding, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Write, func() (SystemOnboarding, error) {
		return r.store.CreateSystemOnboarding(tenantID, createdByUserID, name, description)
	})
}

func (r *repository) ListSystemOnboardings(ctx context.Context, tenantID string) ([]SystemOnboarding, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() ([]SystemOnboarding, error) {
		return r.store.ListSystemOnboardings(tenantID)
	})
}

func (r *repository) SystemOnboardingByID(ctx context.Context, tenantID, onboardingID string) (SystemOnboarding, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Read, func() (SystemOnboarding, error) {
		return r.store.SystemOnboardingByID(tenantID, onboardingID)
	})
}

func (r *repository) CancelSystemOnboarding(ctx context.Context, tenantID, onboardingID string) error {
	return shareddb.WithTimeoutVoid(ctx, r.timeouts.Write, func() error {
		return r.store.CancelSystemOnboarding(tenantID, onboardingID)
	})
}

func (r *repository) ReissueSystemOnboardingCredentials(ctx context.Context, tenantID, onboardingID string) (SystemOnboarding, error) {
	return shareddb.WithTimeout(ctx, r.timeouts.Write, func() (SystemOnboarding, error) {
		return r.store.ReissueSystemOnboardingCredentials(tenantID, onboardingID)
	})
}
