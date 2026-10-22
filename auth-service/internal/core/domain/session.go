package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Session is a server-side login session; its ID is the JWT "sid" claim.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Email     string
	UserAgent string
	IP        string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// NewSession validates the input and creates a session valid for ttl from now.
func NewSession(userID uuid.UUID, email, userAgent, ip string, ttl time.Duration, now time.Time) (*Session, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("%w: user id cannot be empty", ErrInvalidInput)
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, fmt.Errorf("%w: email cannot be empty", ErrInvalidInput)
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("%w: session ttl must be positive", ErrInvalidInput)
	}
	now = now.UTC()
	return &Session{
		ID:        uuid.New(),
		UserID:    userID,
		Email:     email,
		UserAgent: userAgent,
		IP:        ip,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}, nil
}

// IsActive reports whether the session is neither revoked nor expired at the given time.
func (s *Session) IsActive(now time.Time) bool {
	return s.RevokedAt == nil && now.Before(s.ExpiresAt)
}

// SessionClaims are the claims carried by the session JWT (sub, email, sid, iat, exp).
type SessionClaims struct {
	UserID    uuid.UUID
	Email     string
	SessionID uuid.UUID
	IssuedAt  time.Time
	ExpiresAt time.Time
}
