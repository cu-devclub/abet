package commands_test

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"abet/main-service/internal/application/commands"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port/mocks"
	"abet/main-service/internal/testsupport/memstore"
)

var _ = Describe("ReconcileBuildsHandler", func() {
	var (
		ctrl  *gomock.Controller
		repo  *mocks.MockSubmissionRepository
		store *memstore.Store
		ctx   = context.Background()
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = mocks.NewMockSubmissionRepository(ctrl)
		store = memstore.New()
	})
	AfterEach(func() { ctrl.Finish() })

	It("times out stale submissions with the BUILD_TIMEOUT issue and keeps the zip and the output dir for a late success", func() {
		id := uuid.New()
		store.Seed("uploads/"+id.String()+".zip", "zip")
		store.Seed("outputs/"+id.String()+"/a.pdf", "pdf")

		before := time.Now().UTC()
		repo.EXPECT().TimeOutStale(ctx, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, cutoff, at time.Time, issue domain.Issue) ([]uuid.UUID, error) {
				Expect(at.Sub(cutoff)).To(Equal(45 * time.Minute))
				Expect(at).To(BeTemporally(">=", before))
				Expect(issue.Severity).To(Equal(domain.SeverityError))
				Expect(issue.Stage).To(Equal(domain.StageBuild))
				Expect(issue.Code).To(Equal("BUILD_TIMEOUT"))
				Expect(issue.Title).To(Equal("PDF generation timed out. Please upload again."))
				Expect(issue.Description).To(Equal("The portfolio build service did not finish within 45 minutes."))
				Expect(issue.SuggestedFix).To(Equal("Upload the folder again. If this keeps happening, contact the system administrator."))
				return []uuid.UUID{id}, nil
			})

		n, err := commands.NewReconcileBuildsHandler(repo, store, 45*time.Minute).Execute(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(1))
		Expect(store.Has("uploads/" + id.String() + ".zip")).To(BeTrue())
		Expect(store.Has("outputs/" + id.String() + "/a.pdf")).To(BeTrue())
	})

	It("reports 0 when nothing is stale", func() {
		repo.EXPECT().TimeOutStale(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
		n, err := commands.NewReconcileBuildsHandler(repo, store, time.Hour).Execute(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(BeZero())
	})

	It("returns the repository error", func() {
		repo.EXPECT().TimeOutStale(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("db"))
		_, err := commands.NewReconcileBuildsHandler(repo, store, time.Hour).Execute(ctx)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("JanitorHandler", func() {
	var (
		ctrl  *gomock.Controller
		repo  *mocks.MockSubmissionRepository
		store *memstore.Store
		ctx   = context.Background()
		old   = 2 * time.Hour
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = mocks.NewMockSubmissionRepository(ctrl)
		store = memstore.New()
	})
	AfterEach(func() { ctrl.Finish() })

	// seed creates a zip ("zip"), an outputs group ("dir") or a portfolio pdf ("pdf") of the given age and returns its key.
	seed := func(id uuid.UUID, kind string, age time.Duration) string {
		var key string
		switch kind {
		case "zip":
			key = "uploads/" + id.String() + ".zip"
		case "dir":
			key = "outputs/" + id.String() + "/f.pdf"
		default:
			key = "portfolios/" + id.String() + ".pdf"
		}
		store.SeedAged(key, "x", age)
		return key
	}

	It("deletes old leftovers of terminal or unknown submissions and old PDFs without a generated row", func() {
		generated, failed, invalid, missing, orphanPDF := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
		zipFailed := seed(failed, "zip", old)
		dirInvalid := seed(invalid, "dir", old)
		zipMissing := seed(missing, "zip", old)
		dirGenerated := seed(generated, "dir", old)
		pdfGenerated := seed(generated, "pdf", old)
		pdfOrphan := seed(orphanPDF, "pdf", old)
		pdfFailed := seed(failed, "pdf", old)

		repo.EXPECT().Statuses(ctx, gomock.Any()).Return(map[uuid.UUID]domain.SubmissionStatus{
			generated: domain.SubmissionGenerated, failed: domain.SubmissionFailed, invalid: domain.SubmissionInvalid,
		}, nil)

		res, err := commands.NewJanitorHandler(repo, store, time.Hour).Execute(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(commands.JanitorResult{Uploads: 2, Outputs: 2, Portfolios: 2}))
		for _, gone := range []string{zipFailed, dirInvalid, zipMissing, dirGenerated, pdfOrphan, pdfFailed} {
			Expect(store.Has(gone)).To(BeFalse(), gone)
		}
		Expect(store.Has(pdfGenerated)).To(BeTrue())
	})

	It("never touches queued or processing submissions", func() {
		queued, processing := uuid.New(), uuid.New()
		z := seed(queued, "zip", old)
		d := seed(queued, "dir", old)
		z2 := seed(processing, "zip", old)
		repo.EXPECT().Statuses(ctx, gomock.Any()).Return(map[uuid.UUID]domain.SubmissionStatus{
			queued: domain.SubmissionQueued, processing: domain.SubmissionProcessing,
		}, nil)
		res, err := commands.NewJanitorHandler(repo, store, time.Hour).Execute(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(commands.JanitorResult{}))
		Expect(store.Has(z)).To(BeTrue())
		Expect(store.Has(d)).To(BeTrue())
		Expect(store.Has(z2)).To(BeTrue())
	})

	It("leaves recent files alone (an upload may still be in flight) and foreign names", func() {
		id := uuid.New()
		recent := seed(id, "zip", 5*time.Minute)
		store.SeedAged("uploads/notes.txt", "x", old)
		store.SeedAged("outputs/not-a-uuid/f.pdf", "x", old)

		// no Statuses call is expected: nothing is both old and ours
		res, err := commands.NewJanitorHandler(repo, store, time.Hour).Execute(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(commands.JanitorResult{}))
		Expect(store.Has(recent)).To(BeTrue())
		Expect(store.Has("uploads/notes.txt")).To(BeTrue())
		Expect(store.Has("outputs/not-a-uuid/f.pdf")).To(BeTrue())
	})

	It("does nothing when the storage folders do not exist yet", func() {
		res, err := commands.NewJanitorHandler(repo, store, time.Hour).Execute(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(commands.JanitorResult{}))
	})

	It("returns the repository error and deletes nothing", func() {
		id := uuid.New()
		z := seed(id, "zip", old)
		repo.EXPECT().Statuses(ctx, gomock.Any()).Return(nil, errors.New("db"))
		_, err := commands.NewJanitorHandler(repo, store, time.Hour).Execute(ctx)
		Expect(err).To(HaveOccurred())
		Expect(store.Has(z)).To(BeTrue())
	})

	It("judges an outputs group by its newest object", func() {
		id := uuid.New()
		store.SeedAged("outputs/"+id.String()+"/old.pdf", "x", old)
		store.SeedAged("outputs/"+id.String()+"/new.pdf", "x", 5*time.Minute)
		res, err := commands.NewJanitorHandler(repo, store, time.Hour).Execute(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(commands.JanitorResult{}))
		Expect(store.Keys("outputs/")).To(HaveLen(2))
	})
})
