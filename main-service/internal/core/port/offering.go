package port

import (
	"context"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
)

// ---- outbound ----

// OfferingFilter narrows OfferingRepository.List.
type OfferingFilter struct {
	AcademicYear int
	DepartmentID *uuid.UUID
	CurriculumID *uuid.UUID
	Semester     *int
	Search       string // case-insensitive substring of course code or name
	// Restrict limits the result to CurriculumScope (an admin scope). Restrict with an empty scope matches nothing.
	Restrict        bool
	CurriculumScope []uuid.UUID
}

type CreateOfferingParams struct {
	CourseCode   string
	CourseName   string
	CurriculumID uuid.UUID
	AcademicYear int
	Semester     int
}

// UpdateOfferingParams are the resolved (non-optional) target values of an offering.
type UpdateOfferingParams struct {
	ID           uuid.UUID
	CourseCode   string
	CourseName   string
	RenameCourse bool // overwrite the name of an already existing target course
	CurriculumID uuid.UUID
	AcademicYear int
	Semester     int
}

// OfferingRepository persists course offerings together with their course rows.
// A duplicate (course, year, semester) surfaces as domain.ErrConflict.
type OfferingRepository interface {
	List(ctx context.Context, f OfferingFilter) ([]domain.Offering, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Offering, error)
	// ListByIDs returns the offerings that exist (missing ids are simply absent).
	ListByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Offering, error)
	// Years returns the distinct academic years that have offerings, newest first.
	Years(ctx context.Context) ([]int, error)
	// Create upserts the course (curriculum, code; name updated) and inserts the offering in one transaction.
	Create(ctx context.Context, p CreateOfferingParams) (uuid.UUID, error)
	// Update moves/edits the offering in one transaction and deletes the old course if it became orphaned.
	Update(ctx context.Context, p UpdateOfferingParams) error
	// Delete removes the offerings (cascading submissions/access) and the courses left without offerings in one
	// transaction. It returns the ids of the deleted submissions so the caller can remove their files.
	Delete(ctx context.Context, ids []uuid.UUID) (submissionIDs []uuid.UUID, err error)
}

// ---- inbound ----

type ListOfferingsInput struct {
	AcademicYear int
	DepartmentID *uuid.UUID
	CurriculumID *uuid.UUID
	Semester     *int
	Search       string
}

type CreateOfferingInput struct {
	CourseCode   string
	CourseName   string
	CurriculumID uuid.UUID
	AcademicYear int
	Semester     int
}

type UpdateOfferingInput struct {
	ID           uuid.UUID
	CourseCode   *string
	CourseName   *string
	CurriculumID *uuid.UUID
	AcademicYear *int
	Semester     *int
}

type ListOfferingsUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, in ListOfferingsInput) ([]domain.Offering, error)
}

type ListOfferingYearsUseCase interface {
	// Execute returns the years (newest first, always including the current year) and the current CE year.
	Execute(ctx context.Context) (years []int, current int, err error)
}

type CreateOfferingUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, in CreateOfferingInput) (*domain.Offering, error)
}

type UpdateOfferingUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, in UpdateOfferingInput) (*domain.Offering, error)
}

type DeleteOfferingsUseCase interface {
	// Execute deletes one or many offerings (all-or-nothing authorization) and returns how many were deleted.
	Execute(ctx context.Context, actor *domain.Actor, ids []uuid.UUID) (int, error)
}
