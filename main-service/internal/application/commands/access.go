package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// SetAccessHandler implements port.SetAccessUseCase (PUT /offerings/{id}/access/{user_id}).
type SetAccessHandler struct {
	offerings port.OfferingRepository
	access    port.AccessRepository
	users     port.UserRepository
}

var _ port.SetAccessUseCase = (*SetAccessHandler)(nil)

func NewSetAccessHandler(offerings port.OfferingRepository, access port.AccessRepository, users port.UserRepository) *SetAccessHandler {
	return &SetAccessHandler{offerings: offerings, access: access, users: users}
}

func (h *SetAccessHandler) Execute(ctx context.Context, actor *domain.Actor, in port.SetAccessInput) (*domain.AccessEntry, error) {
	off, err := h.offerings.GetByID(ctx, in.OfferingID)
	if err != nil {
		return nil, notFoundAs(err, "offering not found")
	}
	if !actor.CanManageCurriculum(off.Curriculum.ID) {
		return nil, domain.ErrForbidden
	}
	if !in.CanUpload && !in.CanDownload {
		return nil, fmt.Errorf("%w: at least one of can_upload and can_download must be true", domain.ErrInvalidInput)
	}
	target, err := h.users.GetByID(ctx, in.UserID)
	if err != nil {
		return nil, notFoundAs(err, "user not found")
	}
	if target.Role != domain.RoleInstructor && target.Role != domain.RoleStaff {
		return nil, fmt.Errorf("%w: access can only be granted to instructors and staff", domain.ErrInvalidInput)
	}
	if !actor.CanSeeUser(target.ScopeInfo()) {
		return nil, domain.ErrForbidden
	}
	if err := h.access.Upsert(ctx, off.ID, target.ID, in.CanUpload, in.CanDownload); err != nil {
		return nil, err
	}
	return &domain.AccessEntry{
		UserID: target.ID, Name: target.Name, Email: target.Email, Role: target.Role,
		CanUpload: in.CanUpload, CanDownload: in.CanDownload, Source: domain.AccessSourceManual,
	}, nil
}

// RemoveAccessHandler implements port.RemoveAccessUseCase.
type RemoveAccessHandler struct {
	offerings port.OfferingRepository
	access    port.AccessRepository
}

var _ port.RemoveAccessUseCase = (*RemoveAccessHandler)(nil)

func NewRemoveAccessHandler(offerings port.OfferingRepository, access port.AccessRepository) *RemoveAccessHandler {
	return &RemoveAccessHandler{offerings: offerings, access: access}
}

func (h *RemoveAccessHandler) Execute(ctx context.Context, actor *domain.Actor, offeringID, userID uuid.UUID) error {
	off, err := h.offerings.GetByID(ctx, offeringID)
	if err != nil {
		return notFoundAs(err, "offering not found")
	}
	if !actor.CanManageCurriculum(off.Curriculum.ID) {
		return domain.ErrForbidden
	}
	return notFoundAs(h.access.Delete(ctx, offeringID, userID), "access entry not found")
}
