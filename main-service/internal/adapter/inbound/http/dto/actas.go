package dto

import (
	"fmt"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// ActAsRequest is the body of POST /dev/act-as.
type ActAsRequest struct {
	UserID uuid.UUID `json:"user_id"`
}

func (r ActAsRequest) Validate() error {
	if r.UserID == uuid.Nil {
		return fmt.Errorf("%w: user_id is required", domain.ErrInvalidInput)
	}
	return nil
}

// ActAsUserResponse is one switchable user.
type ActAsUserResponse struct {
	ID           uuid.UUID   `json:"id"`
	Email        string      `json:"email"`
	Name         string      `json:"name"`
	Role         domain.Role `json:"role"`
	DepartmentID *uuid.UUID  `json:"department_id"`
	CurriculumID *uuid.UUID  `json:"curriculum_id"`
}

type ActAsUsersResponse struct {
	Users []ActAsUserResponse `json:"users"`
}

func ActAsUsersFromDomain(us []domain.User) ActAsUsersResponse {
	out := make([]ActAsUserResponse, 0, len(us))
	for _, u := range us {
		out = append(out, ActAsUserResponse{
			ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role, DepartmentID: u.DepartmentID, CurriculumID: u.CurriculumID,
		})
	}
	return ActAsUsersResponse{Users: out}
}

// ActAsRealUser is the real super_admin shown in the acting banner.
type ActAsRealUser struct {
	ID    uuid.UUID   `json:"id"`
	Email string      `json:"email"`
	Name  string      `json:"name"`
	Role  domain.Role `json:"role"`
}

// ActAsStateResponse is the "act_as" block of GET /me.
type ActAsStateResponse struct {
	Available bool           `json:"available"`
	RealUser  *ActAsRealUser `json:"real_user"`
}

func ActAsStateFromPort(s *port.ActAsState) ActAsStateResponse {
	out := ActAsStateResponse{Available: s.Available}
	if s.RealUser != nil {
		out.RealUser = &ActAsRealUser{ID: s.RealUser.ID, Email: s.RealUser.Email, Name: s.RealUser.Name, Role: s.RealUser.Role}
	}
	return out
}
