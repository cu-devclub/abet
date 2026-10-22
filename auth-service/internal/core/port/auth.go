package port

import (
	"context"
	"time"

	"github.com/google/uuid"

	"abet/auth-service/internal/core/domain"
)

// ---- Inbound (driving) ports ----

type StartLoginResult struct {
	AuthURL string
	State   domain.OAuthState
}

type StartGoogleLoginUseCase interface {
	Execute(ctx context.Context, redirect string) (StartLoginResult, error)
}

type GoogleCallbackCommand struct {
	Code      string
	State     string            // state query parameter returned by Google
	Stored    domain.OAuthState // state read back from the signed cookie
	UserAgent string
	IP        string
}

type GoogleCallbackResult struct {
	Token    string
	MaxAge   time.Duration
	Redirect string
}

type GoogleCallbackUseCase interface {
	Execute(ctx context.Context, cmd GoogleCallbackCommand) (GoogleCallbackResult, error)
}

type LogoutUseCase interface {
	Execute(ctx context.Context, token string) error
}

type VerifiedSession struct {
	UserID uuid.UUID
	Email  string
}

type VerifySessionUseCase interface {
	Execute(ctx context.Context, token string) (VerifiedSession, error)
}

type CleanupSessionsUseCase interface {
	Execute(ctx context.Context) (int64, error)
}

// ---- Outbound (driven) ports ----

type SessionRepository interface {
	Create(ctx context.Context, s *domain.Session) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Session, error)
	Revoke(ctx context.Context, id uuid.UUID, at time.Time) error
	DeleteExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// IdentityProvider hides the Google OAuth/OIDC exchange so it can be mocked.
type IdentityProvider interface {
	// AuthCodeURL builds the Google authorization URL (PKCE S256 + state). Returns
	// domain.ErrOAuthNotConfigured when no client id is configured.
	AuthCodeURL(state, verifier, hostedDomain string) (string, error)
	// Exchange trades the code for tokens and returns the verified ID-token identity.
	Exchange(ctx context.Context, code, verifier string) (domain.Identity, error)
}

type LoginSyncResult struct {
	UserID uuid.UUID
	Email  string
	Role   string
}

// MainServiceClient calls main-service's internal API.
type MainServiceClient interface {
	// LoginSync returns domain.ErrForbidden when the user is not provisioned.
	LoginSync(ctx context.Context, email, name, pictureURL string) (LoginSyncResult, error)
}

// TokenService issues and parses session JWTs.
type TokenService interface {
	Issue(claims domain.SessionClaims) (string, error)
	// Parse returns domain.ErrUnauthorized for malformed, badly signed or expired tokens.
	Parse(token string) (domain.SessionClaims, error)
}
