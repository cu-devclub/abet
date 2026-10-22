package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"abet/auth-service/internal/core/domain"
)

// oauthCookieCodec signs and verifies the short-lived OAuth state cookie value:
// base64url(json payload) + "." + base64url(HMAC-SHA256(payload, SESSION_SECRET)).
type oauthCookieCodec struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

type oauthCookiePayload struct {
	State    string `json:"s"`
	Verifier string `json:"v"`
	Redirect string `json:"r"`
	Expires  int64  `json:"e"`
}

var errInvalidOAuthCookie = errors.New("invalid oauth cookie")

func (c oauthCookieCodec) sign(st domain.OAuthState) (string, error) {
	raw, err := json.Marshal(oauthCookiePayload{
		State:    st.State,
		Verifier: st.Verifier,
		Redirect: st.Redirect,
		Expires:  c.now().Add(c.ttl).Unix(),
	})
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	return payload + "." + base64.RawURLEncoding.EncodeToString(c.mac(payload)), nil
}

func (c oauthCookieCodec) verify(value string) (domain.OAuthState, error) {
	payload, sig, ok := strings.Cut(value, ".")
	if !ok {
		return domain.OAuthState{}, errInvalidOAuthCookie
	}
	gotSig, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || !hmac.Equal(gotSig, c.mac(payload)) {
		return domain.OAuthState{}, errInvalidOAuthCookie
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return domain.OAuthState{}, errInvalidOAuthCookie
	}
	var p oauthCookiePayload
	if err := json.Unmarshal(raw, &p); err != nil || p.State == "" || p.Verifier == "" {
		return domain.OAuthState{}, errInvalidOAuthCookie
	}
	if c.now().Unix() > p.Expires {
		return domain.OAuthState{}, errInvalidOAuthCookie
	}
	return domain.OAuthState{State: p.State, Verifier: p.Verifier, Redirect: p.Redirect}, nil
}

func (c oauthCookieCodec) mac(payload string) []byte {
	h := hmac.New(sha256.New, c.secret)
	h.Write([]byte(payload))
	return h.Sum(nil)
}
