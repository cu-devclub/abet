package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// CreateOfferingHandler implements port.CreateOfferingUseCase.
type CreateOfferingHandler struct {
	repo  port.OfferingRepository
	currs port.CurriculumRepository
}

var _ port.CreateOfferingUseCase = (*CreateOfferingHandler)(nil)

func NewCreateOfferingHandler(repo port.OfferingRepository, currs port.CurriculumRepository) *CreateOfferingHandler {
	return &CreateOfferingHandler{repo: repo, currs: currs}
}

func (h *CreateOfferingHandler) Execute(ctx context.Context, actor *domain.Actor, in port.CreateOfferingInput) (*domain.Offering, error) {
	if !actor.CanManageCurriculum(in.CurriculumID) {
		return nil, domain.ErrForbidden
	}
	code, name, err := domain.NormalizeCourse(in.CourseCode, in.CourseName)
	if err != nil {
		return nil, err
	}
	if err := domain.ValidateAcademicYear(in.AcademicYear); err != nil {
		return nil, err
	}
	if err := domain.ValidateSemester(in.Semester); err != nil {
		return nil, err
	}
	if _, err := h.currs.GetByID(ctx, in.CurriculumID); err != nil {
		return nil, invalidIfNotFound(err, "curriculum not found")
	}
	id, err := h.repo.Create(ctx, port.CreateOfferingParams{
		CourseCode: code, CourseName: name, CurriculumID: in.CurriculumID, AcademicYear: in.AcademicYear, Semester: in.Semester,
	})
	if err != nil {
		return nil, conflictAs(err, "offering already exists for this course, year and semester")
	}
	return h.repo.GetByID(ctx, id)
}

// UpdateOfferingHandler implements port.UpdateOfferingUseCase.
type UpdateOfferingHandler struct {
	repo  port.OfferingRepository
	currs port.CurriculumRepository
}

var _ port.UpdateOfferingUseCase = (*UpdateOfferingHandler)(nil)

func NewUpdateOfferingHandler(repo port.OfferingRepository, currs port.CurriculumRepository) *UpdateOfferingHandler {
	return &UpdateOfferingHandler{repo: repo, currs: currs}
}

func (h *UpdateOfferingHandler) Execute(ctx context.Context, actor *domain.Actor, in port.UpdateOfferingInput) (*domain.Offering, error) {
	cur, err := h.repo.GetByID(ctx, in.ID)
	if err != nil {
		return nil, notFoundAs(err, "offering not found")
	}
	if !actor.CanManageCurriculum(cur.Curriculum.ID) {
		return nil, domain.ErrForbidden
	}

	p := port.UpdateOfferingParams{
		ID: cur.ID, CourseCode: cur.CourseCode, CourseName: cur.CourseName, RenameCourse: in.CourseName != nil,
		CurriculumID: cur.Curriculum.ID, AcademicYear: cur.AcademicYear, Semester: cur.Semester,
	}
	if in.CourseCode != nil {
		p.CourseCode = *in.CourseCode
	}
	if in.CourseName != nil {
		p.CourseName = *in.CourseName
	}
	if in.AcademicYear != nil {
		p.AcademicYear = *in.AcademicYear
	}
	if in.Semester != nil {
		p.Semester = *in.Semester
	}
	if in.CurriculumID != nil && *in.CurriculumID != cur.Curriculum.ID {
		if !actor.CanManageCurriculum(*in.CurriculumID) {
			return nil, domain.ErrForbidden
		}
		if _, err := h.currs.GetByID(ctx, *in.CurriculumID); err != nil {
			return nil, invalidIfNotFound(err, "curriculum not found")
		}
		p.CurriculumID = *in.CurriculumID
	}
	if p.CourseCode, p.CourseName, err = domain.NormalizeCourse(p.CourseCode, p.CourseName); err != nil {
		return nil, err
	}
	if err := domain.ValidateAcademicYear(p.AcademicYear); err != nil {
		return nil, err
	}
	if err := domain.ValidateSemester(p.Semester); err != nil {
		return nil, err
	}

	if err := h.repo.Update(ctx, p); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return nil, conflictAs(err, "offering already exists for this course, year and semester")
		}
		return nil, notFoundAs(err, "offering not found")
	}
	return h.repo.GetByID(ctx, p.ID)
}

// DeleteOfferingsHandler implements port.DeleteOfferingsUseCase (single and bulk delete).
// After the transaction committed it removes the submission files of the deleted offerings (best effort).
type DeleteOfferingsHandler struct {
	repo    port.OfferingRepository
	storage port.Storage
}

var _ port.DeleteOfferingsUseCase = (*DeleteOfferingsHandler)(nil)

func NewDeleteOfferingsHandler(repo port.OfferingRepository, storage port.Storage) *DeleteOfferingsHandler {
	return &DeleteOfferingsHandler{repo: repo, storage: storage}
}

func (h *DeleteOfferingsHandler) Execute(ctx context.Context, actor *domain.Actor, ids []uuid.UUID) (int, error) {
	ids = uniqueIDs(ids)
	if len(ids) == 0 {
		return 0, fmt.Errorf("%w: ids cannot be empty", domain.ErrInvalidInput)
	}
	found, err := h.repo.ListByIDs(ctx, ids)
	if err != nil {
		return 0, err
	}
	// All-or-nothing authorization: one offering outside the actor scope rejects the whole request.
	for i := range found {
		if !actor.CanManageCurriculum(found[i].Curriculum.ID) {
			return 0, domain.ErrForbidden
		}
	}
	if len(found) != len(ids) {
		return 0, fmt.Errorf("%w: offering not found", domain.ErrNotFound)
	}

	submissionIDs, err := h.repo.Delete(ctx, ids)
	if err != nil {
		return 0, notFoundAs(err, "offering not found")
	}
	for _, sid := range submissionIDs {
		h.removeSubmissionFiles(ctx, sid)
	}
	return len(ids), nil
}

func (h *DeleteOfferingsHandler) removeSubmissionFiles(ctx context.Context, sid uuid.UUID) {
	if err := h.storage.Delete(ctx, submissionUploadKey(sid)); err != nil {
		slog.WarnContext(ctx, "delete submission upload failed", "submission_id", sid, "error", err)
	}
	if err := h.storage.Delete(ctx, submissionPDFKey(sid)); err != nil {
		slog.WarnContext(ctx, "delete submission portfolio failed", "submission_id", sid, "error", err)
	}
	if err := h.storage.DeletePrefix(ctx, submissionOutputPrefix(sid)); err != nil {
		slog.WarnContext(ctx, "delete submission outputs failed", "submission_id", sid, "error", err)
	}
}

func uniqueIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]bool, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// conflictAs rewrites a repository ErrConflict into a conflict with a user-facing message.
func conflictAs(err error, msg string) error {
	if errors.Is(err, domain.ErrConflict) {
		return fmt.Errorf("%w: %s", domain.ErrConflict, msg)
	}
	return err
}
