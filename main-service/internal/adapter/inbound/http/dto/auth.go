package dto

import (
	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// LoginSyncRequest is posted by auth-service after a successful Google login.
type LoginSyncRequest struct {
	Email      string `json:"email" validate:"required,max=254"`
	Name       string `json:"name" validate:"max=300"`
	PictureURL string `json:"picture_url" validate:"max=2000"`
}

func (r LoginSyncRequest) Validate() error { return Validate(r) }

func (r LoginSyncRequest) ToInput() port.LoginSyncInput {
	return port.LoginSyncInput{Email: r.Email, Name: r.Name, PictureURL: r.PictureURL}
}

type LoginSyncResponse struct {
	UserID uuid.UUID   `json:"user_id"`
	Email  string      `json:"email"`
	Role   domain.Role `json:"role"`
}

func LoginSyncFromDomain(u *domain.User) LoginSyncResponse {
	return LoginSyncResponse{UserID: u.ID, Email: u.Email, Role: u.Role}
}
