package commands

import (
	"context"
	"fmt"

	"abet/main-service/internal/application/importplan"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// importScope validates the common import preconditions and parses the CSV.
type importScope struct {
	repo           port.ImportRepository
	currs          port.CurriculumRepository
	allowedDomains []string
	maxBytes       int64
}

type importPrepared struct {
	curriculum *domain.Curriculum
	parsed     importplan.ParseResult
	emails     []string
}

func (s *importScope) prepare(ctx context.Context, actor *domain.Actor, in port.ImportInput) (*importPrepared, error) {
	if !actor.CanManageCurriculum(in.CurriculumID) {
		return nil, domain.ErrForbidden
	}
	if err := domain.ValidateAcademicYear(in.AcademicYear); err != nil {
		return nil, err
	}
	if int64(len(in.Data)) > s.maxBytes {
		return nil, fmt.Errorf("%w: the CSV file is larger than %d MB", domain.ErrTooLarge, s.maxBytes>>20)
	}
	cur, err := s.currs.GetByID(ctx, in.CurriculumID)
	if err != nil {
		return nil, notFoundAs(err, "curriculum not found")
	}
	parsed := importplan.ParseCSV(in.Data, s.allowedDomains)
	return &importPrepared{curriculum: cur, parsed: parsed, emails: distinctEmails(parsed.Rows)}, nil
}

// build runs the pure planner and attaches the parse errors and the row count.
func (p *importPrepared) build(state *domain.ImportState) domain.ImportPlan {
	plan := importplan.BuildImportPlan(p.parsed.Rows, *state)
	plan.Summary.Rows = p.parsed.RowCount
	plan.Errors = append(plan.Errors, p.parsed.Errors...)
	return plan
}

func distinctEmails(rows []domain.ImportRow) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, r := range rows {
		if r.Email != "" && !seen[r.Email] {
			seen[r.Email] = true
			out = append(out, r.Email)
		}
	}
	return out
}

// PreviewImportHandler implements port.PreviewImportUseCase.
type PreviewImportHandler struct{ importScope }

var _ port.PreviewImportUseCase = (*PreviewImportHandler)(nil)

func NewPreviewImportHandler(repo port.ImportRepository, currs port.CurriculumRepository, allowedDomains []string, maxImportBytes int64) *PreviewImportHandler {
	return &PreviewImportHandler{importScope{repo: repo, currs: currs, allowedDomains: allowedDomains, maxBytes: maxImportBytes}}
}

func (h *PreviewImportHandler) Execute(ctx context.Context, actor *domain.Actor, in port.ImportInput) (*domain.ImportPlan, error) {
	p, err := h.prepare(ctx, actor, in)
	if err != nil {
		return nil, err
	}
	state, err := h.repo.LoadState(ctx, in.CurriculumID, in.AcademicYear, p.emails)
	if err != nil {
		return nil, err
	}
	plan := p.build(state)
	return &plan, nil
}

// ApplyImportHandler implements port.ApplyImportUseCase.
type ApplyImportHandler struct{ importScope }

var _ port.ApplyImportUseCase = (*ApplyImportHandler)(nil)

func NewApplyImportHandler(repo port.ImportRepository, currs port.CurriculumRepository, allowedDomains []string, maxImportBytes int64) *ApplyImportHandler {
	return &ApplyImportHandler{importScope{repo: repo, currs: currs, allowedDomains: allowedDomains, maxBytes: maxImportBytes}}
}

func (h *ApplyImportHandler) Execute(ctx context.Context, actor *domain.Actor, in port.ImportInput) (*domain.ImportPlan, error) {
	p, err := h.prepare(ctx, actor, in)
	if err != nil {
		return nil, err
	}
	// The plan is always recomputed inside the transaction from the locked state; the preview is never trusted.
	return h.repo.Apply(ctx, port.ApplyImportParams{
		Curriculum: p.curriculum, AcademicYear: in.AcademicYear, Emails: p.emails,
		Build: p.build,
	})
}
