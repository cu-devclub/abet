package queries

import (
	"context"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// ListCurriculaHandler implements port.ListCurriculaUseCase (any authenticated caller).
// super_admin and admin see every curriculum; an instructor or staff user only their own curriculum, or
// (without one) the curricula of their department.
type ListCurriculaHandler struct {
	repo port.CurriculumRepository
}

var _ port.ListCurriculaUseCase = (*ListCurriculaHandler)(nil)

func NewListCurriculaHandler(repo port.CurriculumRepository) *ListCurriculaHandler {
	return &ListCurriculaHandler{repo: repo}
}

func (h *ListCurriculaHandler) Execute(ctx context.Context, actor *domain.Actor, departmentID *uuid.UUID) ([]domain.Curriculum, error) {
	if seesAllOrganisation(actor) {
		return h.repo.List(ctx, departmentID)
	}
	if actor.CurriculumID == nil && actor.DepartmentID == nil {
		return []domain.Curriculum{}, nil
	}
	// Own department only; the optional department filter can only narrow that further.
	scope := actor.DepartmentID
	if scope == nil {
		scope = departmentID
	}
	all, err := h.repo.List(ctx, scope)
	if err != nil {
		return nil, err
	}
	out := []domain.Curriculum{}
	for _, c := range all {
		if actor.CurriculumID != nil && c.ID != *actor.CurriculumID {
			continue
		}
		if departmentID != nil && c.DepartmentID != *departmentID {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}
