package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// LoginSyncHandler implements port.LoginSyncUseCase.
type LoginSyncHandler struct {
	users port.UserRepository
}

var _ port.LoginSyncUseCase = (*LoginSyncHandler)(nil)

func NewLoginSyncHandler(users port.UserRepository) *LoginSyncHandler {
	return &LoginSyncHandler{users: users}
}

func (h *LoginSyncHandler) Execute(ctx context.Context, in port.LoginSyncInput) (*domain.User, error) {
	email := domain.NormalizeEmail(in.Email)
	if email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("%w: email is required", domain.ErrInvalidInput)
	}
	u, err := h.users.TouchLogin(ctx, email, strings.TrimSpace(in.Name), strings.TrimSpace(in.PictureURL))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("%w: not provisioned", domain.ErrForbidden)
		}
		return nil, err
	}
	return u, nil
}

// BootstrapSuperAdminsHandler implements port.BootstrapSuperAdminsUseCase.
type BootstrapSuperAdminsHandler struct {
	users port.UserRepository
}

var _ port.BootstrapSuperAdminsUseCase = (*BootstrapSuperAdminsHandler)(nil)

func NewBootstrapSuperAdminsHandler(users port.UserRepository) *BootstrapSuperAdminsHandler {
	return &BootstrapSuperAdminsHandler{users: users}
}

// Execute upserts every valid e-mail as super_admin and returns how many were processed.
func (h *BootstrapSuperAdminsHandler) Execute(ctx context.Context, emails []string) (int, error) {
	seen := map[string]bool{}
	n := 0
	for _, raw := range emails {
		email := domain.NormalizeEmail(raw)
		if email == "" || !strings.Contains(email, "@") || seen[email] {
			continue
		}
		seen[email] = true
		if _, err := h.users.UpsertSuperAdmin(ctx, email); err != nil {
			return n, fmt.Errorf("bootstrap %s: %w", email, err)
		}
		n++
	}
	return n, nil
}
