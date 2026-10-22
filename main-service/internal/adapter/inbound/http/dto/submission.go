package dto

import (
	"fmt"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
)

type IssueResponse struct {
	Severity     domain.Severity `json:"severity"`
	Stage        domain.Stage    `json:"stage"`
	Code         string          `json:"code"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	Location     string          `json:"location"`
	SuggestedFix string          `json:"suggested_fix"`
}

type UploaderResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

type SubmissionResponse struct {
	ID               uuid.UUID               `json:"id"`
	OfferingID       uuid.UUID               `json:"offering_id"`
	Status           domain.SubmissionStatus `json:"status"`
	OriginalFilename string                  `json:"original_filename"`
	SizeBytes        int64                   `json:"size_bytes"`
	UploadedBy       *UploaderResponse       `json:"uploaded_by"`
	ErrorCount       int                     `json:"error_count"`
	WarningCount     int                     `json:"warning_count"`
	CreatedAt        string                  `json:"created_at"`
	CompletedAt      *string                 `json:"completed_at"`
	Issues           []IssueResponse         `json:"issues"`
}

// SubmissionFromDomain maps a submission; withIssues=false renders "issues": [] (history and portfolio rows).
func SubmissionFromDomain(s *domain.Submission, withIssues bool) SubmissionResponse {
	out := SubmissionResponse{
		ID: s.ID, OfferingID: s.OfferingID, Status: s.Status, OriginalFilename: s.OriginalFilename,
		SizeBytes: s.SizeBytes, ErrorCount: s.ErrorCount, WarningCount: s.WarningCount,
		CreatedAt: FormatTime(s.CreatedAt), CompletedAt: formatTimePtr(s.CompletedAt),
		Issues: []IssueResponse{},
	}
	if s.UploadedBy != nil {
		out.UploadedBy = &UploaderResponse{ID: s.UploadedBy.ID, Name: s.UploadedBy.Name, Email: s.UploadedBy.Email}
	}
	if withIssues {
		for _, i := range s.Issues {
			out.Issues = append(out.Issues, IssueResponse{
				Severity: i.Severity, Stage: i.Stage, Code: i.Code, Title: i.Title, Description: i.Description,
				Location: i.Location, SuggestedFix: i.SuggestedFix,
			})
		}
	}
	return out
}

func SubmissionsFromDomain(in []domain.Submission) []SubmissionResponse {
	out := make([]SubmissionResponse, 0, len(in))
	for i := range in {
		out = append(out, SubmissionFromDomain(&in[i], false))
	}
	return out
}

type PortfolioRowResponse struct {
	OfferingID       uuid.UUID           `json:"offering_id"`
	CourseCode       string              `json:"course_code"`
	CourseName       string              `json:"course_name"`
	AcademicYear     int                 `json:"academic_year"`
	Semester         int                 `json:"semester"`
	Curriculum       RefResponse         `json:"curriculum"`
	Department       RefResponse         `json:"department"`
	CanUpload        bool                `json:"can_upload"`
	CanDownload      bool                `json:"can_download"`
	LatestSubmission *SubmissionResponse `json:"latest_submission"`
	PDFAvailable     bool                `json:"pdf_available"`
}

func PortfolioRowsFromDomain(rows []domain.PortfolioRow) []PortfolioRowResponse {
	out := make([]PortfolioRowResponse, 0, len(rows))
	for _, r := range rows {
		item := PortfolioRowResponse{
			OfferingID: r.Offering.ID, CourseCode: r.Offering.CourseCode, CourseName: r.Offering.CourseName,
			AcademicYear: r.Offering.AcademicYear, Semester: r.Offering.Semester,
			Curriculum: RefFromDomain(r.Offering.Curriculum), Department: RefFromDomain(r.Offering.Department),
			CanUpload: r.CanUpload, CanDownload: r.CanDownload, PDFAvailable: r.PDFAvailable,
		}
		if r.Latest != nil {
			s := SubmissionFromDomain(r.Latest, false)
			item.LatestSubmission = &s
		}
		out = append(out, item)
	}
	return out
}

// DownloadPortfoliosRequest is the body of POST /portfolios/download.
type DownloadPortfoliosRequest struct {
	OfferingIDs []uuid.UUID `json:"offering_ids"`
}

func (r DownloadPortfoliosRequest) Validate() error {
	if len(r.OfferingIDs) == 0 {
		return fmt.Errorf("%w: offering_ids must not be empty", domain.ErrInvalidInput)
	}
	if len(r.OfferingIDs) > 1000 {
		return fmt.Errorf("%w: at most 1000 offerings per download", domain.ErrInvalidInput)
	}
	return nil
}
