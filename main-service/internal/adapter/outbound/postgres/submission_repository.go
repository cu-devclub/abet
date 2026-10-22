package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"abet/main-service/internal/adapter/outbound/postgres/sqlc"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
	"abet/main-service/internal/core/validation"
)

type SubmissionRepository struct {
	store *Store
}

var _ port.SubmissionRepository = (*SubmissionRepository)(nil)

func NewSubmissionRepository(store *Store) *SubmissionRepository {
	return &SubmissionRepository{store: store}
}

func (r *SubmissionRepository) GetOffering(ctx context.Context, id uuid.UUID) (*domain.OfferingInfo, error) {
	row, err := r.store.q.GetOfferingForSubmission(ctx, id)
	if err != nil {
		return nil, mapErr("get offering", err)
	}
	info := offeringInfo(sqlc.ListOfferingsForSubmissionsRow(row))
	return &info, nil
}

func (r *SubmissionRepository) ListOfferings(ctx context.Context, ids []uuid.UUID) ([]domain.OfferingInfo, error) {
	rows, err := r.store.q.ListOfferingsForSubmissions(ctx, ids)
	if err != nil {
		return nil, mapErr("list offerings", err)
	}
	out := make([]domain.OfferingInfo, 0, len(rows))
	for _, row := range rows {
		out = append(out, offeringInfo(row))
	}
	return out, nil
}

func offeringInfo(row sqlc.ListOfferingsForSubmissionsRow) domain.OfferingInfo {
	return domain.OfferingInfo{
		ID: row.ID, CourseCode: row.CourseCode, CourseName: row.CourseName,
		AcademicYear: int(row.AcademicYear), Semester: int(row.Semester),
		Curriculum:   domain.Ref{ID: row.CurriculumID, Code: row.CurriculumCode, Name: row.CurriculumName},
		DepartmentID: row.DepartmentID,
		Department:   domain.Ref{ID: row.DepartmentID, Code: row.DepartmentCode, Name: row.DepartmentName},
	}
}

func (r *SubmissionRepository) Grants(ctx context.Context, userID uuid.UUID, offeringIDs []uuid.UUID) (map[uuid.UUID]domain.AccessGrant, error) {
	rows, err := r.store.q.ListUserGrants(ctx, sqlc.ListUserGrantsParams{
		UserID: userID, All: offeringIDs == nil, OfferingIds: offeringIDs,
	})
	if err != nil {
		return nil, mapErr("list grants", err)
	}
	out := make(map[uuid.UUID]domain.AccessGrant, len(rows))
	for _, row := range rows {
		out[row.OfferingID] = domain.AccessGrant{CanUpload: row.CanUpload, CanDownload: row.CanDownload}
	}
	return out, nil
}

func (r *SubmissionRepository) Create(ctx context.Context, s *domain.Submission) error {
	return r.store.WithTx(ctx, func(q *sqlc.Queries) error {
		var uploader uuid.NullUUID
		if s.UploadedBy != nil {
			uploader = uuid.NullUUID{UUID: s.UploadedBy.ID, Valid: true}
		}
		err := q.InsertSubmission(ctx, sqlc.InsertSubmissionParams{
			ID: s.ID, OfferingID: s.OfferingID, UploadedBy: uploader, OriginalFilename: s.OriginalFilename,
			SizeBytes: s.SizeBytes, Status: string(s.Status), JobID: s.JobID,
			ErrorCount: int32(s.ErrorCount), WarningCount: int32(s.WarningCount),
			At: s.CreatedAt, CompletedAt: s.CompletedAt,
		})
		if err != nil {
			return mapErr("insert submission", err)
		}
		return insertIssues(ctx, q, s.ID, s.Issues, 0)
	})
}

// insertIssues writes all issues with one statement (unnest), numbering them from firstOrder.
func insertIssues(ctx context.Context, q *sqlc.Queries, submissionID uuid.UUID, issues []domain.Issue, firstOrder int32) error {
	if len(issues) == 0 {
		return nil
	}
	p := sqlc.InsertSubmissionIssuesParams{SubmissionID: submissionID}
	for i, is := range issues {
		p.Severities = append(p.Severities, string(is.Severity))
		p.Stages = append(p.Stages, string(is.Stage))
		p.Codes = append(p.Codes, is.Code)
		p.Titles = append(p.Titles, is.Title)
		p.Descriptions = append(p.Descriptions, is.Description)
		p.Locations = append(p.Locations, is.Location)
		p.SuggestedFixes = append(p.SuggestedFixes, is.SuggestedFix)
		p.SortOrders = append(p.SortOrders, firstOrder+int32(i))
	}
	return mapErr("insert issues", q.InsertSubmissionIssues(ctx, p))
}

func (r *SubmissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Submission, error) {
	row, err := r.store.q.GetSubmission(ctx, id)
	if err != nil {
		return nil, mapErr("get submission", err)
	}
	s := submissionFromRow(row)
	issues, err := r.store.q.ListSubmissionIssues(ctx, id)
	if err != nil {
		return nil, mapErr("list issues", err)
	}
	s.Issues = make([]domain.Issue, 0, len(issues))
	for _, is := range issues {
		s.Issues = append(s.Issues, domain.Issue{
			Severity: domain.Severity(is.Severity), Stage: domain.Stage(is.Stage), Code: is.Code, Title: is.Title,
			Description: is.Description, Location: is.Location, SuggestedFix: is.SuggestedFix,
		})
	}
	return s, nil
}

func (r *SubmissionRepository) GetByJobID(ctx context.Context, jobID uuid.UUID) (*domain.Submission, error) {
	row, err := r.store.q.GetSubmissionByJobID(ctx, jobID)
	if err != nil {
		return nil, mapErr("get submission by job", err)
	}
	return submissionFromRow(sqlc.GetSubmissionRow(row)), nil
}

func (r *SubmissionRepository) ListByOffering(ctx context.Context, offeringID uuid.UUID) ([]domain.Submission, error) {
	rows, err := r.store.q.ListSubmissionsByOffering(ctx, offeringID)
	if err != nil {
		return nil, mapErr("list submissions", err)
	}
	out := make([]domain.Submission, 0, len(rows))
	for _, row := range rows {
		out = append(out, *submissionFromRow(sqlc.GetSubmissionRow(row)))
	}
	return out, nil
}

func submissionFromRow(row sqlc.GetSubmissionRow) *domain.Submission {
	s := &domain.Submission{
		ID: row.ID, OfferingID: row.OfferingID, OriginalFilename: row.OriginalFilename, SizeBytes: row.SizeBytes,
		Status: domain.SubmissionStatus(row.Status), JobID: row.JobID, PDFSizeBytes: row.PdfSizeBytes,
		ErrorCount: int(row.ErrorCount), WarningCount: int(row.WarningCount),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, CompletedAt: row.CompletedAt,
		Issues: []domain.Issue{},
	}
	if row.UploadedBy.Valid {
		s.UploadedBy = &domain.UserRef{ID: row.UploadedBy.UUID, Name: deref(row.UploaderName), Email: deref(row.UploaderEmail)}
	}
	return s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (r *SubmissionRepository) LatestGenerated(ctx context.Context, offeringID uuid.UUID) (uuid.UUID, error) {
	row, err := r.store.q.GetLatestGeneratedSubmission(ctx, offeringID)
	if err != nil {
		return uuid.Nil, mapErr("latest generated", err)
	}
	return row.ID, nil
}

func (r *SubmissionRepository) LatestGeneratedFor(ctx context.Context, offeringIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	rows, err := r.store.q.ListLatestGeneratedSubmissions(ctx, offeringIDs)
	if err != nil {
		return nil, mapErr("latest generated for offerings", err)
	}
	out := make(map[uuid.UUID]uuid.UUID, len(rows))
	for _, row := range rows {
		out[row.OfferingID] = row.ID
	}
	return out, nil
}

func (r *SubmissionRepository) CompleteSuccess(ctx context.Context, id uuid.UUID, pdfSize int64, at time.Time) (port.SuccessOutcome, error) {
	var out port.SuccessOutcome
	err := r.store.WithTx(ctx, func(q *sqlc.Queries) error {
		n, err := q.MarkSubmissionGenerated(ctx, sqlc.MarkSubmissionGeneratedParams{PdfSizeBytes: &pdfSize, At: at, ID: id})
		if err != nil {
			return mapErr("mark generated", err)
		}
		if n == 0 {
			// A late result for a submission the reconciler timed out: it wins, and the timeout issue goes away.
			n, err = q.MarkTimedOutSubmissionGenerated(ctx, sqlc.MarkTimedOutSubmissionGeneratedParams{PdfSizeBytes: &pdfSize, At: at, ID: id})
			if err != nil {
				return mapErr("mark late generated", err)
			}
			if n == 0 {
				return nil
			}
			if err := q.DeleteSubmissionIssuesByCode(ctx, sqlc.DeleteSubmissionIssuesByCodeParams{SubmissionID: id, Code: validation.CodeBuildTimeout}); err != nil {
				return mapErr("remove timeout issue", err)
			}
		}
		out.Applied = true
		newer, err := q.HasNewerGeneratedSubmission(ctx, id)
		if err != nil {
			return mapErr("check newer", err)
		}
		if newer {
			out.Superseded = true
			return mapErr("clear pdf", q.ClearSubmissionPDF(ctx, sqlc.ClearSubmissionPDFParams{At: at, ID: id}))
		}
		older, err := q.SupersedeOlderGenerated(ctx, sqlc.SupersedeOlderGeneratedParams{At: at, ID: id})
		if err != nil {
			return mapErr("supersede older", err)
		}
		out.OlderIDs = older
		return nil
	})
	if err != nil {
		return port.SuccessOutcome{}, err
	}
	return out, nil
}

func (r *SubmissionRepository) CompleteFailure(ctx context.Context, id uuid.UUID, issues []domain.Issue, at time.Time) (bool, error) {
	applied := false
	err := r.store.WithTx(ctx, func(q *sqlc.Queries) error {
		errs, _ := domain.CountIssues(issues)
		n, err := q.MarkSubmissionFailed(ctx, sqlc.MarkSubmissionFailedParams{ErrorCount: int32(errs), At: at, ID: id})
		if err != nil {
			return mapErr("mark failed", err)
		}
		if n == 0 {
			return nil
		}
		applied = true
		next, err := q.NextSubmissionIssueOrder(ctx, id)
		if err != nil {
			return mapErr("next issue order", err)
		}
		return insertIssues(ctx, q, id, issues, next)
	})
	if err != nil {
		return false, fmt.Errorf("complete failure: %w", err)
	}
	return applied, nil
}

func (r *SubmissionRepository) TimeOutStale(ctx context.Context, cutoff, at time.Time, issue domain.Issue) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.store.WithTx(ctx, func(q *sqlc.Queries) error {
		var err error
		ids, err = q.TimeOutStaleSubmissions(ctx, sqlc.TimeOutStaleSubmissionsParams{At: at, Cutoff: cutoff})
		if err != nil {
			return mapErr("time out stale submissions", err)
		}
		for _, id := range ids {
			next, err := q.NextSubmissionIssueOrder(ctx, id)
			if err != nil {
				return mapErr("next issue order", err)
			}
			if err := insertIssues(ctx, q, id, []domain.Issue{issue}, next); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("time out stale: %w", err)
	}
	return ids, nil
}

func (r *SubmissionRepository) Statuses(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.SubmissionStatus, error) {
	out := make(map[uuid.UUID]domain.SubmissionStatus, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.store.q.ListSubmissionStatuses(ctx, ids)
	if err != nil {
		return nil, mapErr("list submission statuses", err)
	}
	for _, row := range rows {
		out[row.ID] = domain.SubmissionStatus(row.Status)
	}
	return out, nil
}

func (r *SubmissionRepository) ListPortfolioRows(ctx context.Context, actor *domain.Actor, f port.PortfolioFilter) ([]port.PortfolioCandidate, error) {
	p := sqlc.ListPortfolioRowsParams{
		All:           actor.Role == domain.RoleSuperAdmin,
		CurriculumIds: actor.ScopeCurriculumIDs,
		UserID:        actor.UserID,
		DepartmentID:  nullUUID(f.DepartmentID),
		CurriculumID:  nullUUID(f.CurriculumID),
	}
	if p.CurriculumIds == nil || actor.Role != domain.RoleAdmin {
		p.CurriculumIds = []uuid.UUID{}
	}
	if actor.Role == domain.RoleStaff {
		p.StaffDepartmentID = nullUUID(actor.DepartmentID)
	}
	if f.AcademicYear != nil {
		v := int32(*f.AcademicYear)
		p.AcademicYear = &v
	}
	if f.Semester != nil {
		v := int16(*f.Semester)
		p.Semester = &v
	}
	if f.Search != "" {
		p.Search = &f.Search
	}
	rows, err := r.store.q.ListPortfolioRows(ctx, p)
	if err != nil {
		return nil, mapErr("list portfolio rows", err)
	}
	out := make([]port.PortfolioCandidate, 0, len(rows))
	for _, row := range rows {
		c := port.PortfolioCandidate{
			Offering: domain.OfferingInfo{
				ID: row.OfferingID, CourseCode: row.CourseCode, CourseName: row.CourseName,
				AcademicYear: int(row.AcademicYear), Semester: int(row.Semester),
				Curriculum:   domain.Ref{ID: row.CurriculumID, Code: row.CurriculumCode, Name: row.CurriculumName},
				DepartmentID: row.DepartmentID,
				Department:   domain.Ref{ID: row.DepartmentID, Code: row.DepartmentCode, Name: row.DepartmentName},
			},
			PDFAvailable: row.PdfAvailable,
		}
		if row.SubID.Valid {
			s := &domain.Submission{
				ID: row.SubID.UUID, OfferingID: row.OfferingID, Issues: []domain.Issue{},
				OriginalFilename: deref(row.SubOriginalFilename), Status: domain.SubmissionStatus(deref(row.SubStatus)),
				CompletedAt: row.SubCompletedAt,
			}
			if row.SubSizeBytes != nil {
				s.SizeBytes = *row.SubSizeBytes
			}
			if row.SubErrorCount != nil {
				s.ErrorCount = int(*row.SubErrorCount)
			}
			if row.SubWarningCount != nil {
				s.WarningCount = int(*row.SubWarningCount)
			}
			if row.SubCreatedAt != nil {
				s.CreatedAt = *row.SubCreatedAt
			}
			if row.SubUploadedBy.Valid {
				s.UploadedBy = &domain.UserRef{ID: row.SubUploadedBy.UUID, Name: deref(row.SubUploaderName), Email: deref(row.SubUploaderEmail)}
			}
			c.Latest = s
		}
		out = append(out, c)
	}
	return out, nil
}
