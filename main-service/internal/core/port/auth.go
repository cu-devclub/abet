package port

import (
	"context"

	"abet/main-service/internal/core/domain"
)

// LoginSyncInput is the body auth-service posts after a successful Google login.
type LoginSyncInput struct {
	Email      string
	Name       string
	PictureURL string
}

// LoginSyncUseCase records a login for a pre-provisioned user; ErrForbidden when the email is not provisioned.
type LoginSyncUseCase interface {
	Execute(ctx context.Context, in LoginSyncInput) (*domain.User, error)
}

// BootstrapSuperAdminsUseCase upserts the SUPER_ADMIN_EMAILS list at startup and returns how many were processed.
type BootstrapSuperAdminsUseCase interface {
	Execute(ctx context.Context, emails []string) (int, error)
}
