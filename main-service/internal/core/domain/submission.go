package domain

import (
	"time"

	"github.com/google/uuid"
)

// SubmissionStatus is the lifecycle state of an uploaded portfolio.
type SubmissionStatus string

const (
	SubmissionInvalid    SubmissionStatus = "invalid"
	SubmissionQueued     SubmissionStatus = "queued"
	SubmissionProcessing SubmissionStatus = "processing"
	SubmissionGenerated  SubmissionStatus = "generated"
	SubmissionFailed     SubmissionStatus = "failed"
)

// Severity of a validation issue.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Stage of the pipeline that produced an issue.
type Stage string

const (
	StageMissing   Stage = "missing"
	StageStructure Stage = "structure"
	StageNaming    Stage = "naming"
	StageCSV       Stage = "csv"
	StageBuild     Stage = "build"
)

// Issue is one validation finding (SRS section 7), produced by the immediate validator or parsed from the worker.
type Issue struct {
	Severity     Severity
	Stage        Stage
	Code         string
	Title        string
	Description  string
	Location     string
	SuggestedFix string
}

// UserRef is a compact id/name/email reference to the uploader.
type UserRef struct {
	ID    uuid.UUID
	Name  string
	Email string
}

// Submission is one upload attempt of an offering's portfolio.
type Submission struct {
	ID               uuid.UUID
	OfferingID       uuid.UUID
	UploadedBy       *UserRef
	OriginalFilename string
	SizeBytes        int64
	Status           SubmissionStatus
	JobID            uuid.UUID
	PDFSizeBytes     *int64
	ErrorCount       int
	WarningCount     int
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CompletedAt      *time.Time
	Issues           []Issue
}

// CountIssues returns the number of errors and warnings in issues.
func CountIssues(issues []Issue) (errs, warns int) {
	for _, is := range issues {
		if is.Severity == SeverityWarning {
			warns++
		} else {
			errs++
		}
	}
	return errs, warns
}

// OfferingInfo is everything the submission pipeline needs to know about an offering.
type OfferingInfo struct {
	ID           uuid.UUID
	CourseCode   string
	CourseName   string
	AcademicYear int
	Semester     int
	Curriculum   Ref
	DepartmentID uuid.UUID
	Department   Ref
}

// Scope returns the authorization scope of the offering.
func (o OfferingInfo) Scope() OfferingScope {
	return OfferingScope{CurriculumID: o.Curriculum.ID, DepartmentID: o.DepartmentID}
}

// BuildJob is the message published to the portfolio worker (JSON field names are the worker's contract).
type BuildJob struct {
	JobID                 uuid.UUID
	ProgramName           string
	ZipKey                string // object key of the uploaded zip
	CoverKey              string // object key of the curriculum cover; empty = default cover
	OutputPrefix          string // object key prefix the worker writes its results to ("outputs/<sid>/")
	DeleteZipOnCompletion bool
}

// BuildStatusEvent is a result message of the worker (success or failed).
type BuildStatusEvent struct {
	Status string
	JobID  string
	Error  string
	// ExportedPDFKeys are the object keys of the PDFs the worker uploaded below the job's output prefix.
	ExportedPDFKeys []string
}

// PortfolioRow is one line of the Course Portfolios page.
type PortfolioRow struct {
	Offering     OfferingInfo
	CanUpload    bool
	CanDownload  bool
	Latest       *Submission // without issues
	PDFAvailable bool
}
