package queries_test

import (
	"context"
	"errors"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"abet/main-service/internal/application/queries"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
	"abet/main-service/internal/core/port/mocks"
)

var _ = Describe("queries", func() {
	var (
		ctrl *gomock.Controller
		ctx  = context.Background()

		deptA = uuid.New()
		deptB = uuid.New()
		currA = uuid.New()

		super = &domain.Actor{UserID: uuid.New(), Role: domain.RoleSuperAdmin}
		admin = &domain.Actor{
			UserID: uuid.New(), Role: domain.RoleAdmin,
			ScopeDepartmentIDs: []uuid.UUID{deptA}, ScopeCurriculumIDs: []uuid.UUID{currA},
			ScopeCurriculumDepartmentIDs: []uuid.UUID{deptA},
		}
		staff      = &domain.Actor{UserID: uuid.New(), Role: domain.RoleStaff}
		instructor = &domain.Actor{UserID: uuid.New(), Role: domain.RoleInstructor}
	)
	BeforeEach(func() { ctrl = gomock.NewController(GinkgoT()) })
	AfterEach(func() { ctrl.Finish() })

	Describe("ListUsers", func() {
		var repo *mocks.MockUserRepository
		inScope := domain.User{ID: uuid.New(), Email: "in@chula.ac.th", Role: domain.RoleInstructor, DepartmentID: &deptA}
		outScope := domain.User{ID: uuid.New(), Email: "out@chula.ac.th", Role: domain.RoleStaff, DepartmentID: &deptB}
		protected := domain.User{ID: uuid.New(), Email: "adm@chula.ac.th", Role: domain.RoleAdmin, DepartmentID: &deptA}
		BeforeEach(func() { repo = mocks.NewMockUserRepository(ctrl) })

		It("returns everyone to a super_admin and forwards the filters", func() {
			role := domain.RoleStaff
			repo.EXPECT().List(ctx, port.UserFilter{Search: "al", Role: &role, DepartmentID: &deptB}).
				Return([]domain.User{inScope, outScope}, nil)
			got, err := queries.NewListUsersHandler(repo).Execute(ctx, super, port.ListUsersInput{Search: "al", Role: &role, DepartmentID: &deptB})
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(HaveLen(2))
		})

		It("hides out-of-scope users from admins but keeps in-scope admins (read-only rows)", func() {
			repo.EXPECT().List(ctx, gomock.Any()).Return([]domain.User{inScope, outScope, protected}, nil)
			got, err := queries.NewListUsersHandler(repo).Execute(ctx, admin, port.ListUsersInput{})
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(HaveLen(2))
			Expect(got[0].Email).To(Equal("in@chula.ac.th"))
			Expect(got[1].Email).To(Equal("adm@chula.ac.th"))
		})

		It("is forbidden for staff and instructors", func() {
			for _, a := range []*domain.Actor{staff, instructor} {
				_, err := queries.NewListUsersHandler(repo).Execute(ctx, a, port.ListUsersInput{})
				Expect(err).To(MatchError(domain.ErrForbidden))
			}
		})

		It("rejects an unknown role filter", func() {
			bad := domain.Role("boss")
			_, err := queries.NewListUsersHandler(repo).Execute(ctx, super, port.ListUsersInput{Role: &bad})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("passes repository errors through", func() {
			boom := errors.New("db")
			repo.EXPECT().List(ctx, gomock.Any()).Return(nil, boom)
			_, err := queries.NewListUsersHandler(repo).Execute(ctx, super, port.ListUsersInput{})
			Expect(err).To(MatchError(boom))
		})
	})

	Describe("GetMe", func() {
		var repo *mocks.MockUserRepository
		BeforeEach(func() { repo = mocks.NewMockUserRepository(ctrl) })

		DescribeTable("capabilities per role",
			func(a *domain.Actor, org, users, courses, covers bool) {
				repo.EXPECT().GetByID(ctx, a.UserID).Return(&domain.User{ID: a.UserID, Role: a.Role}, nil)
				res, err := queries.NewGetMeHandler(repo).Execute(ctx, a)
				Expect(err).NotTo(HaveOccurred())
				Expect(res.ManageOrg).To(Equal(org))
				Expect(res.ManageUsers).To(Equal(users))
				Expect(res.ManageCourses).To(Equal(courses))
				Expect(res.ManageCovers).To(Equal(covers))
			},
			Entry("super_admin", super, true, true, true, true),
			Entry("admin", admin, false, true, true, true),
			Entry("instructor", instructor, false, false, false, false),
			Entry("staff", staff, false, false, false, false),
		)

		It("passes a missing user through as not found", func() {
			repo.EXPECT().GetByID(ctx, staff.UserID).Return(nil, domain.ErrNotFound)
			_, err := queries.NewGetMeHandler(repo).Execute(ctx, staff)
			Expect(err).To(MatchError(domain.ErrNotFound))
		})
	})

	Describe("organisation lists (H1)", func() {
		deptC := uuid.New()
		currB := uuid.New()
		depts := []domain.Department{{ID: deptA, Code: "A"}, {ID: deptB, Code: "B"}, {ID: deptC, Code: "C"}}
		currs := []domain.Curriculum{
			{ID: currA, Code: "CA", DepartmentID: deptA}, {ID: currB, Code: "CB", DepartmentID: deptA},
			{ID: uuid.New(), Code: "CC", DepartmentID: deptB},
		}
		codes := func(items []domain.Department) []string {
			out := []string{}
			for _, d := range items {
				out = append(out, d.Code)
			}
			return out
		}
		ccodes := func(items []domain.Curriculum) []string {
			out := []string{}
			for _, c := range items {
				out = append(out, c.Code)
			}
			return out
		}

		Describe("departments", func() {
			var repo *mocks.MockDepartmentRepository
			BeforeEach(func() { repo = mocks.NewMockDepartmentRepository(ctrl) })

			It("returns all departments to super_admin and admin", func() {
				repo.EXPECT().List(ctx).Return(depts, nil).Times(2)
				for _, a := range []*domain.Actor{super, admin} {
					got, err := queries.NewListDepartmentsHandler(repo).Execute(ctx, a)
					Expect(err).NotTo(HaveOccurred())
					Expect(codes(got)).To(Equal([]string{"A", "B", "C"}))
				}
			})

			It("returns only the own department to instructor and staff", func() {
				repo.EXPECT().List(ctx).Return(depts, nil).Times(2)
				for _, role := range []domain.Role{domain.RoleInstructor, domain.RoleStaff} {
					a := &domain.Actor{UserID: uuid.New(), Role: role, DepartmentID: &deptB}
					got, err := queries.NewListDepartmentsHandler(repo).Execute(ctx, a)
					Expect(err).NotTo(HaveOccurred())
					Expect(codes(got)).To(Equal([]string{"B"}))
				}
			})

			It("returns an empty list (not null) when instructor/staff have no department, without reading", func() {
				for _, a := range []*domain.Actor{instructor, staff} {
					got, err := queries.NewListDepartmentsHandler(repo).Execute(ctx, a)
					Expect(err).NotTo(HaveOccurred())
					Expect(got).NotTo(BeNil())
					Expect(got).To(BeEmpty())
				}
			})

			It("passes repository errors through", func() {
				boom := errors.New("db")
				repo.EXPECT().List(ctx).Return(nil, boom)
				_, err := queries.NewListDepartmentsHandler(repo).Execute(ctx, super)
				Expect(err).To(MatchError(boom))
			})
		})

		Describe("curricula", func() {
			var repo *mocks.MockCurriculumRepository
			BeforeEach(func() { repo = mocks.NewMockCurriculumRepository(ctrl) })

			It("returns all curricula to super_admin and admin, forwarding the department filter", func() {
				repo.EXPECT().List(ctx, &deptA).Return(currs[:2], nil).Times(2)
				for _, a := range []*domain.Actor{super, admin} {
					got, err := queries.NewListCurriculaHandler(repo).Execute(ctx, a, &deptA)
					Expect(err).NotTo(HaveOccurred())
					Expect(ccodes(got)).To(Equal([]string{"CA", "CB"}))
				}
			})

			It("returns only the own curriculum when the user has one", func() {
				repo.EXPECT().List(ctx, &deptA).Return(currs[:2], nil)
				a := &domain.Actor{UserID: uuid.New(), Role: domain.RoleInstructor, DepartmentID: &deptA, CurriculumID: &currB}
				got, err := queries.NewListCurriculaHandler(repo).Execute(ctx, a, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(ccodes(got)).To(Equal([]string{"CB"}))
			})

			It("returns the curricula of the own department when the user has no curriculum", func() {
				repo.EXPECT().List(ctx, &deptA).Return(currs[:2], nil)
				a := &domain.Actor{UserID: uuid.New(), Role: domain.RoleStaff, DepartmentID: &deptA}
				got, err := queries.NewListCurriculaHandler(repo).Execute(ctx, a, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(ccodes(got)).To(Equal([]string{"CA", "CB"}))
			})

			It("does not let the department filter widen the scope beyond the own department", func() {
				repo.EXPECT().List(ctx, &deptA).Return(currs[:2], nil)
				a := &domain.Actor{UserID: uuid.New(), Role: domain.RoleStaff, DepartmentID: &deptA}
				got, err := queries.NewListCurriculaHandler(repo).Execute(ctx, a, &deptB)
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(BeEmpty())
			})

			It("returns an empty list when instructor/staff have neither curriculum nor department", func() {
				for _, a := range []*domain.Actor{instructor, staff} {
					got, err := queries.NewListCurriculaHandler(repo).Execute(ctx, a, nil)
					Expect(err).NotTo(HaveOccurred())
					Expect(got).NotTo(BeNil())
					Expect(got).To(BeEmpty())
				}
			})

			It("passes repository errors through", func() {
				boom := errors.New("db")
				repo.EXPECT().List(ctx, nil).Return(nil, boom)
				_, err := queries.NewListCurriculaHandler(repo).Execute(ctx, super, nil)
				Expect(err).To(MatchError(boom))
			})
		})
	})
})
