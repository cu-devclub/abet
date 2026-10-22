package port

import (
	"context"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
)

// ---- outbound ----

// DepartmentRepository persists departments. Unique-code and referenced-row violations surface as domain.ErrConflict.
type DepartmentRepository interface {
	Create(ctx context.Context, d *domain.Department) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Department, error)
	List(ctx context.Context) ([]domain.Department, error)
	Update(ctx context.Context, d *domain.Department) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ---- inbound ----

type CreateDepartmentInput struct {
	Code string
	Name string
}

type UpdateDepartmentInput struct {
	ID   uuid.UUID
	Code *string
	Name *string
}

type CreateDepartmentUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, in CreateDepartmentInput) (*domain.Department, error)
}

type UpdateDepartmentUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, in UpdateDepartmentInput) (*domain.Department, error)
}

type DeleteDepartmentUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, id uuid.UUID) error
}

// ListDepartmentsUseCase lists the departments the actor may see (all for super_admin/admin, else their own).
type ListDepartmentsUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor) ([]domain.Department, error)
}
