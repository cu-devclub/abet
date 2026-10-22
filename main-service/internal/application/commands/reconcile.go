package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
	"abet/main-service/internal/core/validation"
)

// ReconcileBuildsHandler fails submissions that have been waiting for the portfolio worker for too long, so no row
// stays "Processing" forever (a worker that is down, crashed or lost its event).
type ReconcileBuildsHandler struct {
	repo    port.SubmissionRepository
	storage port.Storage
	timeout time.Duration
	now     func() time.Time
}

func NewReconcileBuildsHandler(repo port.SubmissionRepository, storage port.Storage, timeout time.Duration) *ReconcileBuildsHandler {
	return &ReconcileBuildsHandler{repo: repo, storage: storage, timeout: timeout, now: func() time.Time { return time.Now().UTC() }}
}

// Execute times out the stale queued submissions and returns how many it failed.
// The upload zip and output directory are deliberately kept: a job still waiting in the queue (or being built)
// can finish late and its success is accepted. The janitor removes them once they are old enough.
func (h *ReconcileBuildsHandler) Execute(ctx context.Context) (int, error) {
	now := h.now()
	minutes := int(h.timeout / time.Minute)
	issue := validation.NewIssue(domain.SeverityError, validation.CodeBuildTimeout, "PDF generation timed out. Please upload again.", "")
	issue.Description = fmt.Sprintf("The portfolio build service did not finish within %d minutes.", minutes)
	issue.SuggestedFix = "Upload the folder again. If this keeps happening, contact the system administrator."

	ids, err := h.repo.TimeOutStale(ctx, now.Add(-h.timeout), now, issue)
	if err != nil {
		return 0, fmt.Errorf("time out stale submissions: %w", err)
	}
	for _, id := range ids {
		slog.WarnContext(ctx, "portfolio build timed out", "submission_id", id, "timeout_minutes", minutes)
	}
	return len(ids), nil
}

// JanitorResult counts what one janitor run deleted.
type JanitorResult struct {
	Uploads    int
	Outputs    int
	Portfolios int
}

// JanitorHandler removes storage leftovers of finished (or unknown) submissions. Best effort: individual failures
// are logged and skipped. It never touches files of submissions that are still queued/processing.
type JanitorHandler struct {
	repo    port.SubmissionRepository
	storage port.Storage
	minAge  time.Duration
	now     func() time.Time
}

func NewJanitorHandler(repo port.SubmissionRepository, storage port.Storage, minAge time.Duration) *JanitorHandler {
	return &JanitorHandler{repo: repo, storage: storage, minAge: minAge, now: func() time.Time { return time.Now().UTC() }}
}

type janitorItem struct {
	id     uuid.UUID
	keys   []string // objects to delete (the zip, the PDF, or every object of an outputs/<id>/ prefix)
	prefix string   // non-empty for an outputs/<id>/ group
	kind   int
}

const (
	kindUpload = iota
	kindOutput
	kindPortfolio
)

func (h *JanitorHandler) Execute(ctx context.Context) (JanitorResult, error) {
	var res JanitorResult
	cutoff := h.now().Add(-h.minAge)
	var items []janitorItem
	var ids []uuid.UUID

	// uploads/<id>.zip and portfolios/<id>.pdf: one object each, judged by its own age.
	collectFiles := func(dir, suffix string, kind int) error {
		objs, err := h.storage.List(ctx, dir+"/", true)
		if err != nil {
			return err
		}
		for _, o := range objs {
			name := strings.TrimPrefix(o.Key, dir+"/")
			if strings.Contains(name, "/") || !strings.HasSuffix(name, suffix) || !o.LastModified.Before(cutoff) {
				continue
			}
			id, err := uuid.Parse(strings.TrimSuffix(name, suffix))
			if err != nil {
				continue // not ours
			}
			items = append(items, janitorItem{id: id, keys: []string{o.Key}, kind: kind})
			ids = append(ids, id)
		}
		return nil
	}
	if err := collectFiles("uploads", ".zip", kindUpload); err != nil {
		return res, err
	}
	if err := collectFiles("portfolios", ".pdf", kindPortfolio); err != nil {
		return res, err
	}

	// outputs/<id>/...: one group per submission, as old as its newest object.
	objs, err := h.storage.List(ctx, "outputs/", true)
	if err != nil {
		return res, err
	}
	newest := map[uuid.UUID]time.Time{}
	for _, o := range objs {
		first, _, ok := strings.Cut(strings.TrimPrefix(o.Key, "outputs/"), "/")
		if !ok {
			continue
		}
		id, err := uuid.Parse(first)
		if err != nil {
			continue // not ours
		}
		if o.LastModified.After(newest[id]) {
			newest[id] = o.LastModified
		}
	}
	for id, mod := range newest {
		if mod.Before(cutoff) {
			items = append(items, janitorItem{id: id, prefix: submissionOutputPrefix(id), kind: kindOutput})
			ids = append(ids, id)
		}
	}
	if len(items) == 0 {
		return res, nil
	}

	statuses, err := h.repo.Statuses(ctx, ids)
	if err != nil {
		return res, fmt.Errorf("load submission statuses: %w", err)
	}
	for _, it := range items {
		st, known := statuses[it.id]
		if it.kind == kindPortfolio {
			if known && st == domain.SubmissionGenerated {
				continue
			}
		} else if known && (st == domain.SubmissionQueued || st == domain.SubmissionProcessing) {
			continue
		}
		var err error
		if it.prefix != "" {
			err = h.storage.DeletePrefix(ctx, it.prefix)
		} else {
			for _, k := range it.keys {
				if err = h.storage.Delete(ctx, k); err != nil {
					break
				}
			}
		}
		if err != nil {
			slog.WarnContext(ctx, "janitor could not delete", "id", it.id, "kind", it.kind, "error", err)
			continue
		}
		switch it.kind {
		case kindUpload:
			res.Uploads++
		case kindOutput:
			res.Outputs++
		default:
			res.Portfolios++
		}
	}
	return res, nil
}
