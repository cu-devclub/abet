package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"abet/auth-service/internal/adapter/inbound/http/dto"
	"abet/auth-service/internal/core/domain"
	"abet/auth-service/internal/core/port"
)

const (
	// SessionCookieName is the session JWT cookie read by verify and set after login.
	SessionCookieName = "abet_session"
	// OAuthCookieName is the short-lived signed cookie holding state + PKCE verifier.
	OAuthCookieName = "abet_oauth"
	// OAuthCookiePath is the BROWSER path of the auth endpoints (the gateway strips /public).
	OAuthCookiePath = "/public/api/v1/auth"

	oauthCookieTTL = 10 * time.Minute
)

// AuthConfig holds the settings the auth handler needs.
type AuthConfig struct {
	PublicBaseURL string
	CookieSecure  bool
	SessionSecret string
}

// AuthHandler serves the Google login, callback and logout endpoints (public) and the forward-auth verify endpoint (internal).
type AuthHandler struct {
	start    port.StartGoogleLoginUseCase
	callback port.GoogleCallbackUseCase
	logout   port.LogoutUseCase
	verify   port.VerifySessionUseCase
	cfg      AuthConfig
	codec    oauthCookieCodec
}

// NewAuthHandler creates an AuthHandler; now may be nil (defaults to time.Now).
func NewAuthHandler(
	start port.StartGoogleLoginUseCase,
	callback port.GoogleCallbackUseCase,
	logout port.LogoutUseCase,
	verify port.VerifySessionUseCase,
	cfg AuthConfig,
	now func() time.Time,
) *AuthHandler {
	if now == nil {
		now = time.Now
	}
	return &AuthHandler{
		start:    start,
		callback: callback,
		logout:   logout,
		verify:   verify,
		cfg:      cfg,
		codec:    oauthCookieCodec{secret: []byte(cfg.SessionSecret), ttl: oauthCookieTTL, now: now},
	}
}

// Register mounts the auth routes on the /api/v1 group.
func (h *AuthHandler) Register(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	auth.GET("/google/login", h.GoogleLogin)
	auth.GET("/google/callback", h.GoogleCallback)
	auth.POST("/logout", h.Logout)
}

// RegisterInternal mounts the docker-network-only routes on the /internal/v1 group (never exposed by the gateway).
func (h *AuthHandler) RegisterInternal(rg *gin.RouterGroup) {
	rg.GET("/auth/verify", h.Verify)
}

// GoogleLogin starts the OAuth flow: stores state in a signed cookie and redirects to Google.
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	result, err := h.start.Execute(c.Request.Context(), c.Query("redirect"))
	if err != nil {
		writeError(c, err)
		return
	}

	value, err := h.codec.sign(result.State)
	if err != nil {
		writeError(c, err)
		return
	}
	h.setCookie(c, OAuthCookieName, value, OAuthCookiePath, int(oauthCookieTTL.Seconds()))
	c.Redirect(http.StatusFound, result.AuthURL)
}

// GoogleCallback finishes the OAuth flow and redirects the browser back into the app.
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	// The state cookie is single-use: always clear it.
	h.setCookie(c, OAuthCookieName, "", OAuthCookiePath, -1)

	cookieValue, err := c.Cookie(OAuthCookieName)
	if err != nil {
		h.redirectError(c, "login_failed")
		return
	}
	stored, err := h.codec.verify(cookieValue)
	if err != nil {
		h.redirectError(c, "login_failed")
		return
	}

	result, err := h.callback.Execute(c.Request.Context(), port.GoogleCallbackCommand{
		Code:      c.Query("code"),
		State:     c.Query("state"),
		Stored:    stored,
		UserAgent: truncate(c.Request.UserAgent(), 512),
		IP:        c.ClientIP(),
	})
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			slog.InfoContext(c.Request.Context(), "login denied", "reason", err.Error())
			h.redirectError(c, "access_denied")
			return
		}
		slog.WarnContext(c.Request.Context(), "login failed", "error", err)
		h.redirectError(c, "login_failed")
		return
	}

	h.setCookie(c, SessionCookieName, result.Token, "/", int(result.MaxAge.Seconds()))
	c.Redirect(http.StatusFound, h.cfg.PublicBaseURL+domain.SanitizeRedirect(result.Redirect))
}

// Logout revokes the session (when the cookie is valid), clears the cookie and returns 204.
func (h *AuthHandler) Logout(c *gin.Context) {
	if token, err := c.Cookie(SessionCookieName); err == nil {
		if err := h.logout.Execute(c.Request.Context(), token); err != nil {
			slog.ErrorContext(c.Request.Context(), "logout: revoke session failed", "error", err)
		}
	}
	h.setCookie(c, SessionCookieName, "", "/", -1)
	c.Status(http.StatusNoContent)
}

// Verify is the APISIX forward-auth endpoint: 200 + X-User-* headers, or 401.
func (h *AuthHandler) Verify(c *gin.Context) {
	unauthenticated := func() {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "unauthenticated"})
	}

	token, err := c.Cookie(SessionCookieName)
	if err != nil || token == "" {
		unauthenticated()
		return
	}

	session, err := h.verify.Execute(c.Request.Context(), token)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			unauthenticated()
			return
		}
		writeError(c, err)
		return
	}

	c.Header("X-User-Id", session.UserID.String())
	c.Header("X-User-Email", session.Email)
	c.Status(http.StatusOK)
	c.Writer.WriteHeaderNow()
}

func (h *AuthHandler) redirectError(c *gin.Context, code string) {
	c.Redirect(http.StatusFound, h.cfg.PublicBaseURL+"/login?error="+code)
}

func (h *AuthHandler) setCookie(c *gin.Context, name, value, path string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
