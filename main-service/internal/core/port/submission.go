package port

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
)

// ---- outbound ----

// PortfolioFilter narrows the Course Portfolios list; nil/empty fields are ignored.
type PortfolioFilter struct {
	AcademicYear *int
	DepartmentID *uuid.UUID
	CurriculumID *uuid.UUID
	Semester     *int
	Search       string
}

// SuccessOutcome tells the status consumer what CompleteSuccess did.
type SuccessOutcome struct {
	// Applied is false when the submission was not waiting for a result any more (duplicate delivery).
	Applied bool
	// Superseded is true when a newer submission of the offering already has a generated PDF; this one keeps no PDF.
	Superseded bool
	// OlderIDs are older generated submissions of the same offering whose PDFs must now be deleted.
	OlderIDs []uuid.UUID
}

// SubmissionRepository persists submissions, their issues and the read models of the portfolio pages.
// Unknown ids answer domain.ErrNotFound.
type SubmissionRepository interface {
	// GetOffering loads the offering with its curriculum/department (ErrNotFound if missing).
	GetOffering(ctx context.Context, id uuid.UUID) (*domain.OfferingInfo, error)
	// ListOfferings loads several offerings at once (missing ids are simply absent).
	ListOfferings(ctx context.Context, ids []uuid.UUID) ([]domain.OfferingInfo, error)
	// Grants returns the explicit course_access rows of a user; offeringIDs == nil means all of them.
	Grants(ctx context.Context, userID uuid.UUID, offeringIDs []uuid.UUID) (map[uuid.UUID]domain.AccessGrant, error)

	// Create inserts the submission and its issues in ONE transaction.
	Create(ctx context.Context, s *domain.Submission) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Submission, error)
	GetByJobID(ctx context.Context, jobID uuid.UUID) (*domain.Submission, error)
	ListByOffering(ctx context.Context, offeringID uuid.UUID) ([]domain.Submission, error)

	// LatestGenerated returns the id of the newest generated submission that still has a PDF (ErrNotFound if none).
	LatestGenerated(ctx context.Context, offeringID uuid.UUID) (uuid.UUID, error)
	// LatestGeneratedFor does the same for many offerings in one query (offerings without a PDF are absent).
	LatestGeneratedFor(ctx context.Context, offeringIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error)

	// CompleteSuccess marks a queued submission generated (or a failed one whose only failure was BUILD_TIMEOUT: the late
	// result wins and the timeout issue is removed), and clears the PDF of older generated ones (one transaction).
	CompleteSuccess(ctx context.Context, id uuid.UUID, pdfSize int64, at time.Time) (SuccessOutcome, error)
	// CompleteFailure marks a queued submission failed and appends the issues (one transaction).
	// It reports false when the submission was not waiting for a result any more.
	CompleteFailure(ctx context.Context, id uuid.UUID, issues []domain.Issue, at time.Time) (bool, error)

	// TimeOutStale atomically fails every submission still queued that was created before cutoff
	// (UPDATE ... WHERE status='queued'), attaches the issue to each and returns their ids.
	TimeOutStale(ctx context.Context, cutoff, at time.Time, issue domain.Issue) ([]uuid.UUID, error)
	// Statuses returns the status of each existing submission among ids (unknown ids are absent).
	Statuses(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.SubmissionStatus, error)

	// ListPortfolioRows returns every offering the actor may possibly see (with the latest submission and
	// pdf availability) in ONE query; exact per-offering rights are decided by domain.Actor.OfferingPermissions.
	ListPortfolioRows(ctx context.Context, actor *domain.Actor, f PortfolioFilter) ([]PortfolioCandidate, error)
}

// PortfolioCandidate is an offering row of the portfolio page before permissions are applied.
type PortfolioCandidate struct {
	Offering     domain.OfferingInfo
	Latest       *domain.Submission
	PDFAvailable bool
}

// JobPublisher hands a build job to the portfolio worker (persistent message, publisher confirm).
// Failures are wrapped in domain.ErrServiceUnavailable.
type JobPublisher interface {
	Publish(ctx context.Context, job domain.BuildJob) error
}

// ---- inbound ----

type UploadSubmissionInput struct {
	OfferingID uuid.UUID
	Filename   string
	// Body is streamed to storage; it must already be size-limited (a read error wrapping domain.ErrTooLarge is passed through).
	Body io.Reader
}

type UploadSubmissionUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, in UploadSubmissionInput) (*domain.Submission, error)
}

type GetSubmissionUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, id uuid.UUID) (*domain.Submission, error)
}

type ListSubmissionsUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, offeringID uuid.UUID) ([]domain.Submission, error)
}

// PDFFile is an opened portfolio PDF ready to be served.
type PDFFile struct {
	File     io.ReadCloser
	Info     ObjectInfo
	Filename string // suggested download name
}

type GetPortfolioPDFUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, offeringID uuid.UUID) (*PDFFile, error)
}

type ListPortfoliosUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, f PortfolioFilter) ([]domain.PortfolioRow, error)
}

// DownloadEntry is one PDF of a bulk download.
type DownloadEntry struct {
	Name       string // name inside the zip
	StorageKey string
}

// DownloadPlan is what a bulk download will contain (decided before the first byte is written).
type DownloadPlan struct {
	ZipName string
	Entries []DownloadEntry
	// Missing are human-readable lines for MISSING.txt (offerings without PDF, forbidden or unknown).
	Missing []string
}

type DownloadPortfoliosUseCase interface {
	// Plan checks permissions and picks the PDFs: ErrInvalidInput for an empty list, ErrNotFound when nothing is available.
	Plan(ctx context.Context, actor *domain.Actor, offeringIDs []uuid.UUID) (*DownloadPlan, error)
	// Write streams the plan as a zip archive into w.
	Write(ctx context.Context, plan *DownloadPlan, w io.Writer) error
}

// ProcessBuildStatusUseCase handles one worker status event; a nil result means "ack", an error means "retry later".
type ProcessBuildStatusUseCase interface {
	Execute(ctx context.Context, ev domain.BuildStatusEvent) error
}

// TemplateService serves the downloadable templates.
type TemplateService interface {
	// WriteInstructorPackage streams the instructor package folder as a zip (top folder "course-id-course-name/").
	WriteInstructorPackage(w io.Writer) error
	// CourseMappingCSV returns the sample mapping CSV (header + 2 example rows).
	CourseMappingCSV() []byte
}
