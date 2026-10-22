package domain

import (
	"slices"

	"github.com/google/uuid"
)

// Role is the single global role of a user.
type Role string

const (
	RoleSuperAdmin Role = "super_admin"
	RoleAdmin      Role = "admin"
	RoleInstructor Role = "instructor"
	RoleStaff      Role = "staff"
)

// Valid reports whether r is one of the four known roles.
func (r Role) Valid() bool {
	switch r {
	case RoleSuperAdmin, RoleAdmin, RoleInstructor, RoleStaff:
		return true
	}
	return false
}

// Rank orders roles by privilege (higher = more privileged); unknown roles rank 0.
func (r Role) Rank() int {
	switch r {
	case RoleSuperAdmin:
		return 4
	case RoleAdmin:
		return 3
	case RoleInstructor:
		return 2
	case RoleStaff:
		return 1
	}
	return 0
}

// Actor is the authenticated caller, rebuilt from main_db on every request.
type Actor struct {
	UserID       uuid.UUID
	Email        string
	Role         Role
	DepartmentID *uuid.UUID
	CurriculumID *uuid.UUID

	// ScopeDepartmentIDs are the departments an admin manages in full (admin_scopes.department_id).
	ScopeDepartmentIDs []uuid.UUID
	// ScopeCurriculumIDs are the curricula an admin manages: explicit ones plus every curriculum of a scoped department.
	ScopeCurriculumIDs []uuid.UUID
	// ScopeCurriculumDepartmentIDs are the departments that own at least one curriculum in ScopeCurriculumIDs
	// (needed for the "user without curriculum" scope rule in authz.md).
	ScopeCurriculumDepartmentIDs []uuid.UUID
}

// UserScopeInfo is the part of a user that scope checks depend on.
type UserScopeInfo struct {
	UserID       uuid.UUID
	Role         Role
	DepartmentID *uuid.UUID
	CurriculumID *uuid.UUID
}

func (a *Actor) isSuper() bool { return a.Role == RoleSuperAdmin }
func (a *Actor) isAdmin() bool { return a.Role == RoleAdmin }

// CanManageOrg: departments and curricula writes are super_admin only.
func (a *Actor) CanManageOrg() bool { return a.isSuper() }

// CanManageUsers: may use the users management endpoints at all (super_admin or admin).
func (a *Actor) CanManageUsers() bool { return a.isSuper() || a.isAdmin() }

// CanManageCoursesAny: may use the courses/covers management area (super_admin or admin).
func (a *Actor) CanManageCoursesAny() bool { return a.isSuper() || a.isAdmin() }

// CanManageCurriculum: super_admin, or admin with the curriculum in scope.
func (a *Actor) CanManageCurriculum(curriculumID uuid.UUID) bool {
	if a.isSuper() {
		return true
	}
	return a.isAdmin() && slices.Contains(a.ScopeCurriculumIDs, curriculumID)
}

// InScope reports whether the user described by t falls inside this admin's scope (authz.md "User u in scope").
// super_admin: always; any other non-admin role: never.
func (a *Actor) InScope(t UserScopeInfo) bool {
	if a.isSuper() {
		return true
	}
	if !a.isAdmin() {
		return false
	}
	if t.CurriculumID != nil && slices.Contains(a.ScopeCurriculumIDs, *t.CurriculumID) {
		return true
	}
	if t.DepartmentID != nil && slices.Contains(a.ScopeDepartmentIDs, *t.DepartmentID) {
		return true
	}
	if t.CurriculumID == nil && t.DepartmentID != nil && slices.Contains(a.ScopeCurriculumDepartmentIDs, *t.DepartmentID) {
		return true
	}
	return false
}

// CanSeeUser: super_admin sees everyone, admin sees users in scope, others see no one.
func (a *Actor) CanSeeUser(t UserScopeInfo) bool {
	return a.InScope(t)
}

// CanManageUser: may edit/delete target t. super_admin: anyone; admin: only instructor/staff inside scope.
// Self-protection (no self delete / own role change) is enforced by the use cases.
func (a *Actor) CanManageUser(t UserScopeInfo) bool {
	if a.isSuper() {
		return true
	}
	if !a.isAdmin() {
		return false
	}
	if t.Role != RoleInstructor && t.Role != RoleStaff {
		return false
	}
	return a.InScope(t)
}

// CanAssignRole: super_admin may assign any role; admin only instructor or staff.
func (a *Actor) CanAssignRole(r Role) bool {
	if a.isSuper() {
		return r.Valid()
	}
	return a.isAdmin() && (r == RoleInstructor || r == RoleStaff)
}

// CanSetAdminScope: only super_admin may write admin_scopes.
func (a *Actor) CanSetAdminScope() bool { return a.isSuper() }

// OfferingScope identifies where a course offering sits in the organisation.
type OfferingScope struct {
	CurriculumID uuid.UUID
	DepartmentID uuid.UUID
}

// AccessGrant is an explicit course_access row (nil when none).
type AccessGrant struct {
	CanUpload   bool
	CanDownload bool
}

// OfferingPerms is the effective permission set of an actor on one offering.
type OfferingPerms struct {
	View, Upload, Download, Manage bool
	// IsDefault is true when download is granted only by the staff-department default (A3).
	IsDefault bool
}

// OfferingPermissions implements the per-offering table of authz.md.
func (a *Actor) OfferingPermissions(o OfferingScope, grant *AccessGrant) OfferingPerms {
	if a.isSuper() || (a.isAdmin() && slices.Contains(a.ScopeCurriculumIDs, o.CurriculumID)) {
		return OfferingPerms{View: true, Upload: true, Download: true, Manage: true}
	}

	var p OfferingPerms
	if grant != nil {
		p.Upload = grant.CanUpload
		p.Download = grant.CanDownload
	}
	if a.Role == RoleStaff && a.DepartmentID != nil && *a.DepartmentID == o.DepartmentID && !p.Download {
		p.Download = true
		p.IsDefault = true
	}
	p.View = p.Upload || p.Download
	return p
}
