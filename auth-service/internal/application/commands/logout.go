package commands

import (
	"context"
	"errors"
	"time"

	"abet/auth-service/internal/core/domain"
	"abet/auth-service/internal/core/port"
)

// LogoutHandler revokes the session referenced by a (valid) session token.
type LogoutHandler struct {
	tokens   port.TokenService
	sessions port.SessionRepository
	now      func() time.Time
}

// NewLogoutHandler creates the handler; now may be nil (defaults to time.Now).
func NewLogoutHandler(tokens port.TokenService, sessions port.SessionRepository, now func() time.Time) *LogoutHandler {
	if now == nil {
		now = time.Now
	}
	return &LogoutHandler{tokens: tokens, sessions: sessions, now: now}
}

// Execute never fails for invalid tokens: logout is idempotent and the caller always clears the cookie.
func (h *LogoutHandler) Execute(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	claims, err := h.tokens.Parse(token)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			return nil
		}
		return err
	}
	return h.sessions.Revoke(ctx, claims.SessionID, h.now().UTC())
}

var _ port.LogoutUseCase = (*LogoutHandler)(nil)
