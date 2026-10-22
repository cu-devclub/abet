package commands_test

import (
	"bytes"
	"context"
	"errors"
	"os"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"abet/main-service/internal/application/commands"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
	"abet/main-service/internal/core/port/mocks"
)

// Exact calls on the Storage port (strict gomock), complementing the in-memory store used by the other specs.
var _ = Describe("submission use cases against a mocked Storage", func() {
	var (
		ctrl    *gomock.Controller
		repo    *mocks.MockSubmissionRepository
		pub     *mocks.MockJobPublisher
		storage *mocks.MockStorage
		ctx     = context.Background()
		tmpDir  string
		actor   = &domain.Actor{UserID: uuid.New(), Email: "s@chula.ac.th", Role: domain.RoleSuperAdmin}
		offID   = uuid.New()
		currID  = uuid.New()
		info    *domain.OfferingInfo
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = mocks.NewMockSubmissionRepository(ctrl)
		pub = mocks.NewMockJobPublisher(ctrl)
		storage = mocks.NewMockStorage(ctrl)
		tmpDir = GinkgoT().TempDir()
		info = &domain.OfferingInfo{ID: offID, CourseCode: "1234567", AcademicYear: 2026, Semester: 1,
			Curriculum: domain.Ref{ID: currID, Code: "CS"}}
		repo.EXPECT().GetOffering(ctx, offID).Return(info, nil).AnyTimes()
	})
	AfterEach(func() { ctrl.Finish() })

	Describe("upload", func() {
		upload := func(body []byte) (*domain.Submission, error) {
			return commands.NewUploadSubmissionHandler(repo, storage, pub, tmpDir).Execute(ctx, actor,
				port.UploadSubmissionInput{OfferingID: offID, Filename: "a.zip", Body: bytes.NewReader(body)})
		}
		tmpEmpty := func() {
			entries, _ := os.ReadDir(tmpDir)
			ExpectWithOffset(1, entries).To(BeEmpty(), "the spooled temp file must be removed")
		}

		It("validates the spooled temp file, then PutFile's it to uploads/<sid>.zip and publishes keys", func() {
			zipBytes := validPackageZip(nil)
			var uploadKey string
			storage.EXPECT().PutFile(ctx, gomock.Any(), gomock.Any(), "application/zip").DoAndReturn(
				func(_ context.Context, key, local, _ string) error {
					uploadKey = key
					Expect(local).To(HavePrefix(tmpDir))
					b, err := os.ReadFile(local)
					Expect(err).NotTo(HaveOccurred())
					Expect(b).To(Equal(zipBytes))
					return nil
				})
			storage.EXPECT().Stat(ctx, "covers/"+currID.String()+".pdf").Return(port.ObjectInfo{Key: "covers/" + currID.String() + ".pdf"}, nil)
			var sub *domain.Submission
			repo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, s *domain.Submission) error { sub = s; return nil })
			pub.EXPECT().Publish(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, j domain.BuildJob) error {
				Expect(j.ZipKey).To(Equal(uploadKey))
				Expect(uploadKey).To(Equal("uploads/" + sub.ID.String() + ".zip"))
				Expect(j.OutputPrefix).To(Equal("outputs/" + sub.ID.String() + "/"))
				Expect(j.CoverKey).To(Equal("covers/" + currID.String() + ".pdf"))
				Expect(j.DeleteZipOnCompletion).To(BeTrue())
				return nil
			})
			repo.EXPECT().GetByID(ctx, gomock.Any()).Return(&domain.Submission{}, nil)
			_, err := upload(zipBytes)
			Expect(err).NotTo(HaveOccurred())
			tmpEmpty()
		})

		It("never touches the bucket for a non-zip or an invalid package", func() {
			_, err := upload([]byte("not a zip"))
			Expect(err).To(MatchError(domain.ErrInvalidInput))
			tmpEmpty()

			bad := validPackageZip(func(f map[string][]byte) {
				for _, k := range []string{"02-textbook/textbook-part01.pdf", "02-textbook/textbook-part02.pdf", "02-textbook/textbook-part03.pdf"} {
					delete(f, k)
				}
			})
			var created *domain.Submission
			repo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, s *domain.Submission) error { created = s; return nil })
			repo.EXPECT().GetByID(ctx, gomock.Any()).DoAndReturn(func(context.Context, uuid.UUID) (*domain.Submission, error) { return created, nil })
			sub, err := upload(bad)
			Expect(err).NotTo(HaveOccurred())
			Expect(sub.Status).To(Equal(domain.SubmissionInvalid))
			tmpEmpty()
		})

		It("returns the storage error without creating a submission row", func() {
			storage.EXPECT().PutFile(ctx, gomock.Any(), gomock.Any(), "application/zip").Return(errors.New("minio down"))
			storage.EXPECT().Delete(ctx, gomock.Any()).Return(nil)
			_, err := upload(validPackageZip(nil))
			Expect(err).To(HaveOccurred())
			tmpEmpty()
		})

		It("deletes the uploaded object and records QUEUE_UNAVAILABLE when publishing fails", func() {
			storage.EXPECT().PutFile(ctx, gomock.Any(), gomock.Any(), "application/zip").Return(nil)
			storage.EXPECT().Stat(ctx, gomock.Any()).Return(port.ObjectInfo{}, domain.ErrNotFound)
			var sub *domain.Submission
			repo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, s *domain.Submission) error { sub = s; return nil })
			pub.EXPECT().Publish(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, j domain.BuildJob) error {
				Expect(j.CoverKey).To(BeEmpty())
				return domain.ErrServiceUnavailable
			})
			storage.EXPECT().Delete(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, key string) error {
				Expect(key).To(Equal("uploads/" + sub.ID.String() + ".zip"))
				return nil
			})
			repo.EXPECT().CompleteFailure(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
			repo.EXPECT().GetByID(ctx, gomock.Any()).Return(&domain.Submission{Status: domain.SubmissionFailed}, nil)
			_, err := upload(validPackageZip(nil))
			Expect(err).NotTo(HaveOccurred())
			tmpEmpty()
		})

		It("publishes without cover_key when the cover check fails for a reason other than missing", func() {
			storage.EXPECT().PutFile(ctx, gomock.Any(), gomock.Any(), "application/zip").Return(nil)
			storage.EXPECT().Stat(ctx, gomock.Any()).Return(port.ObjectInfo{}, errors.New("timeout"))
			repo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
			pub.EXPECT().Publish(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, j domain.BuildJob) error {
				Expect(j.CoverKey).To(BeEmpty())
				return nil
			})
			repo.EXPECT().GetByID(ctx, gomock.Any()).Return(&domain.Submission{}, nil)
			_, err := upload(validPackageZip(nil))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("status consumer", func() {
		var (
			h   *commands.ProcessBuildStatusHandler
			sub *domain.Submission
		)
		BeforeEach(func() {
			h = commands.NewProcessBuildStatusHandler(repo, storage)
			sub = &domain.Submission{ID: uuid.New(), JobID: uuid.New(), Status: domain.SubmissionQueued}
			repo.EXPECT().GetByJobID(ctx, sub.JobID).Return(sub, nil)
		})
		prefix := func() string { return "outputs/" + sub.ID.String() + "/" }

		It("copies the -main.pdf server-side, stats the result, commits, then drops zip, outputs and older PDFs", func() {
			older := uuid.New()
			dst := "portfolios/" + sub.ID.String() + ".pdf"
			gomock.InOrder(
				storage.EXPECT().Copy(ctx, prefix()+"1-main.pdf", dst).Return(nil),
				storage.EXPECT().Stat(ctx, dst).Return(port.ObjectInfo{Key: dst, Size: 4242}, nil),
				repo.EXPECT().CompleteSuccess(ctx, sub.ID, int64(4242), gomock.Any()).
					Return(port.SuccessOutcome{Applied: true, OlderIDs: []uuid.UUID{older}}, nil),
				storage.EXPECT().Delete(ctx, "uploads/"+sub.ID.String()+".zip").Return(nil),
				storage.EXPECT().DeletePrefix(ctx, prefix()).Return(nil),
				storage.EXPECT().Delete(ctx, "portfolios/"+older.String()+".pdf").Return(nil),
			)
			Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "success", JobID: sub.JobID.String(),
				ExportedPDFKeys: []string{prefix() + "report.pdf", prefix() + "1-main.pdf"}})).To(Succeed())
		})

		It("requeues (error) and commits nothing when the copy fails", func() {
			storage.EXPECT().Copy(ctx, gomock.Any(), gomock.Any()).Return(errors.New("minio down"))
			Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "success", JobID: sub.JobID.String(),
				ExportedPDFKeys: []string{prefix() + "a-main.pdf"}})).NotTo(Succeed())
		})

		It("failed: removes the uploaded zip and the outputs prefix, ignoring storage errors", func() {
			repo.EXPECT().CompleteFailure(ctx, sub.ID, gomock.Any(), gomock.Any()).Return(true, nil)
			storage.EXPECT().Delete(ctx, "uploads/"+sub.ID.String()+".zip").Return(errors.New("minio down"))
			storage.EXPECT().DeletePrefix(ctx, prefix()).Return(errors.New("minio down"))
			Expect(h.Execute(ctx, domain.BuildStatusEvent{Status: "failed", JobID: sub.JobID.String(), Error: "boom"})).To(Succeed())
		})
	})
})
