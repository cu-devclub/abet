package commands_test

import (
	"context"
	"errors"
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

func offeringIn(curID uuid.UUID) *domain.Offering {
	return &domain.Offering{
		ID: uuid.New(), CourseID: uuid.New(), CourseCode: "100", CourseName: "Calc", AcademicYear: 2026, Semester: 1,
		Curriculum: domain.Ref{ID: curID, Code: "CS", Name: "CS"}, Department: domain.Ref{ID: uuid.New(), Code: "MATH-CS", Name: "Math"},
	}
}

func adminOf(curIDs ...uuid.UUID) *domain.Actor {
	return &domain.Actor{UserID: uuid.New(), Role: domain.RoleAdmin, ScopeCurriculumIDs: curIDs}
}

var _ = Describe("offering commands", func() {
	var (
		ctrl    *gomock.Controller
		repo    *mocks.MockOfferingRepository
		currs   *mocks.MockCurriculumRepository
		storage *mocks.MockStorage
		ctx     = context.Background()
		curID   = uuid.New()
		admin   *domain.Actor
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = mocks.NewMockOfferingRepository(ctrl)
		currs = mocks.NewMockCurriculumRepository(ctrl)
		storage = mocks.NewMockStorage(ctrl)
		admin = adminOf(curID)
	})
	AfterEach(func() { ctrl.Finish() })

	Describe("create", func() {
		in := func() port.CreateOfferingInput {
			return port.CreateOfferingInput{CourseCode: " 100 ", CourseName: " Calc ", CurriculumID: curID, AcademicYear: 2026, Semester: 1}
		}

		It("creates for an admin with the curriculum in scope and trims the input", func() {
			id := uuid.New()
			currs.EXPECT().GetByID(ctx, curID).Return(&domain.Curriculum{ID: curID}, nil)
			repo.EXPECT().Create(ctx, port.CreateOfferingParams{CourseCode: "100", CourseName: "Calc", CurriculumID: curID, AcademicYear: 2026, Semester: 1}).Return(id, nil)
			repo.EXPECT().GetByID(ctx, id).Return(&domain.Offering{ID: id}, nil)
			o, err := commands.NewCreateOfferingHandler(repo, currs).Execute(ctx, admin, in())
			Expect(err).NotTo(HaveOccurred())
			Expect(o.ID).To(Equal(id))
		})

		It("is forbidden for an admin outside the scope, for staff and for instructors", func() {
			h := commands.NewCreateOfferingHandler(repo, currs)
			for _, a := range []*domain.Actor{adminOf(uuid.New()), staffActor, {Role: domain.RoleInstructor}} {
				_, err := h.Execute(ctx, a, in())
				Expect(err).To(MatchError(domain.ErrForbidden))
			}
		})

		It("validates code, name, year and semester", func() {
			h := commands.NewCreateOfferingHandler(repo, currs)
			for _, mut := range []func(*port.CreateOfferingInput){
				func(i *port.CreateOfferingInput) { i.CourseCode = " " },
				func(i *port.CreateOfferingInput) { i.CourseName = "" },
				func(i *port.CreateOfferingInput) { i.AcademicYear = 1999 },
				func(i *port.CreateOfferingInput) { i.AcademicYear = 2201 },
				func(i *port.CreateOfferingInput) { i.Semester = 4 },
			} {
				i := in()
				mut(&i)
				_, err := h.Execute(ctx, superActor, i)
				Expect(err).To(MatchError(domain.ErrInvalidInput))
			}
		})

		It("answers 400 for an unknown curriculum (super admin)", func() {
			currs.EXPECT().GetByID(ctx, curID).Return(nil, domain.ErrNotFound)
			_, err := commands.NewCreateOfferingHandler(repo, currs).Execute(ctx, superActor, in())
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("maps a duplicate offering to a conflict", func() {
			currs.EXPECT().GetByID(ctx, curID).Return(&domain.Curriculum{}, nil)
			repo.EXPECT().Create(ctx, gomock.Any()).Return(uuid.Nil, fmt.Errorf("x: %w", domain.ErrConflict))
			_, err := commands.NewCreateOfferingHandler(repo, currs).Execute(ctx, admin, in())
			Expect(err).To(MatchError(domain.ErrConflict))
			Expect(err.Error()).To(ContainSubstring("offering already exists"))
		})
	})

	Describe("update", func() {
		var cur *domain.Offering
		BeforeEach(func() { cur = offeringIn(curID) })

		It("applies a partial update; a missing name keeps the course name and does not rename", func() {
			code := "200"
			repo.EXPECT().GetByID(ctx, cur.ID).Return(cur, nil)
			repo.EXPECT().Update(ctx, port.UpdateOfferingParams{
				ID: cur.ID, CourseCode: "200", CourseName: "Calc", RenameCourse: false, CurriculumID: curID, AcademicYear: 2026, Semester: 1,
			}).Return(nil)
			repo.EXPECT().GetByID(ctx, cur.ID).Return(cur, nil)
			_, err := commands.NewUpdateOfferingHandler(repo, currs).Execute(ctx, admin, port.UpdateOfferingInput{ID: cur.ID, CourseCode: &code})
			Expect(err).NotTo(HaveOccurred())
		})

		It("renames the course when a name is sent", func() {
			name, year, sem := " Calculus ", 2025, 2
			repo.EXPECT().GetByID(ctx, cur.ID).Return(cur, nil).Times(2)
			repo.EXPECT().Update(ctx, port.UpdateOfferingParams{
				ID: cur.ID, CourseCode: "100", CourseName: "Calculus", RenameCourse: true, CurriculumID: curID, AcademicYear: 2025, Semester: 2,
			}).Return(nil)
			_, err := commands.NewUpdateOfferingHandler(repo, currs).Execute(ctx, admin, port.UpdateOfferingInput{ID: cur.ID, CourseName: &name, AcademicYear: &year, Semester: &sem})
			Expect(err).NotTo(HaveOccurred())
		})

		It("answers 404 for an unknown offering", func() {
			repo.EXPECT().GetByID(ctx, cur.ID).Return(nil, domain.ErrNotFound)
			_, err := commands.NewUpdateOfferingHandler(repo, currs).Execute(ctx, admin, port.UpdateOfferingInput{ID: cur.ID})
			Expect(err).To(MatchError(domain.ErrNotFound))
		})

		It("is forbidden when the current curriculum is out of scope", func() {
			repo.EXPECT().GetByID(ctx, cur.ID).Return(cur, nil)
			_, err := commands.NewUpdateOfferingHandler(repo, currs).Execute(ctx, adminOf(uuid.New()), port.UpdateOfferingInput{ID: cur.ID})
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("requires BOTH the old and the new curriculum in scope when moving", func() {
			target := uuid.New()
			repo.EXPECT().GetByID(ctx, cur.ID).Return(cur, nil)
			_, err := commands.NewUpdateOfferingHandler(repo, currs).Execute(ctx, admin, port.UpdateOfferingInput{ID: cur.ID, CurriculumID: &target})
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("moves between two curricula that are both in scope", func() {
			target := uuid.New()
			repo.EXPECT().GetByID(ctx, cur.ID).Return(cur, nil).Times(2)
			currs.EXPECT().GetByID(ctx, target).Return(&domain.Curriculum{ID: target}, nil)
			repo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, p port.UpdateOfferingParams) error {
				Expect(p.CurriculumID).To(Equal(target))
				return nil
			})
			_, err := commands.NewUpdateOfferingHandler(repo, currs).Execute(ctx, adminOf(curID, target), port.UpdateOfferingInput{ID: cur.ID, CurriculumID: &target})
			Expect(err).NotTo(HaveOccurred())
		})

		It("maps a collision to 409 and rejects invalid values with 400", func() {
			year := 3000
			repo.EXPECT().GetByID(ctx, cur.ID).Return(cur, nil).Times(2)
			_, err := commands.NewUpdateOfferingHandler(repo, currs).Execute(ctx, admin, port.UpdateOfferingInput{ID: cur.ID, AcademicYear: &year})
			Expect(err).To(MatchError(domain.ErrInvalidInput))

			repo.EXPECT().Update(ctx, gomock.Any()).Return(fmt.Errorf("x: %w", domain.ErrConflict))
			_, err = commands.NewUpdateOfferingHandler(repo, currs).Execute(ctx, admin, port.UpdateOfferingInput{ID: cur.ID})
			Expect(err).To(MatchError(domain.ErrConflict))
		})
	})

	Describe("delete (single and bulk)", func() {
		var a, b *domain.Offering
		BeforeEach(func() { a, b = offeringIn(curID), offeringIn(curID) })

		It("deletes in the database first, then removes the submission files best-effort", func() {
			sid1, sid2 := uuid.New(), uuid.New()
			repo.EXPECT().ListByIDs(ctx, []uuid.UUID{a.ID, b.ID}).Return([]domain.Offering{*a, *b}, nil)
			repo.EXPECT().Delete(ctx, []uuid.UUID{a.ID, b.ID}).Return([]uuid.UUID{sid1, sid2}, nil)
			for _, sid := range []uuid.UUID{sid1, sid2} {
				storage.EXPECT().Delete(ctx, "uploads/"+sid.String()+".zip").Return(nil)
				storage.EXPECT().Delete(ctx, "portfolios/"+sid.String()+".pdf").Return(errors.New("disk")) // logged, ignored
				storage.EXPECT().DeletePrefix(ctx, "outputs/"+sid.String()+"/").Return(nil)
			}
			n, err := commands.NewDeleteOfferingsHandler(repo, storage).Execute(ctx, admin, []uuid.UUID{a.ID, b.ID, a.ID})
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(2))
		})

		It("does not touch storage when the transaction fails", func() {
			repo.EXPECT().ListByIDs(ctx, []uuid.UUID{a.ID}).Return([]domain.Offering{*a}, nil)
			repo.EXPECT().Delete(ctx, []uuid.UUID{a.ID}).Return(nil, errors.New("db down"))
			_, err := commands.NewDeleteOfferingsHandler(repo, storage).Execute(ctx, admin, []uuid.UUID{a.ID})
			Expect(err).To(HaveOccurred())
		})

		It("is all-or-nothing: one out-of-scope offering forbids the whole request", func() {
			other := offeringIn(uuid.New())
			repo.EXPECT().ListByIDs(ctx, gomock.Any()).Return([]domain.Offering{*a, *other}, nil)
			_, err := commands.NewDeleteOfferingsHandler(repo, storage).Execute(ctx, admin, []uuid.UUID{a.ID, other.ID})
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("answers 404 when an id does not exist and 400 for an empty list", func() {
			repo.EXPECT().ListByIDs(ctx, gomock.Any()).Return([]domain.Offering{*a}, nil)
			_, err := commands.NewDeleteOfferingsHandler(repo, storage).Execute(ctx, admin, []uuid.UUID{a.ID, uuid.New()})
			Expect(err).To(MatchError(domain.ErrNotFound))

			_, err = commands.NewDeleteOfferingsHandler(repo, storage).Execute(ctx, admin, nil)
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("is forbidden for staff", func() {
			repo.EXPECT().ListByIDs(ctx, gomock.Any()).Return([]domain.Offering{*a}, nil)
			_, err := commands.NewDeleteOfferingsHandler(repo, storage).Execute(ctx, staffActor, []uuid.UUID{a.ID})
			Expect(err).To(MatchError(domain.ErrForbidden))
		})
	})
})
