package domain

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// OAuthState is the per-login data kept in the short-lived signed cookie between
// the login redirect and the Google callback.
type OAuthState struct {
	State    string
	Verifier string // PKCE code verifier
	Redirect string // sanitised post-login path
}

// NewOAuthState generates a random state and PKCE verifier for the given (unsanitised) redirect.
func NewOAuthState(redirect string) (OAuthState, error) {
	state, err := randomToken(32)
	if err != nil {
		return OAuthState{}, err
	}
	verifier, err := randomToken(48)
	if err != nil {
		return OAuthState{}, err
	}
	return OAuthState{State: state, Verifier: verifier, Redirect: SanitizeRedirect(redirect)}, nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("%w: generate random token: %v", ErrServiceUnavailable, err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
