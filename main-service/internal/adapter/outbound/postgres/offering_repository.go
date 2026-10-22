package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"abet/main-service/internal/adapter/outbound/postgres/sqlc"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

type OfferingRepository struct {
	store *Store
}

var _ port.OfferingRepository = (*OfferingRepository)(nil)

func NewOfferingRepository(store *Store) *OfferingRepository {
	return &OfferingRepository{store: store}
}

func (r *OfferingRepository) List(ctx context.Context, f port.OfferingFilter) ([]domain.Offering, error) {
	p := sqlc.ListOfferingsParams{
		AcademicYear:  int32(f.AcademicYear),
		Restrict:      f.Restrict,
		CurriculumIds: f.CurriculumScope,
	}
	if p.CurriculumIds == nil {
		p.CurriculumIds = []uuid.UUID{}
	}
	if f.DepartmentID != nil {
		p.DepartmentID = uuid.NullUUID{UUID: *f.DepartmentID, Valid: true}
	}
	if f.CurriculumID != nil {
		p.CurriculumID = uuid.NullUUID{UUID: *f.CurriculumID, Valid: true}
	}
	if f.Semester != nil {
		s := int16(*f.Semester)
		p.Semester = &s
	}
	if f.Search != "" {
		p.Search = &f.Search
	}
	rows, err := r.store.q.ListOfferings(ctx, p)
	if err != nil {
		return nil, mapErr("list offerings", err)
	}
	out := make([]domain.Offering, 0, len(rows))
	for _, row := range rows {
		out = append(out, offeringFromRow(row))
	}
	return out, nil
}

func (r *OfferingRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Offering, error) {
	list, err := r.ListByIDs(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("get offering: %w", domain.ErrNotFound)
	}
	return &list[0], nil
}

func (r *OfferingRepository) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Offering, error) {
	rows, err := r.store.q.ListOfferingsByIDs(ctx, ids)
	if err != nil {
		return nil, mapErr("list offerings by ids", err)
	}
	out := make([]domain.Offering, 0, len(rows))
	for _, row := range rows {
		out = append(out, offeringFromRow(sqlc.ListOfferingsRow(row)))
	}
	return out, nil
}

func (r *OfferingRepository) Years(ctx context.Context) ([]int, error) {
	rows, err := r.store.q.ListOfferingYears(ctx)
	if err != nil {
		return nil, mapErr("list offering years", err)
	}
	out := make([]int, 0, len(rows))
	for _, y := range rows {
		out = append(out, int(y))
	}
	return out, nil
}

func (r *OfferingRepository) Create(ctx context.Context, p port.CreateOfferingParams) (uuid.UUID, error) {
	id := uuid.New()
	now := time.Now().UTC()
	err := r.store.WithTx(ctx, func(q *sqlc.Queries) error {
		courseID, err := q.UpsertCourse(ctx, sqlc.UpsertCourseParams{
			ID: uuid.New(), Code: p.CourseCode, Name: p.CourseName, CurriculumID: p.CurriculumID, At: now, Rename: true,
		})
		if err != nil {
			return mapErr("upsert course", err)
		}
		return mapErr("create offering", q.CreateOffering(ctx, sqlc.CreateOfferingParams{
			ID: id, CourseID: courseID, AcademicYear: int32(p.AcademicYear), Semester: int16(p.Semester), At: now,
		}))
	})
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *OfferingRepository) Update(ctx context.Context, p port.UpdateOfferingParams) error {
	now := time.Now().UTC()
	return r.store.WithTx(ctx, func(q *sqlc.Queries) error {
		oldCourseID, err := q.LockOffering(ctx, p.ID)
		if err != nil {
			return mapErr("lock offering", err)
		}
		courseID, err := q.UpsertCourse(ctx, sqlc.UpsertCourseParams{
			ID: uuid.New(), Code: p.CourseCode, Name: p.CourseName, CurriculumID: p.CurriculumID, At: now, Rename: p.RenameCourse,
		})
		if err != nil {
			return mapErr("upsert course", err)
		}
		n, err := q.UpdateOffering(ctx, sqlc.UpdateOfferingParams{
			ID: p.ID, CourseID: courseID, AcademicYear: int32(p.AcademicYear), Semester: int16(p.Semester), At: now,
		})
		if err != nil {
			return mapErr("update offering", err)
		}
		if n == 0 {
			return fmt.Errorf("update offering: %w", domain.ErrNotFound)
		}
		if oldCourseID != courseID {
			if _, err := q.DeleteOrphanCourses(ctx, []uuid.UUID{oldCourseID}); err != nil {
				return mapErr("delete orphan course", err)
			}
		}
		return nil
	})
}

func (r *OfferingRepository) Delete(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	var submissionIDs []uuid.UUID
	err := r.store.WithTx(ctx, func(q *sqlc.Queries) error {
		var err error
		if submissionIDs, err = q.ListSubmissionIDsForOfferings(ctx, ids); err != nil {
			return mapErr("list submissions", err)
		}
		courseIDs, err := q.DeleteOfferings(ctx, ids)
		if err != nil {
			return mapErr("delete offerings", err)
		}
		if len(courseIDs) == 0 {
			return fmt.Errorf("delete offerings: %w", domain.ErrNotFound)
		}
		if _, err := q.DeleteOrphanCourses(ctx, courseIDs); err != nil {
			return mapErr("delete orphan courses", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return submissionIDs, nil
}

func offeringFromRow(row sqlc.ListOfferingsRow) domain.Offering {
	return domain.Offering{
		ID: row.ID, CourseID: row.CourseID, CourseCode: row.CourseCode, CourseName: row.CourseName,
		AcademicYear: int(row.AcademicYear), Semester: int(row.Semester),
		Curriculum:      domain.Ref{ID: row.CurriculumID, Code: row.CurriculumCode, Name: row.CurriculumName},
		Department:      domain.Ref{ID: row.DepartmentID, Code: row.DepartmentCode, Name: row.DepartmentName},
		InstructorCount: int(row.InstructorCount), StaffCount: int(row.StaffCount),
		HasSubmissions: row.HasSubmissions, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}
