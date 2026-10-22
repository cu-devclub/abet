package dto

import (
	"encoding/json"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// OptionalUUID tells apart an absent JSON field from an explicit null (PATCH semantics).
type OptionalUUID struct {
	Set   bool
	Value *uuid.UUID
}

func (o *OptionalUUID) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = nil
		return nil
	}
	var id uuid.UUID
	if err := json.Unmarshal(b, &id); err != nil {
		return err
	}
	o.Value = &id
	return nil
}

func (o OptionalUUID) toPort() port.OptionalID { return port.OptionalID{Set: o.Set, Value: o.Value} }

type AdminScopeRequest struct {
	DepartmentIDs []uuid.UUID `json:"department_ids"`
	CurriculumIDs []uuid.UUID `json:"curriculum_ids"`
}

func (r *AdminScopeRequest) toInput() *port.AdminScopeInput {
	if r == nil {
		return nil
	}
	return &port.AdminScopeInput{DepartmentIDs: r.DepartmentIDs, CurriculumIDs: r.CurriculumIDs}
}

type CreateUserRequest struct {
	Email        string             `json:"email" validate:"required,max=254"`
	Role         string             `json:"role" validate:"required"`
	DepartmentID *uuid.UUID         `json:"department_id"`
	CurriculumID *uuid.UUID         `json:"curriculum_id"`
	AdminScope   *AdminScopeRequest `json:"admin_scope"`
}

func (r CreateUserRequest) Validate() error { return Validate(r) }

func (r CreateUserRequest) ToInput() port.CreateUserInput {
	return port.CreateUserInput{
		Email: r.Email, Role: domain.Role(r.Role), DepartmentID: r.DepartmentID,
		CurriculumID: r.CurriculumID, AdminScope: r.AdminScope.toInput(),
	}
}

// UpdateUserRequest: every field optional; email is immutable and therefore not accepted.
type UpdateUserRequest struct {
	Role         *string            `json:"role"`
	DepartmentID OptionalUUID       `json:"department_id"`
	CurriculumID OptionalUUID       `json:"curriculum_id"`
	AdminScope   *AdminScopeRequest `json:"admin_scope"`
}

func (r UpdateUserRequest) Validate() error { return Validate(r) }

func (r UpdateUserRequest) ToInput(id uuid.UUID) port.UpdateUserInput {
	in := port.UpdateUserInput{
		ID: id, DepartmentID: r.DepartmentID.toPort(), CurriculumID: r.CurriculumID.toPort(),
		AdminScope: r.AdminScope.toInput(),
	}
	if r.Role != nil {
		role := domain.Role(*r.Role)
		in.Role = &role
	}
	return in
}

type AdminScopeResponse struct {
	Departments []RefResponse `json:"departments"`
	Curricula   []RefResponse `json:"curricula"`
}

type UserResponse struct {
	ID          uuid.UUID           `json:"id"`
	Email       string              `json:"email"`
	Name        string              `json:"name"`
	PictureURL  string              `json:"picture_url"`
	Role        domain.Role         `json:"role"`
	Department  *RefResponse        `json:"department"`
	Curriculum  *RefResponse        `json:"curriculum"`
	AdminScope  *AdminScopeResponse `json:"admin_scope"`
	LastLoginAt *string             `json:"last_login_at"`
	Editable    bool                `json:"editable"`
	CreatedAt   string              `json:"created_at"`
	UpdatedAt   string              `json:"updated_at"`
}

// UserFromDomain maps a user for the given caller; editable is true when the caller may edit/delete the row
// (never for the caller's own row, which cannot be deleted or re-roled).
func UserFromDomain(u *domain.User, caller *domain.Actor) UserResponse {
	out := UserResponse{
		ID: u.ID, Email: u.Email, Name: u.Name, PictureURL: u.PictureURL, Role: u.Role,
		Department: refPtr(u.Department), Curriculum: refPtr(u.Curriculum),
		LastLoginAt: formatTimePtr(u.LastLoginAt),
		Editable:    caller.CanManageUser(u.ScopeInfo()) && u.ID != caller.UserID,
		CreatedAt:   FormatTime(u.CreatedAt), UpdatedAt: FormatTime(u.UpdatedAt),
	}
	if u.Role == domain.RoleAdmin {
		scope := &AdminScopeResponse{Departments: []RefResponse{}, Curricula: []RefResponse{}}
		if u.AdminScope != nil {
			for _, d := range u.AdminScope.Departments {
				scope.Departments = append(scope.Departments, RefFromDomain(d))
			}
			for _, c := range u.AdminScope.Curricula {
				scope.Curricula = append(scope.Curricula, RefFromDomain(c))
			}
		}
		out.AdminScope = scope
	}
	return out
}

func UsersFromDomain(us []domain.User, caller *domain.Actor) []UserResponse {
	out := make([]UserResponse, 0, len(us))
	for i := range us {
		out = append(out, UserFromDomain(&us[i], caller))
	}
	return out
}

type CapabilitiesResponse struct {
	ManageOrg     bool `json:"manage_org"`
	ManageUsers   bool `json:"manage_users"`
	ManageCourses bool `json:"manage_courses"`
	ManageCovers  bool `json:"manage_covers"`
}

type MeResponse struct {
	User         UserResponse         `json:"user"`
	Capabilities CapabilitiesResponse `json:"capabilities"`
	ActAs        ActAsStateResponse   `json:"act_as"`
}

func MeFromResult(r *port.MeResult, caller *domain.Actor) MeResponse {
	return MeResponse{
		User: UserFromDomain(r.User, caller),
		Capabilities: CapabilitiesResponse{
			ManageOrg: r.ManageOrg, ManageUsers: r.ManageUsers, ManageCourses: r.ManageCourses, ManageCovers: r.ManageCovers,
		},
	}
}
