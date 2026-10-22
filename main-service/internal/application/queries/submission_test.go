package queries_test

import (
	"archive/zip"
	"bytes"
	"context"
	"io"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"abet/main-service/internal/application/queries"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
	"abet/main-service/internal/core/port/mocks"
	"abet/main-service/internal/testsupport/memstore"
)

var _ = Describe("submission queries", func() {
	var (
		ctrl  *gomock.Controller
		repo  *mocks.MockSubmissionRepository
		store *memstore.Store
		ctx   = context.Background()

		curA, curB   = uuid.New(), uuid.New()
		deptA, deptB = uuid.New(), uuid.New()
		offA, offB   = domain.OfferingInfo{ID: uuid.New(), CourseCode: "2301173", AcademicYear: 2026, Semester: 1}, domain.OfferingInfo{ID: uuid.New(), CourseCode: "2301174", AcademicYear: 2026, Semester: 2}

		super      = &domain.Actor{UserID: uuid.New(), Role: domain.RoleSuperAdmin}
		instructor = &domain.Actor{UserID: uuid.New(), Role: domain.RoleInstructor}
		staff      = &domain.Actor{UserID: uuid.New(), Role: domain.RoleStaff}
		admin      = &domain.Actor{UserID: uuid.New(), Role: domain.RoleAdmin, ScopeCurriculumIDs: []uuid.UUID{curA}}
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = mocks.NewMockSubmissionRepository(ctrl)
		store = memstore.New()
		offA.Curriculum, offA.DepartmentID = domain.Ref{ID: curA, Code: "CS"}, deptA
		offB.Curriculum, offB.DepartmentID = domain.Ref{ID: curB, Code: "MATH"}, deptB
		staff.DepartmentID = &deptA
	})
	AfterEach(func() { ctrl.Finish() })

	grants := func(m map[uuid.UUID]domain.AccessGrant) map[uuid.UUID]domain.AccessGrant { return m }

	Describe("GetSubmission", func() {
		It("returns the submission when the actor can view the offering", func() {
			sub := &domain.Submission{ID: uuid.New(), OfferingID: offA.ID}
			repo.EXPECT().GetByID(ctx, sub.ID).Return(sub, nil)
			repo.EXPECT().GetOffering(ctx, offA.ID).Return(&offA, nil)
			repo.EXPECT().Grants(ctx, instructor.UserID, []uuid.UUID{offA.ID}).Return(grants(map[uuid.UUID]domain.AccessGrant{offA.ID: {CanDownload: true}}), nil)
			got, err := queries.NewGetSubmissionHandler(repo).Execute(ctx, instructor, sub.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(BeIdenticalTo(sub))
		})
		It("forbids an instructor without an access row", func() {
			sub := &domain.Submission{ID: uuid.New(), OfferingID: offA.ID}
			repo.EXPECT().GetByID(ctx, sub.ID).Return(sub, nil)
			repo.EXPECT().GetOffering(ctx, offA.ID).Return(&offA, nil)
			repo.EXPECT().Grants(ctx, instructor.UserID, gomock.Any()).Return(grants(nil), nil)
			_, err := queries.NewGetSubmissionHandler(repo).Execute(ctx, instructor, sub.ID)
			Expect(err).To(MatchError(domain.ErrForbidden))
		})
		It("answers 404 for an unknown submission", func() {
			repo.EXPECT().GetByID(ctx, gomock.Any()).Return(nil, domain.ErrNotFound)
			_, err := queries.NewGetSubmissionHandler(repo).Execute(ctx, super, uuid.New())
			Expect(err).To(MatchError(domain.ErrNotFound))
		})
		It("lets an admin see submissions of offerings in scope without loading grants", func() {
			sub := &domain.Submission{ID: uuid.New(), OfferingID: offA.ID}
			repo.EXPECT().GetByID(ctx, sub.ID).Return(sub, nil)
			repo.EXPECT().GetOffering(ctx, offA.ID).Return(&offA, nil)
			repo.EXPECT().Grants(ctx, admin.UserID, gomock.Any()).Return(grants(nil), nil)
			_, err := queries.NewGetSubmissionHandler(repo).Execute(ctx, admin, sub.ID)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ListSubmissions", func() {
		It("lists for viewers and forbids others", func() {
			repo.EXPECT().GetOffering(ctx, offB.ID).Return(&offB, nil).Times(2)
			repo.EXPECT().Grants(ctx, admin.UserID, gomock.Any()).Return(grants(nil), nil)
			_, err := queries.NewListSubmissionsHandler(repo).Execute(ctx, admin, offB.ID) // curriculum B is out of scope
			Expect(err).To(MatchError(domain.ErrForbidden))

			repo.EXPECT().ListByOffering(ctx, offB.ID).Return([]domain.Submission{{}}, nil)
			items, err := queries.NewListSubmissionsHandler(repo).Execute(ctx, super, offB.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
		})
	})

	Describe("GetPortfolioPDF", func() {
		It("serves the newest generated PDF of a downloader with the contract filename", func() {
			sid := uuid.New()
			store.Seed("portfolios/"+sid.String()+".pdf", "%PDF-1.4")
			repo.EXPECT().GetOffering(ctx, offA.ID).Return(&offA, nil)
			repo.EXPECT().Grants(ctx, instructor.UserID, gomock.Any()).Return(grants(map[uuid.UUID]domain.AccessGrant{offA.ID: {CanDownload: true}}), nil)
			repo.EXPECT().LatestGenerated(ctx, offA.ID).Return(sid, nil)
			f, err := queries.NewGetPortfolioPDFHandler(repo, store).Execute(ctx, instructor, offA.ID)
			Expect(err).NotTo(HaveOccurred())
			defer f.File.Close()
			Expect(f.Filename).To(Equal("2301173_2026_S1_portfolio.pdf"))
			Expect(f.Info.Size).To(Equal(int64(len("%PDF-1.4"))))
			b, _ := io.ReadAll(f.File)
			Expect(string(b)).To(Equal("%PDF-1.4"))
		})
		It("forbids upload-only users (no download right)", func() {
			repo.EXPECT().GetOffering(ctx, offA.ID).Return(&offA, nil)
			repo.EXPECT().Grants(ctx, instructor.UserID, gomock.Any()).Return(grants(map[uuid.UUID]domain.AccessGrant{offA.ID: {CanUpload: true}}), nil)
			_, err := queries.NewGetPortfolioPDFHandler(repo, store).Execute(ctx, instructor, offA.ID)
			Expect(err).To(MatchError(domain.ErrForbidden))
		})
		It("answers 404 when nothing was generated, or the file vanished", func() {
			repo.EXPECT().GetOffering(ctx, offA.ID).Return(&offA, nil).Times(2)
			repo.EXPECT().LatestGenerated(ctx, offA.ID).Return(uuid.Nil, domain.ErrNotFound)
			_, err := queries.NewGetPortfolioPDFHandler(repo, store).Execute(ctx, super, offA.ID)
			Expect(err).To(MatchError(domain.ErrNotFound))
			repo.EXPECT().LatestGenerated(ctx, offA.ID).Return(uuid.New(), nil)
			_, err = queries.NewGetPortfolioPDFHandler(repo, store).Execute(ctx, super, offA.ID)
			Expect(err).To(MatchError(domain.ErrNotFound))
		})
		It("gives staff the department default download right", func() {
			sid := uuid.New()
			store.Seed("portfolios/"+sid.String()+".pdf", "%PDF-1.4")
			repo.EXPECT().GetOffering(ctx, offA.ID).Return(&offA, nil)
			repo.EXPECT().Grants(ctx, staff.UserID, gomock.Any()).Return(grants(nil), nil)
			repo.EXPECT().LatestGenerated(ctx, offA.ID).Return(sid, nil)
			f, err := queries.NewGetPortfolioPDFHandler(repo, store).Execute(ctx, staff, offA.ID)
			Expect(err).NotTo(HaveOccurred())
			_ = f.File.Close()
		})
	})

	Describe("ListPortfolios", func() {
		It("uses one candidate query and one grants query, and applies the permission table", func() {
			cands := []port.PortfolioCandidate{
				{Offering: offA, PDFAvailable: true, Latest: &domain.Submission{ID: uuid.New(), Status: domain.SubmissionGenerated}},
				{Offering: offB},
			}
			repo.EXPECT().ListPortfolioRows(ctx, staff, port.PortfolioFilter{}).Return(cands, nil)
			// staff: department A default download on offA; explicit upload row on offB
			repo.EXPECT().Grants(ctx, staff.UserID, []uuid.UUID(nil)).Return(grants(map[uuid.UUID]domain.AccessGrant{offB.ID: {CanUpload: true}}), nil)
			rows, err := queries.NewListPortfoliosHandler(repo).Execute(ctx, staff, port.PortfolioFilter{})
			Expect(err).NotTo(HaveOccurred())
			Expect(rows).To(HaveLen(2))
			Expect(rows[0].CanDownload).To(BeTrue())
			Expect(rows[0].CanUpload).To(BeFalse())
			Expect(rows[0].PDFAvailable).To(BeTrue())
			Expect(rows[0].Latest).NotTo(BeNil())
			Expect(rows[1].CanUpload).To(BeTrue())
			Expect(rows[1].CanDownload).To(BeFalse())
		})
		It("drops candidates the actor cannot view", func() {
			repo.EXPECT().ListPortfolioRows(ctx, instructor, gomock.Any()).Return([]port.PortfolioCandidate{{Offering: offA}, {Offering: offB}}, nil)
			repo.EXPECT().Grants(ctx, instructor.UserID, gomock.Any()).Return(grants(map[uuid.UUID]domain.AccessGrant{offB.ID: {CanDownload: true}}), nil)
			rows, err := queries.NewListPortfoliosHandler(repo).Execute(ctx, instructor, port.PortfolioFilter{})
			Expect(err).NotTo(HaveOccurred())
			Expect(rows).To(HaveLen(1))
			Expect(rows[0].Offering.ID).To(Equal(offB.ID))
		})
		It("does not load grants for super admins or empty results", func() {
			repo.EXPECT().ListPortfolioRows(ctx, super, gomock.Any()).Return([]port.PortfolioCandidate{{Offering: offA}}, nil)
			rows, err := queries.NewListPortfoliosHandler(repo).Execute(ctx, super, port.PortfolioFilter{})
			Expect(err).NotTo(HaveOccurred())
			Expect(rows[0].CanUpload && rows[0].CanDownload).To(BeTrue())

			repo.EXPECT().ListPortfolioRows(ctx, instructor, gomock.Any()).Return(nil, nil)
			rows, err = queries.NewListPortfoliosHandler(repo).Execute(ctx, instructor, port.PortfolioFilter{})
			Expect(err).NotTo(HaveOccurred())
			Expect(rows).To(BeEmpty())
		})
	})

	Describe("DownloadPortfolios", func() {
		var h *queries.DownloadPortfoliosHandler
		BeforeEach(func() { h = queries.NewDownloadPortfoliosHandler(repo, store) })

		It("rejects an empty list", func() {
			_, err := h.Plan(ctx, super, nil)
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("plans with three queries, lists forbidden/missing/ungenerated in MISSING.txt and streams a zip", func() {
			sidA := uuid.New()
			store.Seed("portfolios/"+sidA.String()+".pdf", "%PDF-A")
			unknown := uuid.New()
			offC := domain.OfferingInfo{ID: uuid.New(), CourseCode: "2301175", AcademicYear: 2026, Semester: 1,
				Curriculum: domain.Ref{ID: curA, Code: "CS"}, DepartmentID: deptA}
			ids := []uuid.UUID{offA.ID, offB.ID, offC.ID, unknown, offA.ID} // duplicate id is collapsed
			repo.EXPECT().ListOfferings(ctx, []uuid.UUID{offA.ID, offB.ID, offC.ID, unknown}).Return([]domain.OfferingInfo{offA, offB, offC}, nil)
			repo.EXPECT().Grants(ctx, instructor.UserID, gomock.Any()).Return(grants(map[uuid.UUID]domain.AccessGrant{
				offA.ID: {CanDownload: true}, offC.ID: {CanDownload: true}, // B: no row -> forbidden
			}), nil)
			repo.EXPECT().LatestGeneratedFor(ctx, gomock.Any()).Return(map[uuid.UUID]uuid.UUID{offA.ID: sidA, offB.ID: uuid.New()}, nil) // C has no PDF

			plan, err := h.Plan(ctx, instructor, ids)
			Expect(err).NotTo(HaveOccurred())
			Expect(plan.ZipName).To(Equal("ABET_Course_Portfolios_2026.zip"))
			Expect(plan.Entries).To(HaveLen(1))
			Expect(plan.Entries[0].Name).To(Equal("2301173_2026_S1_portfolio.pdf"))
			Expect(plan.Missing).To(HaveLen(3))

			var buf bytes.Buffer
			Expect(h.Write(ctx, plan, &buf)).To(Succeed())
			zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
			Expect(err).NotTo(HaveOccurred())
			names := []string{}
			for _, f := range zr.File {
				names = append(names, f.Name)
			}
			Expect(names).To(Equal([]string{"2301173_2026_S1_portfolio.pdf", "MISSING.txt"}))
			rc, _ := zr.File[0].Open()
			b, _ := io.ReadAll(rc)
			Expect(string(b)).To(Equal("%PDF-A"))
			rc, _ = zr.File[1].Open()
			b, _ = io.ReadAll(rc)
			Expect(string(b)).To(And(ContainSubstring("2301174"), ContainSubstring("no permission"), ContainSubstring("no generated PDF"), ContainSubstring("offering not found")))
		})

		It("answers 404 when nothing is available and names the zip 'selected' for mixed years", func() {
			repo.EXPECT().ListOfferings(ctx, gomock.Any()).Return([]domain.OfferingInfo{offA}, nil)
			repo.EXPECT().LatestGeneratedFor(ctx, gomock.Any()).Return(map[uuid.UUID]uuid.UUID{}, nil)
			_, err := h.Plan(ctx, super, []uuid.UUID{offA.ID})
			Expect(err).To(MatchError(domain.ErrNotFound))

			offB.AcademicYear = 2025
			sidA, sidB := uuid.New(), uuid.New()
			store.Seed("portfolios/"+sidA.String()+".pdf", "a")
			store.Seed("portfolios/"+sidB.String()+".pdf", "b")
			repo.EXPECT().ListOfferings(ctx, gomock.Any()).Return([]domain.OfferingInfo{offA, offB}, nil)
			repo.EXPECT().LatestGeneratedFor(ctx, gomock.Any()).Return(map[uuid.UUID]uuid.UUID{offA.ID: sidA, offB.ID: sidB}, nil)
			plan, err := h.Plan(ctx, super, []uuid.UUID{offA.ID, offB.ID})
			Expect(err).NotTo(HaveOccurred())
			Expect(plan.ZipName).To(Equal("ABET_Course_Portfolios_selected.zip"))
			Expect(plan.Missing).To(BeEmpty())
		})

		It("disambiguates identical file names (same course code in two curricula)", func() {
			twin := offA
			twin.ID = uuid.New()
			twin.Curriculum = domain.Ref{ID: curB, Code: "MATH"}
			sidA, sidT := uuid.New(), uuid.New()
			store.Seed("portfolios/"+sidA.String()+".pdf", "a")
			store.Seed("portfolios/"+sidT.String()+".pdf", "t")
			repo.EXPECT().ListOfferings(ctx, gomock.Any()).Return([]domain.OfferingInfo{offA, twin}, nil)
			repo.EXPECT().LatestGeneratedFor(ctx, gomock.Any()).Return(map[uuid.UUID]uuid.UUID{offA.ID: sidA, twin.ID: sidT}, nil)
			plan, err := h.Plan(ctx, super, []uuid.UUID{offA.ID, twin.ID})
			Expect(err).NotTo(HaveOccurred())
			Expect(plan.Entries[0].Name).NotTo(Equal(plan.Entries[1].Name))
		})

		It("stops writing when the request context is cancelled", func() {
			cctx, cancel := context.WithCancel(ctx)
			cancel()
			err := h.Write(cctx, &port.DownloadPlan{Entries: []port.DownloadEntry{{Name: "a.pdf", StorageKey: "portfolios/x.pdf"}}}, io.Discard)
			Expect(err).To(HaveOccurred())
		})
	})

})
