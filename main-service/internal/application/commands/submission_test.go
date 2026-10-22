package commands_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"abet/main-service/internal/application/commands"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
	"abet/main-service/internal/core/port/mocks"
	"abet/main-service/internal/testsupport/memstore"
)

// validPackageZip zips the fixture of the validator (identity course code 1234567).
func validPackageZip(mutate func(files map[string][]byte)) []byte {
	files := map[string][]byte{}
	root := filepath.Join("..", "..", "core", "validation", "testdata", "package")
	Expect(filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		rel, _ := filepath.Rel(root, p)
		files[filepath.ToSlash(rel)] = b
		return err
	})).To(Succeed())
	if mutate != nil {
		mutate(files)
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range files {
		w, err := zw.Create(name)
		Expect(err).NotTo(HaveOccurred())
		_, _ = w.Write(data)
	}
	Expect(zw.Close()).To(Succeed())
	return buf.Bytes()
}

var _ = Describe("BuildProgramName / PortfolioFilename", func() {
	It("lower-cases the curriculum code and replaces other characters", func() {
		Expect(commands.BuildProgramName("CS", 2026)).To(Equal("cs-2026"))
		Expect(commands.BuildProgramName(" MATH_CS (Int'l) ", 2025)).To(Equal("math-cs--int-l--2025"))
		Expect(commands.BuildProgramName("วิศวะ", 2026)).To(Equal("------2026"))
		Expect(commands.BuildProgramName("", 2026)).To(Equal("program-2026"))
	})
	It("builds the download filename", func() {
		Expect(commands.PortfolioFilename(domain.OfferingInfo{CourseCode: "2301173", AcademicYear: 2026, Semester: 2})).
			To(Equal("2301173_2026_S2_portfolio.pdf"))
		Expect(commands.PortfolioFilename(domain.OfferingInfo{CourseCode: "../x y", AcademicYear: 2026, Semester: 1})).
			To(Equal("x-y_2026_S1_portfolio.pdf"))
	})
})

var _ = Describe("UploadSubmissionHandler", func() {
	var (
		ctrl      *gomock.Controller
		repo      *mocks.MockSubmissionRepository
		publisher *mocks.MockJobPublisher
		store     *memstore.Store
		tmpDir    string
		h         *commands.UploadSubmissionHandler
		ctx       = context.Background()

		offeringID = uuid.New()
		currID     = uuid.New()
		deptID     = uuid.New()
		info       *domain.OfferingInfo
		instructor = &domain.Actor{UserID: uuid.New(), Email: "inst@chula.ac.th", Role: domain.RoleInstructor}
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = mocks.NewMockSubmissionRepository(ctrl)
		publisher = mocks.NewMockJobPublisher(ctrl)
		store = memstore.New()
		tmpDir = GinkgoT().TempDir()
		h = commands.NewUploadSubmissionHandler(repo, store, publisher, tmpDir)
		info = &domain.OfferingInfo{
			ID: offeringID, CourseCode: "1234567", CourseName: "Calculus", AcademicYear: 2026, Semester: 1,
			Curriculum: domain.Ref{ID: currID, Code: "CS", Name: "Computer Science"}, DepartmentID: deptID,
		}
		repo.EXPECT().GetOffering(ctx, offeringID).Return(info, nil).AnyTimes()
	})
	AfterEach(func() {
		ctrl.Finish()
		entries, _ := os.ReadDir(tmpDir) // the spooled upload is always removed
		Expect(entries).To(BeEmpty())
	})

	allowUpload := func() {
		repo.EXPECT().Grants(ctx, instructor.UserID, []uuid.UUID{offeringID}).
			Return(map[uuid.UUID]domain.AccessGrant{offeringID: {CanUpload: true, CanDownload: true}}, nil)
	}
	uploaded := func(sid uuid.UUID) string { return "uploads/" + sid.String() + ".zip" }

	It("answers 404 for an unknown offering", func() {
		repo2 := mocks.NewMockSubmissionRepository(ctrl)
		repo2.EXPECT().GetOffering(ctx, offeringID).Return(nil, domain.ErrNotFound)
		_, err := commands.NewUploadSubmissionHandler(repo2, store, publisher, tmpDir).Execute(ctx, instructor,
			port.UploadSubmissionInput{OfferingID: offeringID, Filename: "a.zip", Body: strings.NewReader("x")})
		Expect(err).To(MatchError(domain.ErrNotFound))
	})

	It("forbids users without upload permission and never touches storage", func() {
		repo.EXPECT().Grants(ctx, instructor.UserID, []uuid.UUID{offeringID}).
			Return(map[uuid.UUID]domain.AccessGrant{offeringID: {CanDownload: true}}, nil)
		_, err := h.Execute(ctx, instructor, port.UploadSubmissionInput{OfferingID: offeringID, Filename: "a.zip", Body: strings.NewReader("x")})
		Expect(err).To(MatchError(domain.ErrForbidden))
		Expect(store.Keys("")).To(BeEmpty())
	})

	It("rejects a file that is not a zip with 400 and deletes it", func() {
		allowUpload()
		_, err := h.Execute(ctx, instructor, port.UploadSubmissionInput{OfferingID: offeringID, Filename: "a.zip", Body: strings.NewReader("not a zip at all")})
		Expect(err).To(MatchError(domain.ErrInvalidInput))
		Expect(store.Keys("")).To(BeEmpty())
	})

	It("passes a size-limit error from the body through (413)", func() {
		allowUpload()
		_, err := h.Execute(ctx, instructor, port.UploadSubmissionInput{OfferingID: offeringID, Filename: "a.zip",
			Body: &failingReader{err: domain.ErrTooLarge}})
		Expect(err).To(MatchError(domain.ErrTooLarge))
		Expect(store.Keys("")).To(BeEmpty())
	})

	It("stores a valid zip, queues it and publishes the v2 job (object keys)", func() {
		allowUpload()
		zipBytes := validPackageZip(nil)
		var created *domain.Submission
		repo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, s *domain.Submission) error {
			created = s
			return nil
		})
		var job domain.BuildJob
		publisher.EXPECT().Publish(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, j domain.BuildJob) error {
			job = j
			return nil
		})
		repo.EXPECT().GetByID(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID) (*domain.Submission, error) {
			return created, nil
		})

		sub, err := h.Execute(ctx, instructor, port.UploadSubmissionInput{OfferingID: offeringID, Filename: `C:\fakepath\mine.zip`, Body: bytes.NewReader(zipBytes)})
		Expect(err).NotTo(HaveOccurred())
		Expect(sub.Status).To(Equal(domain.SubmissionQueued))
		Expect(created.OriginalFilename).To(Equal("mine.zip"))
		Expect(created.SizeBytes).To(Equal(int64(len(zipBytes))))
		Expect(created.ErrorCount).To(BeZero())
		Expect(created.UploadedBy.ID).To(Equal(instructor.UserID))
		Expect(created.CompletedAt).To(BeNil())

		Expect(job.JobID).To(Equal(created.JobID))
		Expect(job.ProgramName).To(Equal("cs-2026"))
		Expect(job.DeleteZipOnCompletion).To(BeTrue())
		Expect(job.ZipKey).To(Equal(uploaded(created.ID)))
		Expect(job.OutputPrefix).To(Equal("outputs/" + created.ID.String() + "/"))
		Expect(job.CoverKey).To(BeEmpty())
		Expect(store.Bytes(uploaded(created.ID))).To(Equal(zipBytes))
	})

	It("adds the curriculum cover to the job when one is stored", func() {
		allowUpload()
		store.Seed("covers/"+currID.String()+".pdf", "%PDF-1.4")
		repo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
		var job domain.BuildJob
		publisher.EXPECT().Publish(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, j domain.BuildJob) error { job = j; return nil })
		repo.EXPECT().GetByID(ctx, gomock.Any()).Return(&domain.Submission{Status: domain.SubmissionQueued}, nil)
		_, err := h.Execute(ctx, instructor, port.UploadSubmissionInput{OfferingID: offeringID, Filename: "a.zip", Body: bytes.NewReader(validPackageZip(nil))})
		Expect(err).NotTo(HaveOccurred())
		Expect(job.CoverKey).To(Equal("covers/" + currID.String() + ".pdf"))
	})

	It("marks a package with errors invalid, never stores the zip and publishes nothing", func() {
		allowUpload()
		zipBytes := validPackageZip(func(f map[string][]byte) {
			delete(f, "02-textbook/textbook-part01.pdf")
			delete(f, "02-textbook/textbook-part02.pdf")
			delete(f, "02-textbook/textbook-part03.pdf")
			delete(f, "20-assignments/asg02-low.pdf") // + one warning
		})
		var created *domain.Submission
		repo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, s *domain.Submission) error { created = s; return nil })
		repo.EXPECT().GetByID(ctx, gomock.Any()).DoAndReturn(func(context.Context, uuid.UUID) (*domain.Submission, error) { return created, nil })

		sub, err := h.Execute(ctx, instructor, port.UploadSubmissionInput{OfferingID: offeringID, Filename: "a.zip", Body: bytes.NewReader(zipBytes)})
		Expect(err).NotTo(HaveOccurred())
		Expect(sub.Status).To(Equal(domain.SubmissionInvalid))
		Expect(sub.ErrorCount).To(Equal(1))
		Expect(sub.WarningCount).To(Equal(1))
		Expect(sub.CompletedAt).NotTo(BeNil())
		Expect(sub.Issues[0].Title).To(Equal("Missing required folder: 02-textbook"))
		Expect(store.Keys("")).To(BeEmpty())
	})

	It("checks the course code of the identity CSV against the offering", func() {
		allowUpload()
		info.CourseCode = "9999999"
		var created *domain.Submission
		repo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, s *domain.Submission) error { created = s; return nil })
		repo.EXPECT().GetByID(ctx, gomock.Any()).DoAndReturn(func(context.Context, uuid.UUID) (*domain.Submission, error) { return created, nil })
		sub, err := h.Execute(ctx, instructor, port.UploadSubmissionInput{OfferingID: offeringID, Filename: "a.zip", Body: bytes.NewReader(validPackageZip(nil))})
		Expect(err).NotTo(HaveOccurred())
		Expect(sub.Status).To(Equal(domain.SubmissionInvalid))
		Expect(sub.Issues[0].Code).To(Equal("IDENTITY_COURSE_MISMATCH"))
	})

	It("records QUEUE_UNAVAILABLE and removes the files when publishing fails", func() {
		allowUpload()
		var created *domain.Submission
		repo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, s *domain.Submission) error { created = s; return nil })
		publisher.EXPECT().Publish(ctx, gomock.Any()).Return(domain.ErrServiceUnavailable)
		repo.EXPECT().CompleteFailure(ctx, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, id uuid.UUID, issues []domain.Issue, _ time.Time) (bool, error) {
				Expect(id).To(Equal(created.ID))
				Expect(issues).To(HaveLen(1))
				Expect(issues[0].Code).To(Equal("QUEUE_UNAVAILABLE"))
				Expect(issues[0].Stage).To(Equal(domain.StageBuild))
				Expect(issues[0].Title).To(Equal("Portfolio build service unavailable. Please try again later."))
				return true, nil
			})
		repo.EXPECT().GetByID(ctx, gomock.Any()).Return(&domain.Submission{Status: domain.SubmissionFailed}, nil)

		sub, err := h.Execute(ctx, instructor, port.UploadSubmissionInput{OfferingID: offeringID, Filename: "a.zip", Body: bytes.NewReader(validPackageZip(nil))})
		Expect(err).NotTo(HaveOccurred())
		Expect(sub.Status).To(Equal(domain.SubmissionFailed))
		Expect(store.Has(uploaded(created.ID))).To(BeFalse())
		Expect(store.Keys("")).To(BeEmpty())
	})

	It("removes the uploaded object when the database insert fails", func() {
		allowUpload()
		repo.EXPECT().Create(ctx, gomock.Any()).Return(errors.New("db down"))
		_, err := h.Execute(ctx, instructor, port.UploadSubmissionInput{OfferingID: offeringID, Filename: "a.zip", Body: bytes.NewReader(validPackageZip(nil))})
		Expect(err).To(HaveOccurred())
		Expect(store.Keys("")).To(BeEmpty())
	})

	It("lets a super admin upload without loading grants", func() {
		super := &domain.Actor{UserID: uuid.New(), Role: domain.RoleSuperAdmin}
		repo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
		publisher.EXPECT().Publish(ctx, gomock.Any()).Return(nil)
		repo.EXPECT().GetByID(ctx, gomock.Any()).Return(&domain.Submission{}, nil)
		_, err := h.Execute(ctx, super, port.UploadSubmissionInput{OfferingID: offeringID, Filename: "a.zip", Body: bytes.NewReader(validPackageZip(nil))})
		Expect(err).NotTo(HaveOccurred())
	})
})

type failingReader struct{ err error }

func (f *failingReader) Read([]byte) (int, error) { return 0, f.err }

var _ = Describe("ProcessBuildStatusHandler", func() {
	var (
		ctrl  *gomock.Controller
		repo  *mocks.MockSubmissionRepository
		store *memstore.Store
		h     *commands.ProcessBuildStatusHandler
		ctx   = context.Background()
		sub   *domain.Submission
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = mocks.NewMockSubmissionRepository(ctrl)
		store = memstore.New()
		h = commands.NewProcessBuildStatusHandler(repo, store)
		sub = &domain.Submission{ID: uuid.New(), OfferingID: uuid.New(), JobID: uuid.New(), Status: domain.SubmissionQueued}
	})
	AfterEach(func() { ctrl.Finish() })

	seedWorkerFiles := func(pdfs ...string) {
		store.Seed("uploads/"+sub.ID.String()+".zip", "zip")
		for _, p := range pdfs {
			store.Seed("outputs/"+sub.ID.String()+"/"+p, "%PDF-1.4 "+p)
		}
	}
	outPrefix := func() string { return "outputs/" + sub.ID.String() + "/" }
	outKey := func(name string) string { return outPrefix() + name }
	pdfKey := func(id uuid.UUID) string { return "portfolios/" + id.String() + ".pdf" }

	It("acks (nil) events with an unknown or malformed job id", func() {
		Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "success", JobID: "not-a-uuid"})).To(Succeed())
		repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(nil, domain.ErrNotFound)
		Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "success", JobID: sub.JobID.String()})).To(Succeed())
	})

	It("returns an error (requeue) when the lookup fails", func() {
		repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(nil, errors.New("db down"))
		Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "success", JobID: sub.JobID.String()})).NotTo(Succeed())
	})

	It("ignores events of finished submissions and unknown statuses", func() {
		sub.Status = domain.SubmissionGenerated
		repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
		Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "success", JobID: sub.JobID.String()})).To(Succeed())

		sub.Status = domain.SubmissionQueued
		repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
		Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "processing", JobID: sub.JobID.String()})).To(Succeed())
	})

	Describe("late results after the reconciler timed the submission out", func() {
		timedOut := func() *domain.Submission {
			full := *sub
			full.Status = domain.SubmissionFailed
			full.Issues = []domain.Issue{{Severity: domain.SeverityError, Code: "BUILD_TIMEOUT"}}
			return &full
		}
		successEvent := func() domain.BuildStatusEvent {
			return domain.BuildStatusEvent{Status: "success", JobID: sub.JobID.String(), ExportedPDFKeys: []string{outKey("x-main.pdf")}}
		}

		It("accepts a success event for a BUILD_TIMEOUT failure and handles the PDF normally", func() {
			seedWorkerFiles("x-main.pdf")
			sub.Status = domain.SubmissionFailed
			repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
			repo.EXPECT().GetByID(ctx, sub.ID).Return(timedOut(), nil)
			repo.EXPECT().CompleteSuccess(ctx, sub.ID, int64(len("%PDF-1.4 x-main.pdf")), gomock.Any()).
				Return(port.SuccessOutcome{Applied: true}, nil)
			Expect(h.Execute(ctx, successEvent())).To(Succeed())
			Expect(store.Has(pdfKey(sub.ID))).To(BeTrue())
			Expect(store.Keys(outPrefix())).To(BeEmpty())
		})

		It("ignores a success event for a submission that failed for another reason", func() {
			seedWorkerFiles("x-main.pdf")
			sub.Status = domain.SubmissionFailed
			other := *sub
			other.Issues = []domain.Issue{{Severity: domain.SeverityError, Code: "BUILD_FAILED"}}
			repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
			repo.EXPECT().GetByID(ctx, sub.ID).Return(&other, nil)
			Expect(h.Execute(ctx, successEvent())).To(Succeed())
			Expect(store.Has(pdfKey(sub.ID))).To(BeFalse())
		})

		It("retries (error) when the issues cannot be loaded", func() {
			sub.Status = domain.SubmissionFailed
			repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
			repo.EXPECT().GetByID(ctx, sub.ID).Return(nil, errors.New("db down"))
			Expect(h.Execute(ctx, successEvent())).NotTo(Succeed())
		})

		It("ignores a failed event for an already terminal submission", func() {
			sub.Status = domain.SubmissionFailed
			repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
			Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "failed", JobID: sub.JobID.String(), Error: "boom"})).To(Succeed())
		})
	})

	Describe("success", func() {
		It("moves the -main.pdf to portfolios/, completes the row, cleans up and deletes older PDFs", func() {
			seedWorkerFiles("other.pdf", "2301173-Calculus-main.pdf")
			older := uuid.New()
			store.Seed(pdfKey(older), "old")

			repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
			repo.EXPECT().CompleteSuccess(ctx, sub.ID, int64(len("%PDF-1.4 2301173-Calculus-main.pdf")), gomock.Any()).
				Return(port.SuccessOutcome{Applied: true, OlderIDs: []uuid.UUID{older}}, nil)

			ev := domain.BuildStatusEvent{Status: "success", JobID: sub.JobID.String(), ExportedPDFKeys: []string{
				outKey("other.pdf"),
				outKey("2301173-Calculus-main.pdf"),
			}}
			Expect(h.Execute(ctx, ev)).To(Succeed())

			Expect(string(store.Bytes(pdfKey(sub.ID)))).To(Equal("%PDF-1.4 2301173-Calculus-main.pdf"))
			Expect(store.Keys(outPrefix())).To(BeEmpty())
			Expect(store.Has("uploads/" + sub.ID.String() + ".zip")).To(BeFalse())
			Expect(store.Has(pdfKey(older))).To(BeFalse())
		})

		It("falls back to the first pdf when no -main.pdf is listed", func() {
			seedWorkerFiles("only.pdf")
			repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
			repo.EXPECT().CompleteSuccess(ctx, sub.ID, gomock.Any(), gomock.Any()).Return(port.SuccessOutcome{Applied: true}, nil)
			Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "success", JobID: sub.JobID.String(), ExportedPDFKeys: []string{outKey("only.pdf"), outKey("report.md")}})).To(Succeed())
			Expect(store.Has(pdfKey(sub.ID))).To(BeTrue())
		})

		It("drops the new PDF again when a newer submission already won", func() {
			seedWorkerFiles("a-main.pdf")
			repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
			repo.EXPECT().CompleteSuccess(ctx, sub.ID, gomock.Any(), gomock.Any()).Return(port.SuccessOutcome{Applied: true, Superseded: true}, nil)
			Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "success", JobID: sub.JobID.String(), ExportedPDFKeys: []string{outKey("a-main.pdf")}})).To(Succeed())
			Expect(store.Has(pdfKey(sub.ID))).To(BeFalse())
		})

		It("ignores exported keys outside the job's own outputs prefix", func() {
			seedWorkerFiles()
			store.Seed("outputs/"+uuid.NewString()+"/evil-main.pdf", "%PDF-evil")
			repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
			repo.EXPECT().CompleteFailure(ctx, sub.ID, gomock.Any(), gomock.Any()).Return(true, nil)
			ev := domain.BuildStatusEvent{Status: "success", JobID: sub.JobID.String(),
				ExportedPDFKeys: []string{store.Keys("outputs/")[0], "portfolios/" + uuid.NewString() + ".pdf", outKey("../x.pdf")}}
			Expect(h.Execute(ctx, ev)).To(Succeed())
			Expect(store.Has(pdfKey(sub.ID))).To(BeFalse())
		})

		It("fails with BUILD_NO_OUTPUT when the reported PDF does not exist", func() {
			seedWorkerFiles()
			repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
			repo.EXPECT().CompleteFailure(ctx, sub.ID, gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, issues []domain.Issue, _ time.Time) (bool, error) {
					Expect(issues[0].Code).To(Equal("BUILD_NO_OUTPUT"))
					return true, nil
				})
			Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "success", JobID: sub.JobID.String(),
				ExportedPDFKeys: []string{outKey("ghost-main.pdf")}})).To(Succeed())
		})

		It("treats a success without any PDF as failed with BUILD_NO_OUTPUT", func() {
			seedWorkerFiles()
			repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
			repo.EXPECT().CompleteFailure(ctx, sub.ID, gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, issues []domain.Issue, _ time.Time) (bool, error) {
					Expect(issues).To(HaveLen(1))
					Expect(issues[0].Code).To(Equal("BUILD_NO_OUTPUT"))
					Expect(issues[0].Stage).To(Equal(domain.StageBuild))
					return true, nil
				})
			Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "success", JobID: sub.JobID.String()})).To(Succeed())
			Expect(store.Has("uploads/" + sub.ID.String() + ".zip")).To(BeFalse())
		})

		It("requeues (error) when the database commit fails and keeps the output directory", func() {
			seedWorkerFiles("a-main.pdf")
			repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
			repo.EXPECT().CompleteSuccess(ctx, sub.ID, gomock.Any(), gomock.Any()).Return(port.SuccessOutcome{}, errors.New("db down"))
			Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "success", JobID: sub.JobID.String(), ExportedPDFKeys: []string{outKey("a-main.pdf")}})).NotTo(Succeed())
			Expect(store.Has(outKey("a-main.pdf"))).To(BeTrue())
		})

		It("does not clean up when the row was already completed (duplicate delivery)", func() {
			seedWorkerFiles("a-main.pdf")
			repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
			repo.EXPECT().CompleteSuccess(ctx, sub.ID, gomock.Any(), gomock.Any()).Return(port.SuccessOutcome{Applied: false}, nil)
			Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "success", JobID: sub.JobID.String(), ExportedPDFKeys: []string{outKey("a-main.pdf")}})).To(Succeed())
			Expect(store.Has(outKey("a-main.pdf"))).To(BeTrue())
		})
	})

	Describe("failed", func() {
		It("stores the parsed issues and deletes the zip and the output directory", func() {
			seedWorkerFiles("junk.pdf")
			repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
			repo.EXPECT().CompleteFailure(ctx, sub.ID, gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, issues []domain.Issue, _ time.Time) (bool, error) {
					Expect(issues).To(HaveLen(2))
					Expect(issues[0].Title).To(Equal("Row 4: invalid week 'x'. Use integers"))
					Expect(issues[1].Code).To(Equal("BUILD_VALIDATION"))
					return true, nil
				})
			text := "Pipeline processing failed for job j: Validation failed for c.\nErrors:\n  • Row 4: invalid week 'x'. Use integers.\n  • weird\n\n--- Validation Report ---\n"
			Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "failed", JobID: sub.JobID.String(), Error: text})).To(Succeed())
			Expect(store.Keys("")).To(BeEmpty())
		})

		It("requeues when the database fails", func() {
			repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
			repo.EXPECT().CompleteFailure(ctx, sub.ID, gomock.Any(), gomock.Any()).Return(false, errors.New("db down"))
			Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "failed", JobID: sub.JobID.String(), Error: "boom"})).NotTo(Succeed())
		})
	})
})
