package commands_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"abet/main-service/internal/application/commands"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
	"abet/main-service/internal/core/port/mocks"
)

const importHeader = "course id,course name,semester,user email,user role,upload permission,download permission\n"

var _ = Describe("import commands", func() {
	var (
		ctrl  *gomock.Controller
		repo  *mocks.MockImportRepository
		currs *mocks.MockCurriculumRepository
		ctx   = context.Background()
		curID = uuid.New()
		cur   *domain.Curriculum
		allow = []string{"chula.ac.th"}
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = mocks.NewMockImportRepository(ctrl)
		currs = mocks.NewMockCurriculumRepository(ctrl)
		cur = &domain.Curriculum{ID: curID, DepartmentID: uuid.New()}
	})
	AfterEach(func() { ctrl.Finish() })

	in := func(csv string) port.ImportInput {
		return port.ImportInput{CurriculumID: curID, AcademicYear: 2026, Data: []byte(csv)}
	}

	Describe("size limit (MAX_IMPORT_MB)", func() {
		It("uses the configured limit and names it in the message", func() {
			h := commands.NewPreviewImportHandler(repo, currs, allow, 2<<20)
			i := in(importHeader)
			i.Data = make([]byte, 2<<20+1)
			_, err := h.Execute(ctx, superActor, i)
			Expect(err).To(MatchError(domain.ErrTooLarge))
			Expect(err.Error()).To(ContainSubstring("2 MB"))
		})

		It("accepts a file between the old 5 MB and a raised limit", func() {
			currs.EXPECT().GetByID(ctx, curID).Return(cur, nil)
			repo.EXPECT().LoadState(ctx, curID, 2026, gomock.Any()).Return(&domain.ImportState{}, nil)
			h := commands.NewPreviewImportHandler(repo, currs, allow, 10<<20)
			i := in(importHeader)
			i.Data = append(i.Data, bytes.Repeat([]byte("\n"), 6<<20)...)
			_, err := h.Execute(ctx, superActor, i)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("preview", func() {
		It("parses, loads the state for the CSV emails and returns the plan with parse errors", func() {
			currs.EXPECT().GetByID(ctx, curID).Return(cur, nil)
			repo.EXPECT().LoadState(ctx, curID, 2026, []string{"a@chula.ac.th"}).Return(&domain.ImportState{}, nil)
			csv := importHeader + "100,Calc,1,a@chula.ac.th,Staff,No,Yes\n" + "200,Bad,9,,,,\n"
			plan, err := commands.NewPreviewImportHandler(repo, currs, allow, port.DefaultMaxImportBytes).Execute(ctx, adminOf(curID), in(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(plan.Applied).To(BeFalse())
			Expect(plan.Summary.Rows).To(Equal(2), "rows counts invalid rows too")
			Expect(plan.Summary.CoursesCreate).To(Equal(1))
			Expect(plan.Summary.UsersCreate).To(Equal(1))
			Expect(plan.Errors).To(Equal([]domain.ImportError{{Row: 3, Message: "semester must be 1, 2 or 3"}}))
		})

		It("is forbidden outside the scope and for non admins (before anything is read)", func() {
			h := commands.NewPreviewImportHandler(repo, currs, allow, port.DefaultMaxImportBytes)
			_, err := h.Execute(ctx, adminOf(uuid.New()), in(importHeader))
			Expect(err).To(MatchError(domain.ErrForbidden))
			_, err = h.Execute(ctx, staffActor, in(importHeader))
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("validates the academic year (400) and the size (413)", func() {
			h := commands.NewPreviewImportHandler(repo, currs, allow, port.DefaultMaxImportBytes)
			i := in(importHeader)
			i.AcademicYear = 1999
			_, err := h.Execute(ctx, superActor, i)
			Expect(err).To(MatchError(domain.ErrInvalidInput))

			i = port.ImportInput{CurriculumID: curID, AcademicYear: 2026, Data: bytes.Repeat([]byte("x"), port.DefaultMaxImportBytes+1)}
			_, err = h.Execute(ctx, superActor, i)
			Expect(err).To(MatchError(domain.ErrTooLarge))
		})

		It("answers 404 for an unknown curriculum (super admin)", func() {
			currs.EXPECT().GetByID(ctx, curID).Return(nil, domain.ErrNotFound)
			_, err := commands.NewPreviewImportHandler(repo, currs, allow, port.DefaultMaxImportBytes).Execute(ctx, superActor, in(importHeader))
			Expect(err).To(MatchError(domain.ErrNotFound))
		})

		It("reports an empty (header-only) file as an error instead of planning to delete everything", func() {
			currs.EXPECT().GetByID(ctx, curID).Return(cur, nil)
			repo.EXPECT().LoadState(ctx, curID, 2026, gomock.Any()).Return(&domain.ImportState{}, nil)
			plan, err := commands.NewPreviewImportHandler(repo, currs, allow, port.DefaultMaxImportBytes).Execute(ctx, superActor, in(importHeader))
			Expect(err).NotTo(HaveOccurred())
			Expect(plan.Errors).To(HaveLen(1))
		})
	})

	Describe("apply", func() {
		It("recomputes the plan inside the repository transaction", func() {
			currs.EXPECT().GetByID(ctx, curID).Return(cur, nil)
			repo.EXPECT().Apply(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, p port.ApplyImportParams) (*domain.ImportPlan, error) {
				Expect(p.Curriculum).To(BeIdenticalTo(cur))
				Expect(p.AcademicYear).To(Equal(2026))
				Expect(p.Emails).To(Equal([]string{"a@chula.ac.th"}))
				plan := p.Build(&domain.ImportState{})
				Expect(plan.Summary.CoursesCreate).To(Equal(1))
				plan.Applied = true
				return &plan, nil
			})
			plan, err := commands.NewApplyImportHandler(repo, currs, allow, port.DefaultMaxImportBytes).Execute(ctx, adminOf(curID), in(importHeader+"100,Calc,1,a@chula.ac.th,Staff,No,Yes\n"))
			Expect(err).NotTo(HaveOccurred())
			Expect(plan.Applied).To(BeTrue())
		})

		It("returns the plan together with ErrUnprocessable when the CSV has errors", func() {
			currs.EXPECT().GetByID(ctx, curID).Return(cur, nil)
			repo.EXPECT().Apply(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, p port.ApplyImportParams) (*domain.ImportPlan, error) {
				plan := p.Build(&domain.ImportState{})
				Expect(plan.Errors).NotTo(BeEmpty())
				return &plan, fmt.Errorf("%w: the CSV has errors", domain.ErrUnprocessable)
			})
			plan, err := commands.NewApplyImportHandler(repo, currs, allow, port.DefaultMaxImportBytes).Execute(ctx, superActor, in(importHeader+"100,Calc,7,,,,\n"))
			Expect(err).To(MatchError(domain.ErrUnprocessable))
			Expect(plan).NotTo(BeNil())
			Expect(plan.Errors[0].Row).To(Equal(2))
		})

		It("checks authorization and size before touching the repository", func() {
			h := commands.NewApplyImportHandler(repo, currs, allow, port.DefaultMaxImportBytes)
			_, err := h.Execute(ctx, adminOf(uuid.New()), in(importHeader))
			Expect(err).To(MatchError(domain.ErrForbidden))
			_, err = h.Execute(ctx, superActor, port.ImportInput{CurriculumID: curID, AcademicYear: 2026, Data: make([]byte, port.DefaultMaxImportBytes+1)})
			Expect(err).To(MatchError(domain.ErrTooLarge))
		})
	})
})
