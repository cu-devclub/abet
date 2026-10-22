package commands

import (
	"context"

	"abet/auth-service/internal/core/domain"
	"abet/auth-service/internal/core/port"
)

// StartGoogleLoginHandler creates the OAuth state/PKCE verifier and the Google authorization URL.
type StartGoogleLoginHandler struct {
	idp    port.IdentityProvider
	policy *domain.EmailDomainPolicy
}

func NewStartGoogleLoginHandler(idp port.IdentityProvider, policy *domain.EmailDomainPolicy) *StartGoogleLoginHandler {
	return &StartGoogleLoginHandler{idp: idp, policy: policy}
}

func (h *StartGoogleLoginHandler) Execute(_ context.Context, redirect string) (port.StartLoginResult, error) {
	state, err := domain.NewOAuthState(redirect)
	if err != nil {
		return port.StartLoginResult{}, err
	}
	authURL, err := h.idp.AuthCodeURL(state.State, state.Verifier, h.policy.PrimaryDomain())
	if err != nil {
		return port.StartLoginResult{}, err
	}
	return port.StartLoginResult{AuthURL: authURL, State: state}, nil
}

var _ port.StartGoogleLoginUseCase = (*StartGoogleLoginHandler)(nil)
