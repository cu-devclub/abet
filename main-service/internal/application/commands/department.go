package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// CreateDepartmentHandler implements port.CreateDepartmentUseCase.
type CreateDepartmentHandler struct {
	repo port.DepartmentRepository
}

var _ port.CreateDepartmentUseCase = (*CreateDepartmentHandler)(nil)

func NewCreateDepartmentHandler(repo port.DepartmentRepository) *CreateDepartmentHandler {
	return &CreateDepartmentHandler{repo: repo}
}

func (h *CreateDepartmentHandler) Execute(ctx context.Context, actor *domain.Actor, in port.CreateDepartmentInput) (*domain.Department, error) {
	if !actor.CanManageOrg() {
		return nil, domain.ErrForbidden
	}
	d, err := domain.NewDepartment(in.Code, in.Name)
	if err != nil {
		return nil, err
	}
	if err := h.repo.Create(ctx, d); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return nil, fmt.Errorf("%w: department code already exists", domain.ErrConflict)
		}
		return nil, err
	}
	return d, nil
}

// UpdateDepartmentHandler implements port.UpdateDepartmentUseCase.
type UpdateDepartmentHandler struct {
	repo port.DepartmentRepository
}

var _ port.UpdateDepartmentUseCase = (*UpdateDepartmentHandler)(nil)

func NewUpdateDepartmentHandler(repo port.DepartmentRepository) *UpdateDepartmentHandler {
	return &UpdateDepartmentHandler{repo: repo}
}

func (h *UpdateDepartmentHandler) Execute(ctx context.Context, actor *domain.Actor, in port.UpdateDepartmentInput) (*domain.Department, error) {
	if !actor.CanManageOrg() {
		return nil, domain.ErrForbidden
	}
	d, err := h.repo.GetByID(ctx, in.ID)
	if err != nil {
		return nil, notFoundAs(err, "department not found")
	}
	if err := d.Apply(in.Code, in.Name); err != nil {
		return nil, err
	}
	if err := h.repo.Update(ctx, d); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return nil, fmt.Errorf("%w: department code already exists", domain.ErrConflict)
		}
		return nil, notFoundAs(err, "department not found")
	}
	return d, nil
}

// DeleteDepartmentHandler implements port.DeleteDepartmentUseCase.
type DeleteDepartmentHandler struct {
	repo port.DepartmentRepository
}

var _ port.DeleteDepartmentUseCase = (*DeleteDepartmentHandler)(nil)

func NewDeleteDepartmentHandler(repo port.DepartmentRepository) *DeleteDepartmentHandler {
	return &DeleteDepartmentHandler{repo: repo}
}

func (h *DeleteDepartmentHandler) Execute(ctx context.Context, actor *domain.Actor, id uuid.UUID) error {
	if !actor.CanManageOrg() {
		return domain.ErrForbidden
	}
	if err := h.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return fmt.Errorf("%w: department is referenced by curricula or users", domain.ErrConflict)
		}
		return notFoundAs(err, "department not found")
	}
	return nil
}
