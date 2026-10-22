package queries

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// ListOfferingsHandler implements port.ListOfferingsUseCase (Courses page: super_admin or admin).
type ListOfferingsHandler struct {
	repo port.OfferingRepository
}

var _ port.ListOfferingsUseCase = (*ListOfferingsHandler)(nil)

func NewListOfferingsHandler(repo port.OfferingRepository) *ListOfferingsHandler {
	return &ListOfferingsHandler{repo: repo}
}

func (h *ListOfferingsHandler) Execute(ctx context.Context, actor *domain.Actor, in port.ListOfferingsInput) ([]domain.Offering, error) {
	if !actor.CanManageCoursesAny() {
		return nil, domain.ErrForbidden
	}
	if err := domain.ValidateAcademicYear(in.AcademicYear); err != nil {
		return nil, err
	}
	if in.Semester != nil {
		if err := domain.ValidateSemester(*in.Semester); err != nil {
			return nil, err
		}
	}
	f := port.OfferingFilter{
		AcademicYear: in.AcademicYear, DepartmentID: in.DepartmentID, CurriculumID: in.CurriculumID,
		Semester: in.Semester, Search: in.Search,
	}
	if actor.Role != domain.RoleSuperAdmin {
		f.Restrict = true
		f.CurriculumScope = actor.ScopeCurriculumIDs
	}
	return h.repo.List(ctx, f)
}

// ListOfferingYearsHandler implements port.ListOfferingYearsUseCase (any authenticated caller).
type ListOfferingYearsHandler struct {
	repo port.OfferingRepository
	now  func() time.Time
}

var _ port.ListOfferingYearsUseCase = (*ListOfferingYearsHandler)(nil)

func NewListOfferingYearsHandler(repo port.OfferingRepository) *ListOfferingYearsHandler {
	return &ListOfferingYearsHandler{repo: repo, now: time.Now}
}

func (h *ListOfferingYearsHandler) Execute(ctx context.Context) ([]int, int, error) {
	years, err := h.repo.Years(ctx)
	if err != nil {
		return nil, 0, err
	}
	current := h.now().UTC().Year()
	found := false
	for _, y := range years {
		if y == current {
			found = true
			break
		}
	}
	if !found {
		years = append(years, current)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	return years, current, nil
}

// ListAccessHandler implements port.ListAccessUseCase: explicit rows plus the derived staff defaults (A3).
type ListAccessHandler struct {
	offerings port.OfferingRepository
	access    port.AccessRepository
}

var _ port.ListAccessUseCase = (*ListAccessHandler)(nil)

func NewListAccessHandler(offerings port.OfferingRepository, access port.AccessRepository) *ListAccessHandler {
	return &ListAccessHandler{offerings: offerings, access: access}
}

func (h *ListAccessHandler) Execute(ctx context.Context, actor *domain.Actor, offeringID uuid.UUID) ([]domain.AccessEntry, error) {
	off, err := h.offerings.GetByID(ctx, offeringID)
	if err != nil {
		return nil, accessNotFoundAs(err, "offering not found")
	}
	if !actor.CanManageCurriculum(off.Curriculum.ID) {
		return nil, domain.ErrForbidden
	}
	rows, err := h.access.List(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AccessEntry, 0, len(rows))
	for _, e := range rows {
		if e.Role == domain.RoleAdmin || e.Role == domain.RoleSuperAdmin {
			continue // admins manage everything in scope; they are never listed
		}
		out = append(out, e)
	}
	defaults, err := h.access.ListDefaultStaff(ctx, offeringID, off.Department.ID)
	if err != nil {
		return nil, err
	}
	reason := domain.DefaultAccessReason(off.Department.Code)
	for _, s := range defaults {
		r := reason
		out = append(out, domain.AccessEntry{
			UserID: s.UserID, Name: s.Name, Email: s.Email, Role: domain.RoleStaff,
			CanUpload: false, CanDownload: true, IsDefault: true, DefaultReason: &r, Source: domain.AccessSourceDefault,
		})
	}
	return out, nil
}

func accessNotFoundAs(err error, msg string) error {
	if errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("%w: %s", domain.ErrNotFound, msg)
	}
	return err
}
