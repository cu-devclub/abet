package queries_test

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"abet/main-service/internal/application/queries"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
	"abet/main-service/internal/core/port/mocks"
)

var _ = Describe("offering queries", func() {
	var (
		ctrl  *gomock.Controller
		ctx   = context.Background()
		curID = uuid.New()
		repo  *mocks.MockOfferingRepository
		super = &domain.Actor{UserID: uuid.New(), Role: domain.RoleSuperAdmin}
		admin = &domain.Actor{UserID: uuid.New(), Role: domain.RoleAdmin, ScopeCurriculumIDs: []uuid.UUID{curID}}
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = mocks.NewMockOfferingRepository(ctrl)
	})
	AfterEach(func() { ctrl.Finish() })

	Describe("ListOfferings", func() {
		It("does not restrict a super admin", func() {
			repo.EXPECT().List(ctx, port.OfferingFilter{AcademicYear: 2026, Search: "calc"}).Return([]domain.Offering{{CourseCode: "1"}}, nil)
			items, err := queries.NewListOfferingsHandler(repo).Execute(ctx, super, port.ListOfferingsInput{AcademicYear: 2026, Search: "calc"})
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
		})

		It("restricts an admin to the curricula in scope", func() {
			sem := 2
			repo.EXPECT().List(ctx, port.OfferingFilter{AcademicYear: 2026, Semester: &sem, Restrict: true, CurriculumScope: []uuid.UUID{curID}}).Return(nil, nil)
			_, err := queries.NewListOfferingsHandler(repo).Execute(ctx, admin, port.ListOfferingsInput{AcademicYear: 2026, Semester: &sem})
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects staff and instructors with 403 and bad years or semesters with 400", func() {
			h := queries.NewListOfferingsHandler(repo)
			for _, r := range []domain.Role{domain.RoleStaff, domain.RoleInstructor} {
				_, err := h.Execute(ctx, &domain.Actor{Role: r}, port.ListOfferingsInput{AcademicYear: 2026})
				Expect(err).To(MatchError(domain.ErrForbidden))
			}
			_, err := h.Execute(ctx, super, port.ListOfferingsInput{AcademicYear: 0})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
			bad := 4
			_, err = h.Execute(ctx, super, port.ListOfferingsInput{AcademicYear: 2026, Semester: &bad})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})
	})

	Describe("ListOfferingYears", func() {
		It("returns the distinct years plus the current year, newest first", func() {
			current := time.Now().UTC().Year()
			repo.EXPECT().Years(ctx).Return([]int{current - 1, current - 3}, nil)
			years, cur, err := queries.NewListOfferingYearsHandler(repo).Execute(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(cur).To(Equal(current))
			Expect(years).To(Equal([]int{current, current - 1, current - 3}))
		})

		It("does not duplicate the current year", func() {
			current := time.Now().UTC().Year()
			repo.EXPECT().Years(ctx).Return([]int{current + 1, current}, nil)
			years, _, err := queries.NewListOfferingYearsHandler(repo).Execute(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(years).To(Equal([]int{current + 1, current}))
		})

		It("propagates repository errors", func() {
			repo.EXPECT().Years(ctx).Return(nil, errors.New("db"))
			_, _, err := queries.NewListOfferingYearsHandler(repo).Execute(ctx)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ListAccess", func() {
		var (
			access *mocks.MockAccessRepository
			off    *domain.Offering
		)
		BeforeEach(func() {
			access = mocks.NewMockAccessRepository(ctrl)
			off = &domain.Offering{ID: uuid.New(), Curriculum: domain.Ref{ID: curID}, Department: domain.Ref{ID: uuid.New(), Code: "MATH-CS"}}
		})

		It("lists explicit rows without admins, then the default staff entries", func() {
			offeringsOK := func() { repo.EXPECT().GetByID(ctx, off.ID).Return(off, nil) }
			offeringsOK()
			inst := domain.AccessEntry{UserID: uuid.New(), Email: "i@chula.ac.th", Role: domain.RoleInstructor, CanUpload: true, CanDownload: true, Source: "csv"}
			adm := domain.AccessEntry{UserID: uuid.New(), Email: "a@chula.ac.th", Role: domain.RoleAdmin, CanDownload: true, Source: "csv"}
			sup := domain.AccessEntry{UserID: uuid.New(), Email: "s@chula.ac.th", Role: domain.RoleSuperAdmin, CanDownload: true, Source: "manual"}
			access.EXPECT().List(ctx, off.ID).Return([]domain.AccessEntry{inst, adm, sup}, nil)
			staffID := uuid.New()
			access.EXPECT().ListDefaultStaff(ctx, off.ID, off.Department.ID).Return([]port.DefaultStaff{{UserID: staffID, Name: "Sam", Email: "sam@chula.ac.th"}}, nil)

			items, err := queries.NewListAccessHandler(repo, access).Execute(ctx, admin, off.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(2))
			Expect(items[0].Email).To(Equal("i@chula.ac.th"))
			def := items[1]
			Expect(def.UserID).To(Equal(staffID))
			Expect(def.IsDefault).To(BeTrue())
			Expect(def.Source).To(Equal("default"))
			Expect(def.Role).To(Equal(domain.RoleStaff))
			Expect(def.CanDownload).To(BeTrue())
			Expect(def.CanUpload).To(BeFalse())
			Expect(*def.DefaultReason).To(Equal("Staff in department MATH-CS (default download)"))
		})

		It("is forbidden for admins out of scope and staff; 404 for an unknown offering", func() {
			repo.EXPECT().GetByID(ctx, off.ID).Return(off, nil).Times(2)
			h := queries.NewListAccessHandler(repo, access)
			_, err := h.Execute(ctx, &domain.Actor{Role: domain.RoleAdmin, ScopeCurriculumIDs: []uuid.UUID{uuid.New()}}, off.ID)
			Expect(err).To(MatchError(domain.ErrForbidden))
			_, err = h.Execute(ctx, &domain.Actor{Role: domain.RoleStaff}, off.ID)
			Expect(err).To(MatchError(domain.ErrForbidden))

			repo.EXPECT().GetByID(ctx, off.ID).Return(nil, domain.ErrNotFound)
			_, err = h.Execute(ctx, super, off.ID)
			Expect(err).To(MatchError(domain.ErrNotFound))
		})
	})
})
