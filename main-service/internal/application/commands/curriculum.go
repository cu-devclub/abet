package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// CoverStorageKey is the storage key of a curriculum cover PDF.
func CoverStorageKey(curriculumID uuid.UUID) string {
	return "covers/" + curriculumID.String() + ".pdf"
}

// CreateCurriculumHandler implements port.CreateCurriculumUseCase.
type CreateCurriculumHandler struct {
	repo  port.CurriculumRepository
	depts port.DepartmentRepository
}

var _ port.CreateCurriculumUseCase = (*CreateCurriculumHandler)(nil)

func NewCreateCurriculumHandler(repo port.CurriculumRepository, depts port.DepartmentRepository) *CreateCurriculumHandler {
	return &CreateCurriculumHandler{repo: repo, depts: depts}
}

func (h *CreateCurriculumHandler) Execute(ctx context.Context, actor *domain.Actor, in port.CreateCurriculumInput) (*domain.Curriculum, error) {
	if !actor.CanManageOrg() {
		return nil, domain.ErrForbidden
	}
	c, err := domain.NewCurriculum(in.Code, in.Name, in.DepartmentID)
	if err != nil {
		return nil, err
	}
	if _, err := h.depts.GetByID(ctx, in.DepartmentID); err != nil {
		return nil, invalidIfNotFound(err, "department not found")
	}
	if err := h.repo.Create(ctx, c); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return nil, fmt.Errorf("%w: curriculum code already exists", domain.ErrConflict)
		}
		return nil, err
	}
	return h.repo.GetByID(ctx, c.ID)
}

// UpdateCurriculumHandler implements port.UpdateCurriculumUseCase.
type UpdateCurriculumHandler struct {
	repo  port.CurriculumRepository
	depts port.DepartmentRepository
}

var _ port.UpdateCurriculumUseCase = (*UpdateCurriculumHandler)(nil)

func NewUpdateCurriculumHandler(repo port.CurriculumRepository, depts port.DepartmentRepository) *UpdateCurriculumHandler {
	return &UpdateCurriculumHandler{repo: repo, depts: depts}
}

func (h *UpdateCurriculumHandler) Execute(ctx context.Context, actor *domain.Actor, in port.UpdateCurriculumInput) (*domain.Curriculum, error) {
	if !actor.CanManageOrg() {
		return nil, domain.ErrForbidden
	}
	c, err := h.repo.GetByID(ctx, in.ID)
	if err != nil {
		return nil, notFoundAs(err, "curriculum not found")
	}
	if in.DepartmentID != nil && *in.DepartmentID != c.DepartmentID {
		if _, err := h.depts.GetByID(ctx, *in.DepartmentID); err != nil {
			return nil, invalidIfNotFound(err, "department not found")
		}
	}
	if err := c.Apply(in.Code, in.Name, in.DepartmentID); err != nil {
		return nil, err
	}
	if err := h.repo.Update(ctx, c); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return nil, fmt.Errorf("%w: curriculum code already exists", domain.ErrConflict)
		}
		return nil, notFoundAs(err, "curriculum not found")
	}
	return h.repo.GetByID(ctx, c.ID)
}

// DeleteCurriculumHandler implements port.DeleteCurriculumUseCase. It also removes the curriculum cover file.
type DeleteCurriculumHandler struct {
	repo    port.CurriculumRepository
	storage port.Storage
}

var _ port.DeleteCurriculumUseCase = (*DeleteCurriculumHandler)(nil)

func NewDeleteCurriculumHandler(repo port.CurriculumRepository, storage port.Storage) *DeleteCurriculumHandler {
	return &DeleteCurriculumHandler{repo: repo, storage: storage}
}

func (h *DeleteCurriculumHandler) Execute(ctx context.Context, actor *domain.Actor, id uuid.UUID) error {
	if !actor.CanManageOrg() {
		return domain.ErrForbidden
	}
	if err := h.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return fmt.Errorf("%w: curriculum is referenced by users or courses", domain.ErrConflict)
		}
		return notFoundAs(err, "curriculum not found")
	}
	// Best effort: a leftover cover file must not fail an already committed delete.
	_ = h.storage.Delete(ctx, CoverStorageKey(id))
	return nil
}
