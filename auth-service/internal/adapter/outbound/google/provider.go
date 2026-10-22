package google

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"abet/auth-service/internal/core/domain"
	"abet/auth-service/internal/core/port"
)

const issuerURL = "https://accounts.google.com"

// googleEndpoint is the well-known Google endpoint pair; building the login URL therefore
// needs no network access. Exchange uses the endpoints from OIDC discovery.
var googleEndpoint = oauth2.Endpoint{
	AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
	TokenURL: "https://oauth2.googleapis.com/token",
}

// Provider implements port.IdentityProvider using Google OIDC (authorization code + PKCE S256).
// The OIDC discovery document is fetched lazily so the service starts without internet access.
type Provider struct {
	clientID     string
	clientSecret string
	redirectURL  string

	mu       sync.Mutex
	oidcProv *oidc.Provider
	client   *http.Client // optional (traced transport); used for discovery, token exchange and JWKS
}

func NewProvider(clientID, clientSecret, redirectURL string) *Provider {
	return &Provider{clientID: clientID, clientSecret: clientSecret, redirectURL: redirectURL}
}

// WithHTTPClient makes every outgoing call to Google (discovery, token, JWKS) use the given client.
func (p *Provider) WithHTTPClient(c *http.Client) *Provider {
	p.client = c
	return p
}

func (p *Provider) withClient(ctx context.Context) context.Context {
	if p.client == nil {
		return ctx
	}
	return oidc.ClientContext(ctx, p.client)
}

// Configured reports whether Google OAuth credentials were provided.
func (p *Provider) Configured() bool {
	return p.clientID != ""
}

func (p *Provider) discover(ctx context.Context) (*oidc.Provider, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.oidcProv != nil {
		return p.oidcProv, nil
	}
	prov, err := oidc.NewProvider(p.withClient(ctx), issuerURL)
	if err != nil {
		return nil, fmt.Errorf("%w: google oidc discovery: %v", domain.ErrServiceUnavailable, err)
	}
	p.oidcProv = prov
	return prov, nil
}

func (p *Provider) oauthConfig(endpoint oauth2.Endpoint) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		RedirectURL:  p.redirectURL,
		Endpoint:     endpoint,
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}
}

func (p *Provider) AuthCodeURL(state, verifier, hostedDomain string) (string, error) {
	if !p.Configured() {
		return "", domain.ErrOAuthNotConfigured
	}
	opts := []oauth2.AuthCodeOption{
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("prompt", "select_account"),
	}
	if hostedDomain != "" {
		opts = append(opts, oauth2.SetAuthURLParam("hd", hostedDomain))
	}
	return p.oauthConfig(googleEndpoint).AuthCodeURL(state, opts...), nil
}

func (p *Provider) Exchange(ctx context.Context, code, verifier string) (domain.Identity, error) {
	if !p.Configured() {
		return domain.Identity{}, domain.ErrOAuthNotConfigured
	}
	ctx = p.withClient(ctx)
	prov, err := p.discover(ctx)
	if err != nil {
		return domain.Identity{}, err
	}

	token, err := p.oauthConfig(prov.Endpoint()).Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return domain.Identity{}, fmt.Errorf("%w: exchange code: %v", domain.ErrUnauthorized, err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return domain.Identity{}, fmt.Errorf("%w: id_token missing in token response", domain.ErrUnauthorized)
	}

	// Verify validates signature, issuer, audience (= client id) and expiry.
	idToken, err := prov.Verifier(&oidc.Config{ClientID: p.clientID}).Verify(ctx, rawIDToken)
	if err != nil {
		return domain.Identity{}, fmt.Errorf("%w: verify id_token: %v", domain.ErrUnauthorized, err)
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return domain.Identity{}, fmt.Errorf("%w: decode id_token claims: %v", domain.ErrUnauthorized, err)
	}

	return domain.Identity{
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		Name:          claims.Name,
		PictureURL:    claims.Picture,
	}, nil
}

var _ port.IdentityProvider = (*Provider)(nil)
