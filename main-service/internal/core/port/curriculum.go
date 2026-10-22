package port

import (
	"context"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
)

// ---- outbound ----

// CurriculumRepository persists curricula. Returned curricula include their department Ref and Cover.
// Unique-code and referenced-row violations surface as domain.ErrConflict.
type CurriculumRepository interface {
	Create(ctx context.Context, c *domain.Curriculum) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Curriculum, error)
	List(ctx context.Context, departmentID *uuid.UUID) ([]domain.Curriculum, error)
	Update(ctx context.Context, c *domain.Curriculum) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ---- inbound ----

type CreateCurriculumInput struct {
	Code         string
	Name         string
	DepartmentID uuid.UUID
}

type UpdateCurriculumInput struct {
	ID           uuid.UUID
	Code         *string
	Name         *string
	DepartmentID *uuid.UUID
}

type CreateCurriculumUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, in CreateCurriculumInput) (*domain.Curriculum, error)
}

type UpdateCurriculumUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, in UpdateCurriculumInput) (*domain.Curriculum, error)
}

type DeleteCurriculumUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, id uuid.UUID) error
}

// ListCurriculaUseCase lists the curricula the actor may see (all for super_admin/admin, else their own or their department's).
type ListCurriculaUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, departmentID *uuid.UUID) ([]domain.Curriculum, error)
}
