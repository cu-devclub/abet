package port

import (
	"context"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
)

// Dev-only "act as user" (D28). Every use case takes the REAL actor: only a real super_admin may use them.

// ListActAsUsersInput filters the list of users a super_admin can switch to.
type ListActAsUsersInput struct {
	Search string // case-insensitive substring of email or name
	Role   *domain.Role
}

// ListActAsUsersUseCase returns every user except the real one, sorted by role then email (max 500).
type ListActAsUsersUseCase interface {
	Execute(ctx context.Context, real *domain.Actor, in ListActAsUsersInput) ([]domain.User, error)
}

// StartActAsUseCase validates a switch to targetID (super_admin only, target must exist and differ from the real user).
type StartActAsUseCase interface {
	Execute(ctx context.Context, real *domain.Actor, targetID uuid.UUID) error
}

// ActAsState is the "act_as" block of GET /me.
type ActAsState struct {
	Available bool
	RealUser  *domain.User // set only while acting
}

// GetActAsStateUseCase resolves the act_as block for the real and the effective (acting) actor.
type GetActAsStateUseCase interface {
	Execute(ctx context.Context, real, effective *domain.Actor) (*ActAsState, error)
}
