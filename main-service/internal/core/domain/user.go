package domain

import (
	"time"

	"github.com/google/uuid"
)

// AdminScope lists what an admin may manage (only meaningful for role=admin).
type AdminScope struct {
	Departments []Ref
	Curricula   []Ref
}

// User is a pre-provisioned account.
type User struct {
	ID           uuid.UUID
	Email        string
	Name         string
	PictureURL   string
	Role         Role
	DepartmentID *uuid.UUID
	CurriculumID *uuid.UUID
	Department   *Ref
	Curriculum   *Ref
	// AdminScope is non-nil only for role=admin.
	AdminScope  *AdminScope
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ScopeInfo returns the fields scope checks need.
func (u *User) ScopeInfo() UserScopeInfo {
	return UserScopeInfo{UserID: u.ID, Role: u.Role, DepartmentID: u.DepartmentID, CurriculumID: u.CurriculumID}
}
