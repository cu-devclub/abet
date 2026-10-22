package commands

import (
	"context"
	"crypto/subtle"
	"fmt"
	"time"

	"abet/auth-service/internal/core/domain"
	"abet/auth-service/internal/core/port"
)

// GoogleCallbackHandler completes the Google login: validates state, verifies the identity,
// checks the email domain, syncs with main-service and opens a session.
type GoogleCallbackHandler struct {
	idp      port.IdentityProvider
	main     port.MainServiceClient
	sessions port.SessionRepository
	tokens   port.TokenService
	policy   *domain.EmailDomainPolicy
	ttl      time.Duration
	now      func() time.Time
}

// NewGoogleCallbackHandler creates the handler; now may be nil (defaults to time.Now).
func NewGoogleCallbackHandler(
	idp port.IdentityProvider,
	main port.MainServiceClient,
	sessions port.SessionRepository,
	tokens port.TokenService,
	policy *domain.EmailDomainPolicy,
	ttl time.Duration,
	now func() time.Time,
) *GoogleCallbackHandler {
	if now == nil {
		now = time.Now
	}
	return &GoogleCallbackHandler{idp: idp, main: main, sessions: sessions, tokens: tokens, policy: policy, ttl: ttl, now: now}
}

func (h *GoogleCallbackHandler) Execute(ctx context.Context, cmd port.GoogleCallbackCommand) (port.GoogleCallbackResult, error) {
	if cmd.Code == "" || cmd.State == "" || cmd.Stored.State == "" ||
		subtle.ConstantTimeCompare([]byte(cmd.State), []byte(cmd.Stored.State)) != 1 {
		return port.GoogleCallbackResult{}, fmt.Errorf("%w: invalid oauth state", domain.ErrInvalidInput)
	}

	identity, err := h.idp.Exchange(ctx, cmd.Code, cmd.Stored.Verifier)
	if err != nil {
		return port.GoogleCallbackResult{}, err
	}

	if !identity.EmailVerified {
		return port.GoogleCallbackResult{}, fmt.Errorf("%w: email is not verified", domain.ErrForbidden)
	}
	if !h.policy.Allows(identity.Email) {
		return port.GoogleCallbackResult{}, fmt.Errorf("%w: email domain is not allowed", domain.ErrForbidden)
	}

	synced, err := h.main.LoginSync(ctx, identity.Email, identity.Name, identity.PictureURL)
	if err != nil {
		return port.GoogleCallbackResult{}, err
	}

	now := h.now()
	session, err := domain.NewSession(synced.UserID, synced.Email, cmd.UserAgent, cmd.IP, h.ttl, now)
	if err != nil {
		return port.GoogleCallbackResult{}, err
	}
	if err := h.sessions.Create(ctx, session); err != nil {
		return port.GoogleCallbackResult{}, err
	}

	token, err := h.tokens.Issue(domain.SessionClaims{
		UserID:    session.UserID,
		Email:     session.Email,
		SessionID: session.ID,
		IssuedAt:  session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
	})
	if err != nil {
		return port.GoogleCallbackResult{}, err
	}

	return port.GoogleCallbackResult{
		Token:    token,
		MaxAge:   h.ttl,
		Redirect: domain.SanitizeRedirect(cmd.Stored.Redirect),
	}, nil
}

var _ port.GoogleCallbackUseCase = (*GoogleCallbackHandler)(nil)
