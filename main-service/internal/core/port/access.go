package port

import (
	"context"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
)

// ---- outbound ----

// DefaultStaff is a staff user that gets the department default download on an offering.
type DefaultStaff struct {
	UserID uuid.UUID
	Name   string
	Email  string
}

// AccessRepository persists course_access rows.
type AccessRepository interface {
	// List returns the explicit rows of an offering (instructors first, then staff; by e-mail).
	List(ctx context.Context, offeringID uuid.UUID) ([]domain.AccessEntry, error)
	// ListDefaultStaff returns the staff users of the department that have no explicit row on the offering.
	ListDefaultStaff(ctx context.Context, offeringID, departmentID uuid.UUID) ([]DefaultStaff, error)
	// Upsert writes the row with source "manual".
	Upsert(ctx context.Context, offeringID, userID uuid.UUID, canUpload, canDownload bool) error
	// Delete removes the explicit row; domain.ErrNotFound if there was none.
	Delete(ctx context.Context, offeringID, userID uuid.UUID) error
}

// ---- inbound ----

type SetAccessInput struct {
	OfferingID  uuid.UUID
	UserID      uuid.UUID
	CanUpload   bool
	CanDownload bool
}

type ListAccessUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, offeringID uuid.UUID) ([]domain.AccessEntry, error)
}

type SetAccessUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, in SetAccessInput) (*domain.AccessEntry, error)
}

type RemoveAccessUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, offeringID, userID uuid.UUID) error
}
