package commands

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
	"abet/main-service/internal/core/validation"
)

// Object layout in the bucket (shared with the portfolio worker).

func submissionUploadKey(id uuid.UUID) string    { return "uploads/" + id.String() + ".zip" }
func submissionPDFKey(id uuid.UUID) string       { return "portfolios/" + id.String() + ".pdf" }
func submissionOutputPrefix(id uuid.UUID) string { return "outputs/" + id.String() + "/" }

var programNameRe = regexp.MustCompile(`[^a-z0-9-]`)

// BuildProgramName is the worker's "program_name": the lower-cased curriculum code (anything but [a-z0-9-] becomes '-')
// followed by the academic year, e.g. "cs-2026".
func BuildProgramName(curriculumCode string, academicYear int) string {
	code := programNameRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(curriculumCode)), "-")
	if code == "" {
		code = "program"
	}
	return fmt.Sprintf("%s-%d", code, academicYear)
}

// PortfolioFilename is the download name of an offering's PDF: "<course_code>_<year>_S<semester>_portfolio.pdf".
func PortfolioFilename(o domain.OfferingInfo) string {
	return fmt.Sprintf("%s_%d_S%d_portfolio.pdf", safeFilePart(o.CourseCode), o.AcademicYear, o.Semester)
}

var unsafeFileRe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func safeFilePart(s string) string {
	s = strings.Trim(unsafeFileRe.ReplaceAllString(strings.TrimSpace(s), "-"), "-.")
	if s == "" {
		return "course"
	}
	return s
}

// sanitizeUploadName keeps the base name of the browser-supplied filename (no path, no control characters).
func sanitizeUploadName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
	if name == "." || name == "/" || name == "" {
		return "upload.zip"
	}
	if utf8.RuneCountInString(name) > 200 {
		name = string([]rune(name)[:200])
	}
	return name
}

// LoadOfferingAccess loads an offering and the actor's effective permissions on it.
func LoadOfferingAccess(ctx context.Context, repo port.SubmissionRepository, actor *domain.Actor, offeringID uuid.UUID) (*domain.OfferingInfo, domain.OfferingPerms, error) {
	info, err := repo.GetOffering(ctx, offeringID)
	if err != nil {
		return nil, domain.OfferingPerms{}, notFoundAs(err, "offering not found")
	}
	var grant *domain.AccessGrant
	if actor.Role != domain.RoleSuperAdmin {
		grants, err := repo.Grants(ctx, actor.UserID, []uuid.UUID{offeringID})
		if err != nil {
			return nil, domain.OfferingPerms{}, err
		}
		if g, ok := grants[offeringID]; ok {
			grant = &g
		}
	}
	return info, actor.OfferingPermissions(info.Scope(), grant), nil
}

// ---- upload ----

// UploadSubmissionHandler implements port.UploadSubmissionUseCase (submission-pipeline.md sections 1 and 2).
type UploadSubmissionHandler struct {
	repo      port.SubmissionRepository
	storage   port.Storage
	publisher port.JobPublisher
	tmpDir    string
	now       func() time.Time
}

var _ port.UploadSubmissionUseCase = (*UploadSubmissionHandler)(nil)

// NewUploadSubmissionHandler: tmpDir is where the request body is spooled before validation ("" = os.TempDir()).
func NewUploadSubmissionHandler(repo port.SubmissionRepository, storage port.Storage, publisher port.JobPublisher, tmpDir string) *UploadSubmissionHandler {
	return &UploadSubmissionHandler{repo: repo, storage: storage, publisher: publisher, tmpDir: tmpDir, now: func() time.Time { return time.Now().UTC() }}
}

func (h *UploadSubmissionHandler) Execute(ctx context.Context, actor *domain.Actor, in port.UploadSubmissionInput) (*domain.Submission, error) {
	info, perms, err := LoadOfferingAccess(ctx, h.repo, actor, in.OfferingID)
	if err != nil {
		return nil, err
	}
	if !perms.Upload {
		return nil, domain.ErrForbidden
	}

	// The body goes to a local temp file first: validation needs random access and an invalid zip must never reach the bucket.
	tmp, size, err := h.spool(in.Body)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	issues, err := validateZip(tmp, size, info.CourseCode)
	if err != nil {
		return nil, err
	}

	sid, jobID := uuid.New(), uuid.New()
	nErr, nWarn := domain.CountIssues(issues)
	now := h.now()
	sub := &domain.Submission{
		ID: sid, OfferingID: info.ID, UploadedBy: &domain.UserRef{ID: actor.UserID, Email: actor.Email},
		OriginalFilename: sanitizeUploadName(in.Filename), SizeBytes: size, Status: domain.SubmissionQueued, JobID: jobID,
		ErrorCount: nErr, WarningCount: nWarn, CreatedAt: now, UpdatedAt: now, Issues: issues,
	}
	if nErr > 0 {
		sub.Status = domain.SubmissionInvalid
		sub.CompletedAt = &now
		if err := h.repo.Create(ctx, sub); err != nil {
			return nil, fmt.Errorf("save submission: %w", err)
		}
		return h.repo.GetByID(ctx, sid)
	}

	uploadKey := submissionUploadKey(sid)
	if err := h.storage.PutFile(ctx, uploadKey, tmp.Name(), "application/zip"); err != nil {
		_ = h.storage.Delete(ctx, uploadKey)
		return nil, fmt.Errorf("store upload: %w", err)
	}
	if err := h.repo.Create(ctx, sub); err != nil {
		_ = h.storage.Delete(ctx, uploadKey)
		return nil, fmt.Errorf("save submission: %w", err)
	}

	if err := h.publish(ctx, info, sub); err != nil {
		slog.ErrorContext(ctx, "queueing the portfolio build failed", "submission_id", sid, "error", err)
		_ = h.storage.Delete(ctx, uploadKey)
		issue := validation.NewIssue(domain.SeverityError, validation.CodeQueueUnavailable,
			"Portfolio build service unavailable. Please try again later.", "")
		if _, ferr := h.repo.CompleteFailure(ctx, sid, []domain.Issue{issue}, h.now()); ferr != nil {
			return nil, fmt.Errorf("record queue failure: %w", ferr)
		}
		return h.repo.GetByID(ctx, sid)
	}
	return h.repo.GetByID(ctx, sid)
}

// spool copies the body into a new temp file and returns it with its size. On error nothing is left behind.
func (h *UploadSubmissionHandler) spool(body io.Reader) (*os.File, int64, error) {
	tmp, err := os.CreateTemp(h.tmpDir, "upload-*.zip")
	if err != nil {
		return nil, 0, fmt.Errorf("create temp file: %w", err)
	}
	size, err := io.Copy(tmp, body)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return nil, 0, err
	}
	return tmp, size, nil
}

// publish sends the v2 job (object keys, no filesystem paths).
func (h *UploadSubmissionHandler) publish(ctx context.Context, info *domain.OfferingInfo, sub *domain.Submission) error {
	job := domain.BuildJob{
		JobID:                 sub.JobID,
		ProgramName:           BuildProgramName(info.Curriculum.Code, info.AcademicYear),
		ZipKey:                submissionUploadKey(sub.ID),
		OutputPrefix:          submissionOutputPrefix(sub.ID),
		DeleteZipOnCompletion: true,
	}
	coverKey := CoverStorageKey(info.Curriculum.ID)
	if _, err := h.storage.Stat(ctx, coverKey); err == nil {
		job.CoverKey = coverKey
	} else if !errors.Is(err, domain.ErrNotFound) {
		slog.WarnContext(ctx, "could not check the curriculum cover, building with the default cover", "curriculum_id", info.Curriculum.ID, "error", err)
	}
	return h.publisher.Publish(ctx, job)
}

// validateZip runs the immediate validator on a spooled file. A file that is not a zip is a 400.
func validateZip(f io.ReaderAt, size int64, courseCode string) ([]domain.Issue, error) {
	if _, err := zip.NewReader(f, size); err != nil {
		return nil, fmt.Errorf("%w: the file is not a valid zip archive", domain.ErrInvalidInput)
	}
	return validation.Validate(f, size, courseCode), nil
}

// ---- worker status events ----

// ProcessBuildStatusHandler implements port.ProcessBuildStatusUseCase (submission-pipeline.md section 3).
type ProcessBuildStatusHandler struct {
	repo    port.SubmissionRepository
	storage port.Storage
	now     func() time.Time
}

var _ port.ProcessBuildStatusUseCase = (*ProcessBuildStatusHandler)(nil)

func NewProcessBuildStatusHandler(repo port.SubmissionRepository, storage port.Storage) *ProcessBuildStatusHandler {
	return &ProcessBuildStatusHandler{repo: repo, storage: storage, now: func() time.Time { return time.Now().UTC() }}
}

// Execute returns nil when the event is dealt with (or can never be dealt with) and an error when it should be retried.
func (h *ProcessBuildStatusHandler) Execute(ctx context.Context, ev domain.BuildStatusEvent) error {
	jobID, err := uuid.Parse(ev.JobID)
	if err != nil {
		slog.WarnContext(ctx, "status event with unusable job_id ignored", "job_id", ev.JobID, "status", ev.Status)
		return nil
	}
	sub, err := h.repo.GetByJobID(ctx, jobID)
	if errors.Is(err, domain.ErrNotFound) {
		slog.WarnContext(ctx, "status event for unknown job ignored", "job_id", ev.JobID, "status", ev.Status)
		return nil
	}
	if err != nil {
		return fmt.Errorf("load submission: %w", err)
	}
	if sub.Status != domain.SubmissionQueued && sub.Status != domain.SubmissionProcessing {
		// A success that arrives after the reconciler timed the submission out still wins.
		if strings.EqualFold(ev.Status, "success") && sub.Status == domain.SubmissionFailed {
			late, err := h.timedOut(ctx, sub.ID)
			if err != nil {
				return err
			}
			if late {
				slog.InfoContext(ctx, "late success for a timed-out submission accepted", "submission_id", sub.ID)
				return h.success(ctx, sub, ev)
			}
		}
		slog.InfoContext(ctx, "status event for finished submission ignored", "submission_id", sub.ID, "status", sub.Status)
		return nil
	}

	switch strings.ToLower(ev.Status) {
	case "success":
		return h.success(ctx, sub, ev)
	case "failed":
		return h.failure(ctx, sub, validation.ParseWorkerFailure(ev.Error))
	default:
		slog.InfoContext(ctx, "status event ignored", "job_id", ev.JobID, "status", ev.Status)
		return nil
	}
}

// timedOut reports whether a failed submission failed only because the reconciler timed it out.
func (h *ProcessBuildStatusHandler) timedOut(ctx context.Context, id uuid.UUID) (bool, error) {
	full, err := h.repo.GetByID(ctx, id)
	if err != nil {
		return false, fmt.Errorf("load submission issues: %w", err)
	}
	for _, is := range full.Issues {
		if is.Code == validation.CodeBuildTimeout {
			return true, nil
		}
	}
	return false, nil
}

func (h *ProcessBuildStatusHandler) success(ctx context.Context, sub *domain.Submission, ev domain.BuildStatusEvent) error {
	srcKey := pickExportedPDF(ev.ExportedPDFKeys, submissionOutputPrefix(sub.ID))
	if srcKey == "" {
		slog.WarnContext(ctx, "worker reported success without a PDF", "submission_id", sub.ID, "exported", ev.ExportedPDFKeys)
		return h.noOutput(ctx, sub)
	}

	// Copy first, commit second: a row never says "generated" without its file.
	dstKey := submissionPDFKey(sub.ID)
	if err := h.storage.Copy(ctx, srcKey, dstKey); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			slog.WarnContext(ctx, "worker reported a PDF that does not exist", "submission_id", sub.ID, "key", srcKey)
			return h.noOutput(ctx, sub)
		}
		return fmt.Errorf("copy built pdf: %w", err)
	}
	st, err := h.storage.Stat(ctx, dstKey)
	if err != nil {
		return fmt.Errorf("stat stored pdf: %w", err)
	}

	out, err := h.repo.CompleteSuccess(ctx, sub.ID, st.Size, h.now())
	if err != nil {
		return fmt.Errorf("complete submission: %w", err)
	}
	if !out.Applied {
		return nil
	}
	h.cleanup(ctx, sub.ID)
	if out.Superseded {
		h.deleteQuietly(ctx, dstKey)
	}
	for _, old := range out.OlderIDs {
		h.deleteQuietly(ctx, submissionPDFKey(old))
	}
	return nil
}

func (h *ProcessBuildStatusHandler) noOutput(ctx context.Context, sub *domain.Submission) error {
	return h.failure(ctx, sub, []domain.Issue{
		validation.NewIssue(domain.SeverityError, validation.CodeBuildNoOutput, "PDF generation produced no output", ""),
	})
}

func (h *ProcessBuildStatusHandler) failure(ctx context.Context, sub *domain.Submission, issues []domain.Issue) error {
	applied, err := h.repo.CompleteFailure(ctx, sub.ID, issues, h.now())
	if err != nil {
		return fmt.Errorf("complete submission: %w", err)
	}
	if applied {
		h.cleanup(ctx, sub.ID)
	}
	return nil
}

// cleanup removes the leftovers of a finished job: the uploaded zip (if the worker kept it) and the outputs prefix.
func (h *ProcessBuildStatusHandler) cleanup(ctx context.Context, id uuid.UUID) {
	h.deleteQuietly(ctx, submissionUploadKey(id))
	if err := h.storage.DeletePrefix(ctx, submissionOutputPrefix(id)); err != nil {
		slog.WarnContext(ctx, "could not delete worker outputs", "submission_id", id, "error", err)
	}
}

func (h *ProcessBuildStatusHandler) deleteQuietly(ctx context.Context, key string) {
	if err := h.storage.Delete(ctx, key); err != nil {
		slog.WarnContext(ctx, "could not delete object", "key", key, "error", err)
	}
}

// pickExportedPDF chooses the portfolio PDF among the exported object keys: the first "*-main.pdf", else the first ".pdf".
// Only keys below the job's own output prefix are accepted (the worker's message is not trusted beyond that).
func pickExportedPDF(keys []string, prefix string) string {
	first := ""
	for _, k := range keys {
		lower := strings.ToLower(k)
		if !strings.HasPrefix(k, prefix) || len(k) == len(prefix) || strings.Contains(k, "..") || !strings.HasSuffix(lower, ".pdf") {
			continue
		}
		if strings.HasSuffix(lower, "-main.pdf") {
			return k
		}
		if first == "" {
			first = k
		}
	}
	return first
}
