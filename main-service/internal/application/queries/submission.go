package queries

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"abet/main-service/internal/application/commands"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

func submissionPDFKey(id uuid.UUID) string { return "portfolios/" + id.String() + ".pdf" }

func notFound(err error, msg string) error {
	if errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("%w: %s", domain.ErrNotFound, msg)
	}
	return err
}

// ---- GET /submissions/{id} ----

type GetSubmissionHandler struct{ repo port.SubmissionRepository }

var _ port.GetSubmissionUseCase = (*GetSubmissionHandler)(nil)

func NewGetSubmissionHandler(repo port.SubmissionRepository) *GetSubmissionHandler {
	return &GetSubmissionHandler{repo: repo}
}

func (h *GetSubmissionHandler) Execute(ctx context.Context, actor *domain.Actor, id uuid.UUID) (*domain.Submission, error) {
	sub, err := h.repo.GetByID(ctx, id)
	if err != nil {
		return nil, notFound(err, "submission not found")
	}
	_, perms, err := commands.LoadOfferingAccess(ctx, h.repo, actor, sub.OfferingID)
	if err != nil {
		return nil, err
	}
	if !perms.View {
		return nil, domain.ErrForbidden
	}
	return sub, nil
}

// ---- GET /offerings/{id}/submissions ----

type ListSubmissionsHandler struct{ repo port.SubmissionRepository }

var _ port.ListSubmissionsUseCase = (*ListSubmissionsHandler)(nil)

func NewListSubmissionsHandler(repo port.SubmissionRepository) *ListSubmissionsHandler {
	return &ListSubmissionsHandler{repo: repo}
}

func (h *ListSubmissionsHandler) Execute(ctx context.Context, actor *domain.Actor, offeringID uuid.UUID) ([]domain.Submission, error) {
	_, perms, err := commands.LoadOfferingAccess(ctx, h.repo, actor, offeringID)
	if err != nil {
		return nil, err
	}
	if !perms.View {
		return nil, domain.ErrForbidden
	}
	return h.repo.ListByOffering(ctx, offeringID)
}

// ---- GET /offerings/{id}/portfolio.pdf ----

type GetPortfolioPDFHandler struct {
	repo    port.SubmissionRepository
	storage port.Storage
}

var _ port.GetPortfolioPDFUseCase = (*GetPortfolioPDFHandler)(nil)

func NewGetPortfolioPDFHandler(repo port.SubmissionRepository, storage port.Storage) *GetPortfolioPDFHandler {
	return &GetPortfolioPDFHandler{repo: repo, storage: storage}
}

func (h *GetPortfolioPDFHandler) Execute(ctx context.Context, actor *domain.Actor, offeringID uuid.UUID) (*port.PDFFile, error) {
	info, perms, err := commands.LoadOfferingAccess(ctx, h.repo, actor, offeringID)
	if err != nil {
		return nil, err
	}
	if !perms.Download {
		return nil, domain.ErrForbidden
	}
	sid, err := h.repo.LatestGenerated(ctx, offeringID)
	if err != nil {
		return nil, notFound(err, "no generated portfolio PDF for this offering")
	}
	f, fi, err := h.storage.Get(ctx, submissionPDFKey(sid))
	if err != nil {
		return nil, notFound(err, "no generated portfolio PDF for this offering")
	}
	return &port.PDFFile{File: f, Info: fi, Filename: commands.PortfolioFilename(*info)}, nil
}

// ---- GET /portfolios ----

type ListPortfoliosHandler struct{ repo port.SubmissionRepository }

var _ port.ListPortfoliosUseCase = (*ListPortfoliosHandler)(nil)

func NewListPortfoliosHandler(repo port.SubmissionRepository) *ListPortfoliosHandler {
	return &ListPortfoliosHandler{repo: repo}
}

// Execute runs two queries in total: the candidate rows (with latest submission) and the user's access rows.
func (h *ListPortfoliosHandler) Execute(ctx context.Context, actor *domain.Actor, f port.PortfolioFilter) ([]domain.PortfolioRow, error) {
	cands, err := h.repo.ListPortfolioRows(ctx, actor, f)
	if err != nil {
		return nil, err
	}
	var grants map[uuid.UUID]domain.AccessGrant
	if actor.Role != domain.RoleSuperAdmin && len(cands) > 0 {
		if grants, err = h.repo.Grants(ctx, actor.UserID, nil); err != nil {
			return nil, err
		}
	}
	rows := make([]domain.PortfolioRow, 0, len(cands))
	for _, c := range cands {
		var grant *domain.AccessGrant
		if g, ok := grants[c.Offering.ID]; ok {
			grant = &g
		}
		perms := actor.OfferingPermissions(c.Offering.Scope(), grant)
		if !perms.View {
			continue
		}
		rows = append(rows, domain.PortfolioRow{
			Offering: c.Offering, CanUpload: perms.Upload, CanDownload: perms.Download,
			Latest: c.Latest, PDFAvailable: c.PDFAvailable,
		})
	}
	return rows, nil
}

// ---- POST /portfolios/download ----

type DownloadPortfoliosHandler struct {
	repo    port.SubmissionRepository
	storage port.Storage
}

var _ port.DownloadPortfoliosUseCase = (*DownloadPortfoliosHandler)(nil)

func NewDownloadPortfoliosHandler(repo port.SubmissionRepository, storage port.Storage) *DownloadPortfoliosHandler {
	return &DownloadPortfoliosHandler{repo: repo, storage: storage}
}

// maxDownloadOfferings bounds one request (each id costs a zip entry and a line in MISSING.txt).
const maxDownloadOfferings = 1000

func (h *DownloadPortfoliosHandler) Plan(ctx context.Context, actor *domain.Actor, offeringIDs []uuid.UUID) (*port.DownloadPlan, error) {
	ids := dedupe(offeringIDs)
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: offering_ids must not be empty", domain.ErrInvalidInput)
	}
	if len(ids) > maxDownloadOfferings {
		return nil, fmt.Errorf("%w: too many offerings in one download", domain.ErrInvalidInput)
	}

	// Three queries for any number of offerings: infos, grants, newest generated submissions.
	infos, err := h.repo.ListOfferings(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]domain.OfferingInfo, len(infos))
	for _, i := range infos {
		byID[i.ID] = i
	}
	var grants map[uuid.UUID]domain.AccessGrant
	if actor.Role != domain.RoleSuperAdmin {
		if grants, err = h.repo.Grants(ctx, actor.UserID, ids); err != nil {
			return nil, err
		}
	}
	latest, err := h.repo.LatestGeneratedFor(ctx, ids)
	if err != nil {
		return nil, err
	}

	plan := &port.DownloadPlan{}
	used := map[string]int{}
	years := map[int]bool{}
	for _, id := range ids {
		info, ok := byID[id]
		if !ok {
			plan.Missing = append(plan.Missing, id.String()+": offering not found")
			continue
		}
		label := fmt.Sprintf("%s (%d, semester %d)", info.CourseCode, info.AcademicYear, info.Semester)
		var grant *domain.AccessGrant
		if g, ok := grants[id]; ok {
			grant = &g
		}
		if !actor.OfferingPermissions(info.Scope(), grant).Download {
			plan.Missing = append(plan.Missing, label+": no permission to download")
			continue
		}
		sid, ok := latest[id]
		if ok {
			if _, serr := h.storage.Stat(ctx, submissionPDFKey(sid)); serr != nil {
				if !errors.Is(serr, domain.ErrNotFound) {
					return nil, serr
				}
				ok = false
			}
		}
		if !ok {
			plan.Missing = append(plan.Missing, label+": no generated PDF")
			continue
		}
		years[info.AcademicYear] = true
		name := commands.PortfolioFilename(info)
		if n := used[name]; n > 0 { // same course code in two curricula
			name = strings.TrimSuffix(name, ".pdf") + "_" + safeSuffix(info.Curriculum.Code) + ".pdf"
		}
		used[name]++
		plan.Entries = append(plan.Entries, port.DownloadEntry{Name: name, StorageKey: submissionPDFKey(sid)})
	}
	if len(plan.Entries) == 0 {
		return nil, fmt.Errorf("%w: none of the selected portfolios is available", domain.ErrNotFound)
	}
	year := "selected"
	if len(years) == 1 {
		for y := range years {
			year = fmt.Sprint(y)
		}
	}
	plan.ZipName = "ABET_Course_Portfolios_" + year + ".zip"
	return plan, nil
}

func safeSuffix(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "x"
	}
	return b.String()
}

func dedupe(ids []uuid.UUID) []uuid.UUID {
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

// Write streams the planned PDFs (stored, not recompressed) and MISSING.txt straight into w.
func (h *DownloadPortfoliosHandler) Write(ctx context.Context, plan *port.DownloadPlan, w io.Writer) error {
	zw := zip.NewWriter(w)
	for _, e := range plan.Entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := h.writeEntry(ctx, zw, e); err != nil {
			slog.ErrorContext(ctx, "portfolio zip entry failed", "name", e.Name, "key", e.StorageKey, "error", err)
			return err
		}
	}
	if len(plan.Missing) > 0 {
		mw, err := zw.CreateHeader(&zip.FileHeader{Name: "MISSING.txt", Method: zip.Deflate, Modified: time.Now()})
		if err != nil {
			return err
		}
		text := "The following portfolios are not included in this archive:\r\n\r\n- " + strings.Join(plan.Missing, "\r\n- ") + "\r\n"
		if _, err := io.WriteString(mw, text); err != nil {
			return err
		}
	}
	return zw.Close()
}

func (h *DownloadPortfoliosHandler) writeEntry(ctx context.Context, zw *zip.Writer, e port.DownloadEntry) error {
	f, fi, err := h.storage.Get(ctx, e.StorageKey)
	if err != nil {
		return err
	}
	defer f.Close()
	hdr := &zip.FileHeader{Name: e.Name, Method: zip.Store, Modified: fi.LastModified}
	ew, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(ew, f)
	return err
}
