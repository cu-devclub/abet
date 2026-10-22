package commands_test

import (
	"context"
	"errors"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"abet/main-service/internal/application/commands"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
	"abet/main-service/internal/core/port/mocks"
)

var _ = Describe("user commands", func() {
	var (
		ctrl    *gomock.Controller
		users   *mocks.MockUserRepository
		depts   *mocks.MockDepartmentRepository
		currs   *mocks.MockCurriculumRepository
		ctx     = context.Background()
		allowed = []string{"chula.ac.th"}

		deptA = uuid.New()
		deptB = uuid.New()
		currA = uuid.New() // in deptA
		currB = uuid.New() // in deptB

		// admin managing deptA (curriculum A)
		admin = &domain.Actor{
			UserID: uuid.New(), Email: "admin@chula.ac.th", Role: domain.RoleAdmin,
			ScopeDepartmentIDs: []uuid.UUID{deptA}, ScopeCurriculumIDs: []uuid.UUID{currA},
			ScopeCurriculumDepartmentIDs: []uuid.UUID{deptA},
		}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		users = mocks.NewMockUserRepository(ctrl)
		depts = mocks.NewMockDepartmentRepository(ctrl)
		currs = mocks.NewMockCurriculumRepository(ctrl)
	})
	AfterEach(func() { ctrl.Finish() })

	expectDept := func(id uuid.UUID) {
		depts.EXPECT().GetByID(ctx, id).Return(&domain.Department{ID: id, Code: "D", Name: "D"}, nil).AnyTimes()
	}
	expectCurr := func(id, dept uuid.UUID) {
		currs.EXPECT().GetByID(ctx, id).Return(&domain.Curriculum{ID: id, Code: "C", Name: "C", DepartmentID: dept}, nil).AnyTimes()
	}

	Describe("create", func() {
		var h *commands.CreateUserHandler
		BeforeEach(func() { h = commands.NewCreateUserHandler(users, depts, currs, allowed) })

		stored := func() {
			users.EXPECT().GetByID(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID) (*domain.User, error) {
				return &domain.User{ID: id}, nil
			})
		}

		It("super_admin creates an instructor, lower-casing the e-mail and auto-filling the department", func() {
			expectCurr(currA, deptA)
			expectDept(deptA)
			users.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
				Expect(u.Email).To(Equal("new@chula.ac.th"))
				Expect(u.Role).To(Equal(domain.RoleInstructor))
				Expect(*u.DepartmentID).To(Equal(deptA))
				Expect(*u.CurriculumID).To(Equal(currA))
				Expect(u.AdminScope).To(BeNil())
				return nil
			})
			stored()
			_, err := h.Execute(ctx, superActor, port.CreateUserInput{Email: " New@Chula.ac.th", Role: domain.RoleInstructor, CurriculumID: &currA})
			Expect(err).NotTo(HaveOccurred())
		})

		It("super_admin creates an admin with a scope", func() {
			expectDept(deptA)
			expectCurr(currB, deptB)
			users.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
				Expect(u.AdminScope.Departments).To(HaveLen(1))
				Expect(u.AdminScope.Curricula).To(HaveLen(1))
				return nil
			})
			stored()
			_, err := h.Execute(ctx, superActor, port.CreateUserInput{
				Email: "a@chula.ac.th", Role: domain.RoleAdmin,
				AdminScope: &port.AdminScopeInput{DepartmentIDs: []uuid.UUID{deptA, deptA}, CurriculumIDs: []uuid.UUID{currB}},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("super_admin creates an admin without a scope (empty scope)", func() {
			users.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
				Expect(u.AdminScope).NotTo(BeNil())
				return nil
			})
			stored()
			_, err := h.Execute(ctx, superActor, port.CreateUserInput{Email: "a@chula.ac.th", Role: domain.RoleAdmin})
			Expect(err).NotTo(HaveOccurred())
		})

		It("admin creates a staff inside scope", func() {
			expectDept(deptA)
			users.EXPECT().Create(ctx, gomock.Any()).Return(nil)
			stored()
			_, err := h.Execute(ctx, admin, port.CreateUserInput{Email: "s@chula.ac.th", Role: domain.RoleStaff, DepartmentID: &deptA})
			Expect(err).NotTo(HaveOccurred())
		})

		It("admin cannot create a user outside scope", func() {
			expectDept(deptB)
			_, err := h.Execute(ctx, admin, port.CreateUserInput{Email: "s@chula.ac.th", Role: domain.RoleStaff, DepartmentID: &deptB})
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("admin cannot create a user without any placement (out of scope)", func() {
			_, err := h.Execute(ctx, admin, port.CreateUserInput{Email: "s@chula.ac.th", Role: domain.RoleInstructor})
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		DescribeTable("admin cannot assign privileged roles",
			func(role domain.Role) {
				_, err := h.Execute(ctx, admin, port.CreateUserInput{Email: "s@chula.ac.th", Role: role, DepartmentID: &deptA})
				Expect(err).To(MatchError(domain.ErrForbidden))
			},
			Entry("admin", domain.RoleAdmin),
			Entry("super_admin", domain.RoleSuperAdmin),
		)

		It("admin cannot set admin_scope", func() {
			_, err := h.Execute(ctx, admin, port.CreateUserInput{
				Email: "s@chula.ac.th", Role: domain.RoleInstructor, DepartmentID: &deptA,
				AdminScope: &port.AdminScopeInput{},
			})
			Expect(err).To(MatchError(domain.ErrInvalidInput)) // not an admin role at all
		})

		It("instructor and staff cannot create users", func() {
			for _, a := range []*domain.Actor{staffActor, {UserID: uuid.New(), Role: domain.RoleInstructor}} {
				_, err := h.Execute(ctx, a, port.CreateUserInput{Email: "s@chula.ac.th", Role: domain.RoleStaff})
				Expect(err).To(MatchError(domain.ErrForbidden))
			}
		})

		It("rejects e-mails outside the allowed domains", func() {
			_, err := h.Execute(ctx, superActor, port.CreateUserInput{Email: "x@gmail.com", Role: domain.RoleStaff})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("rejects an unknown role", func() {
			_, err := h.Execute(ctx, superActor, port.CreateUserInput{Email: "x@chula.ac.th", Role: "boss"})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("rejects admin_scope for a non-admin role", func() {
			_, err := h.Execute(ctx, superActor, port.CreateUserInput{Email: "x@chula.ac.th", Role: domain.RoleStaff, AdminScope: &port.AdminScopeInput{}})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("rejects a curriculum that does not belong to the given department", func() {
			expectCurr(currA, deptA)
			_, err := h.Execute(ctx, superActor, port.CreateUserInput{Email: "x@chula.ac.th", Role: domain.RoleStaff, DepartmentID: &deptB, CurriculumID: &currA})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("rejects an unknown curriculum, department and scope entry", func() {
			currs.EXPECT().GetByID(ctx, currA).Return(nil, domain.ErrNotFound)
			_, err := h.Execute(ctx, superActor, port.CreateUserInput{Email: "x@chula.ac.th", Role: domain.RoleStaff, CurriculumID: &currA})
			Expect(err).To(MatchError(domain.ErrInvalidInput))

			depts.EXPECT().GetByID(ctx, deptA).Return(nil, domain.ErrNotFound)
			_, err = h.Execute(ctx, superActor, port.CreateUserInput{Email: "x@chula.ac.th", Role: domain.RoleStaff, DepartmentID: &deptA})
			Expect(err).To(MatchError(domain.ErrInvalidInput))

			depts.EXPECT().GetByID(ctx, deptB).Return(nil, domain.ErrNotFound)
			_, err = h.Execute(ctx, superActor, port.CreateUserInput{
				Email: "x@chula.ac.th", Role: domain.RoleAdmin, AdminScope: &port.AdminScopeInput{DepartmentIDs: []uuid.UUID{deptB}},
			})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("maps a duplicate e-mail to conflict", func() {
			users.EXPECT().Create(ctx, gomock.Any()).Return(domain.ErrConflict)
			_, err := h.Execute(ctx, superActor, port.CreateUserInput{Email: "x@chula.ac.th", Role: domain.RoleStaff})
			Expect(err).To(MatchError(domain.ErrConflict))
		})
	})

	Describe("update", func() {
		var (
			h      *commands.UpdateUserHandler
			target *domain.User
		)
		BeforeEach(func() {
			h = commands.NewUpdateUserHandler(users, depts, currs)
			target = &domain.User{ID: uuid.New(), Email: "t@chula.ac.th", Role: domain.RoleInstructor, DepartmentID: &deptA}
		})

		loadTarget := func() { users.EXPECT().GetByID(ctx, target.ID).Return(target, nil).AnyTimes() }

		It("super_admin changes the role of another user and clears the scope of a demoted admin", func() {
			target.Role = domain.RoleAdmin
			target.AdminScope = &domain.AdminScope{Departments: []domain.Ref{{ID: deptA}}}
			loadTarget()
			expectDept(deptA)
			users.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
				Expect(u.Role).To(Equal(domain.RoleStaff))
				Expect(u.AdminScope).To(BeNil())
				return nil
			})
			_, err := h.Execute(ctx, superActor, port.UpdateUserInput{ID: target.ID, Role: ptr(domain.RoleStaff)})
			Expect(err).NotTo(HaveOccurred())
		})

		It("super_admin replaces the full admin scope", func() {
			target.Role = domain.RoleAdmin
			target.AdminScope = &domain.AdminScope{Departments: []domain.Ref{{ID: deptA}}}
			loadTarget()
			expectDept(deptA)
			expectCurr(currB, deptB)
			users.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
				Expect(u.AdminScope.Departments).To(BeEmpty())
				Expect(u.AdminScope.Curricula).To(HaveLen(1))
				return nil
			})
			_, err := h.Execute(ctx, superActor, port.UpdateUserInput{ID: target.ID, AdminScope: &port.AdminScopeInput{CurriculumIDs: []uuid.UUID{currB}}})
			Expect(err).NotTo(HaveOccurred())
		})

		It("keeps the existing scope when promoting is not part of the patch", func() {
			target.Role = domain.RoleAdmin
			target.AdminScope = &domain.AdminScope{Departments: []domain.Ref{{ID: deptA}}}
			loadTarget()
			expectDept(deptA)
			users.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
				Expect(u.AdminScope.Departments).To(HaveLen(1))
				return nil
			})
			_, err := h.Execute(ctx, superActor, port.UpdateUserInput{ID: target.ID, DepartmentID: port.OptionalID{Set: true, Value: &deptA}})
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects admin_scope for a non-admin role", func() {
			loadTarget()
			_, err := h.Execute(ctx, superActor, port.UpdateUserInput{ID: target.ID, AdminScope: &port.AdminScopeInput{}})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("super_admin cannot change their own role", func() {
			self := &domain.User{ID: superActor.UserID, Role: domain.RoleSuperAdmin}
			users.EXPECT().GetByID(ctx, self.ID).Return(self, nil)
			_, err := h.Execute(ctx, superActor, port.UpdateUserInput{ID: self.ID, Role: ptr(domain.RoleAdmin)})
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("auto-fills department from a new curriculum and drops a stale curriculum when the department changes", func() {
			loadTarget()
			expectCurr(currB, deptB)
			expectDept(deptB)
			users.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
				Expect(*u.DepartmentID).To(Equal(deptB))
				Expect(*u.CurriculumID).To(Equal(currB))
				return nil
			})
			_, err := h.Execute(ctx, superActor, port.UpdateUserInput{ID: target.ID, CurriculumID: port.OptionalID{Set: true, Value: &currB}})
			Expect(err).NotTo(HaveOccurred())
		})

		It("clears placement with explicit nulls", func() {
			target.CurriculumID = &currA
			loadTarget()
			users.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
				Expect(u.DepartmentID).To(BeNil())
				Expect(u.CurriculumID).To(BeNil())
				return nil
			})
			_, err := h.Execute(ctx, superActor, port.UpdateUserInput{
				ID: target.ID, DepartmentID: port.OptionalID{Set: true}, CurriculumID: port.OptionalID{Set: true},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects a curriculum/department mismatch", func() {
			loadTarget()
			expectCurr(currB, deptB)
			_, err := h.Execute(ctx, superActor, port.UpdateUserInput{
				ID: target.ID, DepartmentID: port.OptionalID{Set: true, Value: &deptA}, CurriculumID: port.OptionalID{Set: true, Value: &currB},
			})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("admin edits an in-scope instructor", func() {
			loadTarget()
			expectDept(deptA)
			users.EXPECT().Update(ctx, gomock.Any()).Return(nil)
			_, err := h.Execute(ctx, admin, port.UpdateUserInput{ID: target.ID, Role: ptr(domain.RoleStaff)})
			Expect(err).NotTo(HaveOccurred())
		})

		It("admin cannot promote a user to admin or super_admin", func() {
			loadTarget()
			for _, r := range []domain.Role{domain.RoleAdmin, domain.RoleSuperAdmin} {
				_, err := h.Execute(ctx, admin, port.UpdateUserInput{ID: target.ID, Role: ptr(r)})
				Expect(err).To(MatchError(domain.ErrForbidden))
			}
		})

		It("admin cannot edit another admin or a super_admin (protected rows)", func() {
			for _, r := range []domain.Role{domain.RoleAdmin, domain.RoleSuperAdmin} {
				other := &domain.User{ID: uuid.New(), Role: r, DepartmentID: &deptA}
				users.EXPECT().GetByID(ctx, other.ID).Return(other, nil)
				_, err := h.Execute(ctx, admin, port.UpdateUserInput{ID: other.ID, Role: ptr(domain.RoleStaff)})
				Expect(err).To(MatchError(domain.ErrForbidden))
			}
		})

		It("admin cannot edit a user out of scope", func() {
			target.DepartmentID = &deptB
			loadTarget()
			_, err := h.Execute(ctx, admin, port.UpdateUserInput{ID: target.ID, Role: ptr(domain.RoleStaff)})
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("admin cannot move an in-scope user out of scope", func() {
			loadTarget()
			expectDept(deptB)
			_, err := h.Execute(ctx, admin, port.UpdateUserInput{ID: target.ID, DepartmentID: port.OptionalID{Set: true, Value: &deptB}})
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("admin cannot write admin_scope", func() {
			loadTarget()
			_, err := h.Execute(ctx, admin, port.UpdateUserInput{ID: target.ID, AdminScope: &port.AdminScopeInput{}})
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("staff cannot update anybody", func() {
			_, err := h.Execute(ctx, staffActor, port.UpdateUserInput{ID: target.ID})
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("reports an unknown user as not found", func() {
			users.EXPECT().GetByID(ctx, target.ID).Return(nil, domain.ErrNotFound)
			_, err := h.Execute(ctx, superActor, port.UpdateUserInput{ID: target.ID})
			Expect(err).To(MatchError(domain.ErrNotFound))
		})

		It("rejects an unknown role", func() {
			loadTarget()
			_, err := h.Execute(ctx, superActor, port.UpdateUserInput{ID: target.ID, Role: ptr(domain.Role("boss"))})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})
	})

	Describe("delete", func() {
		var h *commands.DeleteUserHandler
		BeforeEach(func() { h = commands.NewDeleteUserHandler(users) })

		It("super_admin deletes another user", func() {
			t := &domain.User{ID: uuid.New(), Role: domain.RoleAdmin}
			users.EXPECT().GetByID(ctx, t.ID).Return(t, nil)
			users.EXPECT().Delete(ctx, t.ID).Return(nil)
			Expect(h.Execute(ctx, superActor, t.ID)).To(Succeed())
		})

		It("nobody can delete themselves (409)", func() {
			Expect(h.Execute(ctx, superActor, superActor.UserID)).To(MatchError(domain.ErrConflict))
			Expect(h.Execute(ctx, admin, admin.UserID)).To(MatchError(domain.ErrConflict))
		})

		It("admin deletes an in-scope staff", func() {
			t := &domain.User{ID: uuid.New(), Role: domain.RoleStaff, CurriculumID: &currA, DepartmentID: &deptA}
			users.EXPECT().GetByID(ctx, t.ID).Return(t, nil)
			users.EXPECT().Delete(ctx, t.ID).Return(nil)
			Expect(h.Execute(ctx, admin, t.ID)).To(Succeed())
		})

		It("admin cannot delete an out-of-scope user or an admin", func() {
			out := &domain.User{ID: uuid.New(), Role: domain.RoleStaff, DepartmentID: &deptB}
			adm := &domain.User{ID: uuid.New(), Role: domain.RoleAdmin, DepartmentID: &deptA}
			users.EXPECT().GetByID(ctx, out.ID).Return(out, nil)
			users.EXPECT().GetByID(ctx, adm.ID).Return(adm, nil)
			Expect(h.Execute(ctx, admin, out.ID)).To(MatchError(domain.ErrForbidden))
			Expect(h.Execute(ctx, admin, adm.ID)).To(MatchError(domain.ErrForbidden))
		})

		It("staff cannot delete", func() {
			Expect(h.Execute(ctx, staffActor, uuid.New())).To(MatchError(domain.ErrForbidden))
		})

		It("reports an unknown user as not found", func() {
			id := uuid.New()
			users.EXPECT().GetByID(ctx, id).Return(nil, domain.ErrNotFound)
			Expect(h.Execute(ctx, superActor, id)).To(MatchError(domain.ErrNotFound))
		})
	})
})

var _ = Describe("auth commands", func() {
	var (
		ctrl  *gomock.Controller
		users *mocks.MockUserRepository
		ctx   = context.Background()
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		users = mocks.NewMockUserRepository(ctrl)
	})
	AfterEach(func() { ctrl.Finish() })

	Describe("login sync", func() {
		It("normalises the e-mail and records the login", func() {
			u := &domain.User{ID: uuid.New(), Email: "a@chula.ac.th", Role: domain.RoleInstructor}
			users.EXPECT().TouchLogin(ctx, "a@chula.ac.th", "Alice", "https://x/p.png").Return(u, nil)
			got, err := commands.NewLoginSyncHandler(users).Execute(ctx, port.LoginSyncInput{Email: " A@Chula.ac.th ", Name: " Alice ", PictureURL: "https://x/p.png"})
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(u))
		})

		It("answers forbidden / not provisioned for an unknown e-mail", func() {
			users.EXPECT().TouchLogin(ctx, "ghost@chula.ac.th", "", "").Return(nil, domain.ErrNotFound)
			_, err := commands.NewLoginSyncHandler(users).Execute(ctx, port.LoginSyncInput{Email: "ghost@chula.ac.th"})
			Expect(err).To(MatchError(domain.ErrForbidden))
			Expect(err.Error()).To(ContainSubstring("not provisioned"))
		})

		It("rejects a missing e-mail", func() {
			_, err := commands.NewLoginSyncHandler(users).Execute(ctx, port.LoginSyncInput{Email: "  "})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("passes database failures through", func() {
			boom := errors.New("db")
			users.EXPECT().TouchLogin(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, boom)
			_, err := commands.NewLoginSyncHandler(users).Execute(ctx, port.LoginSyncInput{Email: "a@chula.ac.th"})
			Expect(err).To(MatchError(boom))
		})
	})

	Describe("bootstrap super admins", func() {
		It("upserts every distinct normalised e-mail and counts them", func() {
			users.EXPECT().UpsertSuperAdmin(ctx, "root@chula.ac.th").Return(true, nil)
			users.EXPECT().UpsertSuperAdmin(ctx, "boss@chula.ac.th").Return(false, nil)
			n, err := commands.NewBootstrapSuperAdminsHandler(users).Execute(ctx, []string{" Root@Chula.ac.th", "root@chula.ac.th", "", "no-at-sign", "BOSS@chula.ac.th"})
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(2))
		})

		It("does nothing for an empty list", func() {
			n, err := commands.NewBootstrapSuperAdminsHandler(users).Execute(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(0))
		})

		It("stops with an error when the repository fails", func() {
			users.EXPECT().UpsertSuperAdmin(ctx, "a@chula.ac.th").Return(false, errors.New("db"))
			_, err := commands.NewBootstrapSuperAdminsHandler(users).Execute(ctx, []string{"a@chula.ac.th"})
			Expect(err).To(HaveOccurred())
		})
	})
})
