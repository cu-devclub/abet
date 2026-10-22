package jwt

import (
	"fmt"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"abet/auth-service/internal/core/domain"
	"abet/auth-service/internal/core/port"
)

// TokenService issues and parses HS256 session JWTs with claims {sub, email, sid, iat, exp}.
type TokenService struct {
	secret []byte
}

func NewTokenService(secret string) *TokenService {
	return &TokenService{secret: []byte(secret)}
}

type sessionClaims struct {
	Email string `json:"email"`
	SID   string `json:"sid"`
	gojwt.RegisteredClaims
}

func (s *TokenService) Issue(c domain.SessionClaims) (string, error) {
	claims := sessionClaims{
		Email: c.Email,
		SID:   c.SessionID.String(),
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   c.UserID.String(),
			IssuedAt:  gojwt.NewNumericDate(c.IssuedAt),
			ExpiresAt: gojwt.NewNumericDate(c.ExpiresAt),
		},
	}
	signed, err := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("%w: sign session token: %v", domain.ErrServiceUnavailable, err)
	}
	return signed, nil
}

func (s *TokenService) Parse(token string) (domain.SessionClaims, error) {
	var claims sessionClaims
	parsed, err := gojwt.ParseWithClaims(token, &claims, func(*gojwt.Token) (any, error) {
		return s.secret, nil
	}, gojwt.WithValidMethods([]string{gojwt.SigningMethodHS256.Alg()}), gojwt.WithExpirationRequired())
	if err != nil {
		return domain.SessionClaims{}, fmt.Errorf("%w: %v", domain.ErrUnauthorized, err)
	}
	if !parsed.Valid {
		return domain.SessionClaims{}, fmt.Errorf("%w: invalid token", domain.ErrUnauthorized)
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return domain.SessionClaims{}, fmt.Errorf("%w: invalid subject", domain.ErrUnauthorized)
	}
	sid, err := uuid.Parse(claims.SID)
	if err != nil {
		return domain.SessionClaims{}, fmt.Errorf("%w: invalid session id", domain.ErrUnauthorized)
	}

	out := domain.SessionClaims{UserID: userID, Email: claims.Email, SessionID: sid}
	if claims.IssuedAt != nil {
		out.IssuedAt = claims.IssuedAt.Time
	}
	if claims.ExpiresAt != nil {
		out.ExpiresAt = claims.ExpiresAt.Time
	}
	return out, nil
}

var _ port.TokenService = (*TokenService)(nil)
