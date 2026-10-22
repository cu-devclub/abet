package commands_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"abet/main-service/internal/application/commands"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
	"abet/main-service/internal/core/port/mocks"
)

var _ = Describe("cover commands", func() {
	const maxBytes = 1 << 20
	var (
		ctrl    *gomock.Controller
		currs   *mocks.MockCurriculumRepository
		covers  *mocks.MockCoverRepository
		storage *mocks.MockStorage
		ctx     = context.Background()
		curID   = uuid.New()
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		currs = mocks.NewMockCurriculumRepository(ctrl)
		covers = mocks.NewMockCoverRepository(ctrl)
		storage = mocks.NewMockStorage(ctrl)
	})
	AfterEach(func() { ctrl.Finish() })

	pdf := func(extra int) []byte { return append([]byte("%PDF-1.7\n"), make([]byte, extra)...) }

	Describe("put", func() {
		h := func() *commands.PutCoverHandler { return commands.NewPutCoverHandler(currs, covers, storage, maxBytes) }

		It("stores the PDF at covers/<id>.pdf, records the metadata and returns the curriculum", func() {
			data := pdf(100)
			currs.EXPECT().GetByID(ctx, curID).Return(&domain.Curriculum{ID: curID}, nil).Times(2)
			storage.EXPECT().Put(ctx, "covers/"+curID.String()+".pdf", gomock.Any(), int64(len(data)), "application/pdf").DoAndReturn(
				func(_ context.Context, _ string, r io.Reader, _ int64, _ string) error {
					b, _ := io.ReadAll(r)
					Expect(b).To(Equal(data))
					return nil
				})
			covers.EXPECT().Set(ctx, curID, "My Cover.pdf", int64(len(data)), gomock.Any()).Return(nil)
			_, err := h().Execute(ctx, adminOf(curID), port.PutCoverInput{CurriculumID: curID, Filename: `C:\fakepath\My Cover.pdf`, File: bytes.NewReader(data)})
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects a file without the %PDF- magic with 400 and stores nothing", func() {
			currs.EXPECT().GetByID(ctx, curID).Return(&domain.Curriculum{}, nil)
			_, err := h().Execute(ctx, superActor, port.PutCoverInput{CurriculumID: curID, Filename: "x.pdf", File: bytes.NewReader([]byte("<html>not a pdf</html>"))})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("rejects a file whose magic is not at offset 0", func() {
			currs.EXPECT().GetByID(ctx, curID).Return(&domain.Curriculum{}, nil)
			_, err := h().Execute(ctx, superActor, port.PutCoverInput{CurriculumID: curID, File: bytes.NewReader(append([]byte("\n"), pdf(10)...))})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("rejects an empty file with 400", func() {
			currs.EXPECT().GetByID(ctx, curID).Return(&domain.Curriculum{}, nil)
			_, err := h().Execute(ctx, superActor, port.PutCoverInput{CurriculumID: curID, File: bytes.NewReader(nil)})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("rejects a file over the limit with 413 and stores nothing; exactly at the limit passes", func() {
			currs.EXPECT().GetByID(ctx, curID).Return(&domain.Curriculum{}, nil)
			_, err := h().Execute(ctx, superActor, port.PutCoverInput{CurriculumID: curID, File: bytes.NewReader(pdf(maxBytes))})
			Expect(err).To(MatchError(domain.ErrTooLarge))

			exact := pdf(maxBytes - 9)
			Expect(exact).To(HaveLen(maxBytes))
			currs.EXPECT().GetByID(ctx, curID).Return(&domain.Curriculum{}, nil).Times(2)
			storage.EXPECT().Put(ctx, gomock.Any(), gomock.Any(), int64(maxBytes), "application/pdf").Return(nil)
			covers.EXPECT().Set(ctx, curID, "cover.pdf", int64(maxBytes), gomock.Any()).Return(nil)
			_, err = h().Execute(ctx, superActor, port.PutCoverInput{CurriculumID: curID, File: bytes.NewReader(exact)})
			Expect(err).NotTo(HaveOccurred())
		})

		It("is forbidden for an admin out of scope and for staff; 404 for an unknown curriculum", func() {
			_, err := h().Execute(ctx, adminOf(uuid.New()), port.PutCoverInput{CurriculumID: curID, File: bytes.NewReader(pdf(1))})
			Expect(err).To(MatchError(domain.ErrForbidden))
			_, err = h().Execute(ctx, staffActor, port.PutCoverInput{CurriculumID: curID, File: bytes.NewReader(pdf(1))})
			Expect(err).To(MatchError(domain.ErrForbidden))

			currs.EXPECT().GetByID(ctx, curID).Return(nil, domain.ErrNotFound)
			_, err = h().Execute(ctx, superActor, port.PutCoverInput{CurriculumID: curID, File: bytes.NewReader(pdf(1))})
			Expect(err).To(MatchError(domain.ErrNotFound))
		})

		It("sanitizes filenames", func() {
			Expect(commands.SanitizeCoverFilename("../../etc/passwd")).To(Equal("passwd"))
			Expect(commands.SanitizeCoverFilename(`a"b;c.pdf`)).To(Equal("abc.pdf"))
			Expect(commands.SanitizeCoverFilename("  ")).To(Equal("cover.pdf"))
			Expect(commands.SanitizeCoverFilename("ปก.pdf")).To(Equal("ปก.pdf"))
		})
	})

	Describe("get", func() {
		It("opens the stored file when the curriculum has a cover", func() {
			cover := &domain.Cover{Filename: "c.pdf", SizeBytes: 5}
			currs.EXPECT().GetByID(ctx, curID).Return(&domain.Curriculum{ID: curID, Cover: cover}, nil)
			info := port.ObjectInfo{Key: "covers/" + curID.String() + ".pdf", Size: 5}
			storage.EXPECT().Get(ctx, "covers/"+curID.String()+".pdf").Return(io.NopCloser(strings.NewReader("%PDF-")), info, nil)
			cf, err := commands.NewGetCoverHandler(currs, storage).Execute(ctx, adminOf(curID), curID)
			Expect(err).NotTo(HaveOccurred())
			Expect(cf.Info).To(Equal(info))
			Expect(cf.Cover.Filename).To(Equal("c.pdf"))
		})

		It("answers 404 without a cover and when the file is missing", func() {
			currs.EXPECT().GetByID(ctx, curID).Return(&domain.Curriculum{ID: curID}, nil)
			_, err := commands.NewGetCoverHandler(currs, storage).Execute(ctx, superActor, curID)
			Expect(err).To(MatchError(domain.ErrNotFound))

			currs.EXPECT().GetByID(ctx, curID).Return(&domain.Curriculum{ID: curID, Cover: &domain.Cover{}}, nil)
			storage.EXPECT().Get(ctx, gomock.Any()).Return(nil, port.ObjectInfo{}, domain.ErrNotFound)
			_, err = commands.NewGetCoverHandler(currs, storage).Execute(ctx, superActor, curID)
			Expect(err).To(MatchError(domain.ErrNotFound))
		})

		It("is forbidden out of scope", func() {
			_, err := commands.NewGetCoverHandler(currs, storage).Execute(ctx, adminOf(uuid.New()), curID)
			Expect(err).To(MatchError(domain.ErrForbidden))
		})
	})

	Describe("delete", func() {
		It("clears the metadata and removes the file (best effort)", func() {
			currs.EXPECT().GetByID(ctx, curID).Return(&domain.Curriculum{ID: curID}, nil)
			covers.EXPECT().Clear(ctx, curID, gomock.Any()).Return(nil)
			storage.EXPECT().Delete(ctx, "covers/"+curID.String()+".pdf").Return(errors.New("disk"))
			Expect(commands.NewDeleteCoverHandler(currs, covers, storage).Execute(ctx, adminOf(curID), curID)).To(Succeed())
		})

		It("is forbidden out of scope and 404 for an unknown curriculum", func() {
			h := commands.NewDeleteCoverHandler(currs, covers, storage)
			Expect(h.Execute(ctx, adminOf(uuid.New()), curID)).To(MatchError(domain.ErrForbidden))
			currs.EXPECT().GetByID(ctx, curID).Return(nil, domain.ErrNotFound)
			Expect(h.Execute(ctx, superActor, curID)).To(MatchError(domain.ErrNotFound))
		})
	})
})
