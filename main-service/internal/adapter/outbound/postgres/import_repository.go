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

type ImportRepository struct {
	store *Store
}

var _ port.ImportRepository = (*ImportRepository)(nil)

func NewImportRepository(store *Store) *ImportRepository {
	return &ImportRepository{store: store}
}

func (r *ImportRepository) LoadState(ctx context.Context, curriculumID uuid.UUID, academicYear int, emails []string) (*domain.ImportState, error) {
	return loadImportState(ctx, r.store.q, curriculumID, academicYear, emails)
}

func (r *ImportRepository) Apply(ctx context.Context, p port.ApplyImportParams) (*domain.ImportPlan, error) {
	var plan domain.ImportPlan
	err := r.store.WithTx(ctx, func(q *sqlc.Queries) error {
		state, err := loadImportState(ctx, q, p.Curriculum.ID, p.AcademicYear, p.Emails)
		if err != nil {
			return err
		}
		plan = p.Build(state)
		if len(plan.Errors) > 0 {
			return fmt.Errorf("%w: the CSV has errors", domain.ErrUnprocessable)
		}
		if err := executeImportPlan(ctx, q, p, state, plan); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if len(plan.Errors) > 0 {
			return &plan, err
		}
		return nil, err
	}
	plan.Applied = true
	return &plan, nil
}

// loadImportState locks the offerings of the scope (FOR UPDATE) and reads their access rows and the CSV users.
func loadImportState(ctx context.Context, q *sqlc.Queries, curriculumID uuid.UUID, year int, emails []string) (*domain.ImportState, error) {
	offerings, err := q.ListScopeOfferingsForUpdate(ctx, sqlc.ListScopeOfferingsForUpdateParams{CurriculumID: curriculumID, AcademicYear: int32(year)})
	if err != nil {
		return nil, mapErr("list scope offerings", err)
	}
	accessRows, err := q.ListScopeAccess(ctx, sqlc.ListScopeAccessParams{CurriculumID: curriculumID, AcademicYear: int32(year)})
	if err != nil {
		return nil, mapErr("list scope access", err)
	}
	if emails == nil {
		emails = []string{}
	}
	users, err := q.ListUsersByEmails(ctx, emails)
	if err != nil {
		return nil, mapErr("list users by email", err)
	}

	state := &domain.ImportState{Users: make(map[string]domain.ImportUserState, len(users))}
	byOffering := make(map[uuid.UUID]int, len(offerings))
	for _, o := range offerings {
		byOffering[o.ID] = len(state.Offerings)
		state.Offerings = append(state.Offerings, domain.ImportOfferingState{
			OfferingID: o.ID, CourseID: o.CourseID, CourseCode: o.CourseCode, CourseName: o.CourseName,
			Semester: int(o.Semester), HasSubmissions: o.HasSubmissions,
		})
	}
	for _, a := range accessRows {
		i, ok := byOffering[a.OfferingID]
		if !ok {
			continue
		}
		state.Offerings[i].Access = append(state.Offerings[i].Access, domain.ImportAccessState{
			UserID: a.UserID, Email: a.Email, Role: domain.Role(a.Role),
			CanUpload: a.CanUpload, CanDownload: a.CanDownload, Source: a.Source,
		})
	}
	for _, u := range users {
		state.Users[u.Email] = domain.ImportUserState{UserID: u.ID, Email: u.Email, Role: domain.Role(u.Role)}
	}
	return state, nil
}

type importKey struct {
	code     string
	semester int
}

// executeImportPlan writes the plan; state is the locked state the plan was built from.
func executeImportPlan(ctx context.Context, q *sqlc.Queries, p port.ApplyImportParams, state *domain.ImportState, plan domain.ImportPlan) error {
	now := time.Now().UTC()

	userIDs := make(map[string]uuid.UUID, len(state.Users))
	for email, u := range state.Users {
		userIDs[email] = u.UserID
	}
	for _, u := range plan.Users {
		switch u.Action {
		case domain.UserCreate:
			id := uuid.New()
			err := q.CreateUser(ctx, sqlc.CreateUserParams{
				ID: id, Email: u.Email, Role: string(u.Role),
				DepartmentID: uuid.NullUUID{UUID: p.Curriculum.DepartmentID, Valid: true},
				CurriculumID: uuid.NullUUID{UUID: p.Curriculum.ID, Valid: true},
				CreatedAt:    now, UpdatedAt: now,
			})
			if err != nil {
				return mapErr("create user", err)
			}
			userIDs[u.Email] = id
		case domain.UserRaiseRole:
			if _, err := q.UpdateUserRole(ctx, sqlc.UpdateUserRoleParams{ID: userIDs[u.Email], Role: string(u.Role), UpdatedAt: now}); err != nil {
				return mapErr("raise user role", err)
			}
		}
	}

	current := make(map[importKey]domain.ImportOfferingState, len(state.Offerings))
	offeringIDs := make(map[importKey]uuid.UUID, len(state.Offerings))
	for _, o := range state.Offerings {
		k := importKey{o.CourseCode, o.Semester}
		current[k] = o
		offeringIDs[k] = o.OfferingID
		for _, a := range o.Access {
			userIDs[a.Email] = a.UserID
		}
	}

	var deleteIDs []uuid.UUID
	for _, c := range plan.Courses {
		k := importKey{c.CourseCode, c.Semester}
		switch c.Action {
		case domain.CourseCreate:
			courseID, err := q.UpsertCourse(ctx, sqlc.UpsertCourseParams{
				ID: uuid.New(), Code: c.CourseCode, Name: c.CourseName, CurriculumID: p.Curriculum.ID, At: now, Rename: true,
			})
			if err != nil {
				return mapErr("upsert course", err)
			}
			id := uuid.New()
			if err := q.CreateOffering(ctx, sqlc.CreateOfferingParams{
				ID: id, CourseID: courseID, AcademicYear: int32(p.AcademicYear), Semester: int16(c.Semester), At: now,
			}); err != nil {
				return mapErr("create offering", err)
			}
			offeringIDs[k] = id
		case domain.CourseUpdate:
			if err := q.UpdateCourseName(ctx, sqlc.UpdateCourseNameParams{ID: current[k].CourseID, Name: c.CourseName, UpdatedAt: now}); err != nil {
				return mapErr("update course", err)
			}
		case domain.CourseDelete:
			deleteIDs = append(deleteIDs, offeringIDs[k])
		}
	}

	for _, a := range plan.Access {
		k := importKey{a.CourseCode, a.Semester}
		offeringID, userID := offeringIDs[k], userIDs[a.Email]
		switch a.Action {
		case domain.AccessAdd, domain.AccessUpdate:
			err := q.UpsertAccess(ctx, sqlc.UpsertAccessParams{
				ID: uuid.New(), OfferingID: offeringID, UserID: userID,
				CanUpload: a.CanUpload, CanDownload: a.CanDownload, Source: domain.AccessSourceCSV, At: now,
			})
			if err != nil {
				return mapErr("upsert access", err)
			}
		case domain.AccessRemove:
			if _, err := q.DeleteAccess(ctx, sqlc.DeleteAccessParams{OfferingID: offeringID, UserID: userID}); err != nil {
				return mapErr("delete access", err)
			}
		}
	}

	if len(deleteIDs) > 0 {
		courseIDs, err := q.DeleteOfferings(ctx, deleteIDs)
		if err != nil {
			return mapErr("delete offerings", err)
		}
		if _, err := q.DeleteOrphanCourses(ctx, courseIDs); err != nil {
			return mapErr("delete orphan courses", err)
		}
	}
	return nil
}
