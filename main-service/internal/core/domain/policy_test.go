package domain_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/main-service/internal/core/domain"
)

var (
	deptA  = uuid.New()
	deptB  = uuid.New()
	currA1 = uuid.New() // belongs to deptA
	currA2 = uuid.New() // belongs to deptA
	currB1 = uuid.New() // belongs to deptB

	superAdmin = &domain.Actor{UserID: uuid.New(), Role: domain.RoleSuperAdmin}
	// admin scoped to the whole department A (=> curricula A1, A2)
	adminDept = &domain.Actor{
		UserID: uuid.New(), Role: domain.RoleAdmin,
		ScopeDepartmentIDs:           []uuid.UUID{deptA},
		ScopeCurriculumIDs:           []uuid.UUID{currA1, currA2},
		ScopeCurriculumDepartmentIDs: []uuid.UUID{deptA},
	}
	// admin scoped to the single curriculum A1
	adminCurr = &domain.Actor{
		UserID: uuid.New(), Role: domain.RoleAdmin,
		ScopeCurriculumIDs:           []uuid.UUID{currA1},
		ScopeCurriculumDepartmentIDs: []uuid.UUID{deptA},
	}
	// admin without any scope
	adminNone  = &domain.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}
	instructor = &domain.Actor{UserID: uuid.New(), Role: domain.RoleInstructor, DepartmentID: &deptA}
	staffA     = &domain.Actor{UserID: uuid.New(), Role: domain.RoleStaff, DepartmentID: &deptA}
	staffNoDep = &domain.Actor{UserID: uuid.New(), Role: domain.RoleStaff}
)

func ptr[T any](v T) *T { return &v }

var _ = Describe("Actor policy", func() {
	Describe("organisation and curriculum management", func() {
		It("CanManageOrg is super_admin only", func() {
			Expect(superAdmin.CanManageOrg()).To(BeTrue())
			for _, a := range []*domain.Actor{adminDept, adminCurr, adminNone, instructor, staffA} {
				Expect(a.CanManageOrg()).To(BeFalse(), string(a.Role))
			}
		})

		DescribeTable("CanManageCurriculum",
			func(a *domain.Actor, curriculum uuid.UUID, want bool) {
				Expect(a.CanManageCurriculum(curriculum)).To(Equal(want))
			},
			Entry("super: any curriculum", superAdmin, currB1, true),
			Entry("admin(dept A): curriculum of dept A", adminDept, currA2, true),
			Entry("admin(dept A): curriculum of dept B", adminDept, currB1, false),
			Entry("admin(curr A1): own curriculum", adminCurr, currA1, true),
			Entry("admin(curr A1): sibling curriculum", adminCurr, currA2, false),
			Entry("admin(no scope): nothing", adminNone, currA1, false),
			Entry("instructor: never", instructor, currA1, false),
			Entry("staff: never", staffA, currA1, false),
		)

		It("CanManageUsers / CanManageCoursesAny are super_admin or admin", func() {
			for _, a := range []*domain.Actor{superAdmin, adminDept, adminNone} {
				Expect(a.CanManageUsers()).To(BeTrue())
				Expect(a.CanManageCoursesAny()).To(BeTrue())
			}
			for _, a := range []*domain.Actor{instructor, staffA} {
				Expect(a.CanManageUsers()).To(BeFalse())
				Expect(a.CanManageCoursesAny()).To(BeFalse())
			}
		})
	})

	Describe("users in scope", func() {
		DescribeTable("CanSeeUser",
			func(a *domain.Actor, t domain.UserScopeInfo, want bool) {
				Expect(a.CanSeeUser(t)).To(Equal(want))
			},
			Entry("super sees a user with no placement", superAdmin, domain.UserScopeInfo{Role: domain.RoleStaff}, true),
			Entry("admin(dept A): user with curriculum in scope", adminDept,
				domain.UserScopeInfo{Role: domain.RoleInstructor, CurriculumID: &currA1, DepartmentID: &deptA}, true),
			Entry("admin(dept A): user with department in ScopeDepts", adminDept,
				domain.UserScopeInfo{Role: domain.RoleStaff, DepartmentID: &deptA}, true),
			Entry("admin(dept A): user of another department", adminDept,
				domain.UserScopeInfo{Role: domain.RoleStaff, DepartmentID: &deptB, CurriculumID: &currB1}, false),
			Entry("admin(dept A): user without placement", adminDept, domain.UserScopeInfo{Role: domain.RoleStaff}, false),
			Entry("admin(curr A1): user in that curriculum", adminCurr,
				domain.UserScopeInfo{Role: domain.RoleInstructor, CurriculumID: &currA1, DepartmentID: &deptA}, true),
			Entry("admin(curr A1): user in sibling curriculum", adminCurr,
				domain.UserScopeInfo{Role: domain.RoleInstructor, CurriculumID: &currA2, DepartmentID: &deptA}, false),
			Entry("admin(curr A1): dept-only user whose dept owns a scoped curriculum (rule 3)", adminCurr,
				domain.UserScopeInfo{Role: domain.RoleStaff, DepartmentID: &deptA}, true),
			Entry("admin(curr A1): dept-only user of unrelated dept", adminCurr,
				domain.UserScopeInfo{Role: domain.RoleStaff, DepartmentID: &deptB}, false),
			Entry("admin(no scope): nobody", adminNone,
				domain.UserScopeInfo{Role: domain.RoleStaff, DepartmentID: &deptA}, false),
			Entry("instructor sees nobody", instructor, domain.UserScopeInfo{Role: domain.RoleStaff, DepartmentID: &deptA}, false),
			Entry("staff sees nobody", staffA, domain.UserScopeInfo{Role: domain.RoleStaff, DepartmentID: &deptA}, false),
		)

		DescribeTable("CanManageUser",
			func(a *domain.Actor, t domain.UserScopeInfo, want bool) {
				Expect(a.CanManageUser(t)).To(Equal(want))
			},
			Entry("super manages an admin", superAdmin, domain.UserScopeInfo{Role: domain.RoleAdmin}, true),
			Entry("super manages a super_admin", superAdmin, domain.UserScopeInfo{Role: domain.RoleSuperAdmin}, true),
			Entry("admin manages in-scope instructor", adminDept,
				domain.UserScopeInfo{Role: domain.RoleInstructor, DepartmentID: &deptA}, true),
			Entry("admin manages in-scope staff", adminDept,
				domain.UserScopeInfo{Role: domain.RoleStaff, CurriculumID: &currA1, DepartmentID: &deptA}, true),
			Entry("admin may NOT manage in-scope admin (protected)", adminDept,
				domain.UserScopeInfo{Role: domain.RoleAdmin, DepartmentID: &deptA}, false),
			Entry("admin may NOT manage super_admin", adminDept,
				domain.UserScopeInfo{Role: domain.RoleSuperAdmin, DepartmentID: &deptA}, false),
			Entry("admin may NOT manage out-of-scope instructor", adminDept,
				domain.UserScopeInfo{Role: domain.RoleInstructor, DepartmentID: &deptB}, false),
			Entry("instructor manages nobody", instructor, domain.UserScopeInfo{Role: domain.RoleStaff, DepartmentID: &deptA}, false),
			Entry("staff manages nobody", staffA, domain.UserScopeInfo{Role: domain.RoleStaff, DepartmentID: &deptA}, false),
		)

		DescribeTable("CanAssignRole",
			func(a *domain.Actor, r domain.Role, want bool) {
				Expect(a.CanAssignRole(r)).To(Equal(want))
			},
			Entry("super -> super_admin", superAdmin, domain.RoleSuperAdmin, true),
			Entry("super -> admin", superAdmin, domain.RoleAdmin, true),
			Entry("super -> instructor", superAdmin, domain.RoleInstructor, true),
			Entry("super -> staff", superAdmin, domain.RoleStaff, true),
			Entry("super -> unknown", superAdmin, domain.Role("boss"), false),
			Entry("admin -> super_admin", adminDept, domain.RoleSuperAdmin, false),
			Entry("admin -> admin", adminDept, domain.RoleAdmin, false),
			Entry("admin -> instructor", adminDept, domain.RoleInstructor, true),
			Entry("admin -> staff", adminDept, domain.RoleStaff, true),
			Entry("instructor -> staff", instructor, domain.RoleStaff, false),
			Entry("staff -> staff", staffA, domain.RoleStaff, false),
		)

		It("only super_admin may set admin scopes", func() {
			Expect(superAdmin.CanSetAdminScope()).To(BeTrue())
			for _, a := range []*domain.Actor{adminDept, instructor, staffA} {
				Expect(a.CanSetAdminScope()).To(BeFalse())
			}
		})
	})

	// One entry per row of the authz.md per-offering table.
	Describe("OfferingPermissions", func() {
		inA1 := domain.OfferingScope{CurriculumID: currA1, DepartmentID: deptA}
		inA2 := domain.OfferingScope{CurriculumID: currA2, DepartmentID: deptA}
		inB1 := domain.OfferingScope{CurriculumID: currB1, DepartmentID: deptB}

		full := domain.OfferingPerms{View: true, Upload: true, Download: true, Manage: true}
		none := domain.OfferingPerms{}

		DescribeTable("table",
			func(a *domain.Actor, o domain.OfferingScope, grant *domain.AccessGrant, want domain.OfferingPerms) {
				Expect(a.OfferingPermissions(o, grant)).To(Equal(want))
			},
			// super_admin: everything, grants irrelevant
			Entry("super_admin, no grant", superAdmin, inB1, nil, full),
			Entry("super_admin, restrictive grant", superAdmin, inB1, &domain.AccessGrant{}, full),

			// admin, curriculum in scope
			Entry("admin, c in scope (dept scope)", adminDept, inA2, nil, full),
			Entry("admin, c in scope (curriculum scope)", adminCurr, inA1, nil, full),

			// admin, c not in scope: same as instructor (grant only), never manage
			Entry("admin, c not in scope, no grant", adminCurr, inA2, nil, none),
			Entry("admin, c not in scope, other dept, no grant", adminDept, inB1, nil, none),
			Entry("admin, c not in scope, upload+download grant", adminDept, inB1, &domain.AccessGrant{CanUpload: true, CanDownload: true},
				domain.OfferingPerms{View: true, Upload: true, Download: true}),
			Entry("admin, c not in scope, download-only grant", adminCurr, inA2, &domain.AccessGrant{CanDownload: true},
				domain.OfferingPerms{View: true, Download: true}),
			Entry("admin without scope, same-dept offering gets no default", adminNone, inA1, nil, none),

			// instructor: grant only
			Entry("instructor, no row", instructor, inA1, nil, none),
			Entry("instructor, upload-only", instructor, inA1, &domain.AccessGrant{CanUpload: true},
				domain.OfferingPerms{View: true, Upload: true}),
			Entry("instructor, download-only", instructor, inA1, &domain.AccessGrant{CanDownload: true},
				domain.OfferingPerms{View: true, Download: true}),
			Entry("instructor, both", instructor, inA1, &domain.AccessGrant{CanUpload: true, CanDownload: true},
				domain.OfferingPerms{View: true, Upload: true, Download: true}),
			Entry("instructor of the same department still gets no default", instructor, inA2, nil, none),

			// staff: row OR own department (default download)
			Entry("staff, other department, no row", staffA, inB1, nil, none),
			Entry("staff, other department, download row", staffA, inB1, &domain.AccessGrant{CanDownload: true},
				domain.OfferingPerms{View: true, Download: true}),
			Entry("staff, other department, upload row", staffA, inB1, &domain.AccessGrant{CanUpload: true},
				domain.OfferingPerms{View: true, Upload: true}),
			Entry("staff, own department, no row => default download", staffA, inA1, nil,
				domain.OfferingPerms{View: true, Download: true, IsDefault: true}),
			Entry("staff, own department, upload-only row => upload + default download", staffA, inA1,
				&domain.AccessGrant{CanUpload: true},
				domain.OfferingPerms{View: true, Upload: true, Download: true, IsDefault: true}),
			Entry("staff, own department, download row => explicit, not default", staffA, inA1,
				&domain.AccessGrant{CanDownload: true},
				domain.OfferingPerms{View: true, Download: true}),
			Entry("staff, own department, both row => explicit", staffA, inA1,
				&domain.AccessGrant{CanUpload: true, CanDownload: true},
				domain.OfferingPerms{View: true, Upload: true, Download: true}),
			Entry("staff without a department gets no default", staffNoDep, inA1, nil, none),
		)

		It("never grants manage to non-admin roles", func() {
			g := &domain.AccessGrant{CanUpload: true, CanDownload: true}
			for _, a := range []*domain.Actor{instructor, staffA} {
				Expect(a.OfferingPermissions(inA1, g).Manage).To(BeFalse())
			}
		})
	})
})
