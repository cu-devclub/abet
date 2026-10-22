package queries

import (
	"context"
	"errors"
	"time"

	"abet/auth-service/internal/core/domain"
	"abet/auth-service/internal/core/port"
)

// VerifySessionHandler validates a session token and its server-side session row.
type VerifySessionHandler struct {
	tokens   port.TokenService
	sessions port.SessionRepository
	now      func() time.Time
}

// NewVerifySessionHandler creates the handler; now may be nil (defaults to time.Now).
func NewVerifySessionHandler(tokens port.TokenService, sessions port.SessionRepository, now func() time.Time) *VerifySessionHandler {
	if now == nil {
		now = time.Now
	}
	return &VerifySessionHandler{tokens: tokens, sessions: sessions, now: now}
}

// Execute returns domain.ErrUnauthorized for any invalid, revoked, expired or unknown session.
func (h *VerifySessionHandler) Execute(ctx context.Context, token string) (port.VerifiedSession, error) {
	if token == "" {
		return port.VerifiedSession{}, domain.ErrUnauthorized
	}
	claims, err := h.tokens.Parse(token)
	if err != nil {
		return port.VerifiedSession{}, err
	}

	session, err := h.sessions.GetByID(ctx, claims.SessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return port.VerifiedSession{}, domain.ErrUnauthorized
		}
		return port.VerifiedSession{}, err
	}
	if session.UserID != claims.UserID || !session.IsActive(h.now()) {
		return port.VerifiedSession{}, domain.ErrUnauthorized
	}

	return port.VerifiedSession{UserID: session.UserID, Email: session.Email}, nil
}

var _ port.VerifySessionUseCase = (*VerifySessionHandler)(nil)
