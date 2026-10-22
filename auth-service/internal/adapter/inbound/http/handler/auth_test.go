package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"abet/auth-service/internal/adapter/inbound/http/handler"
	"abet/auth-service/internal/core/domain"
	"abet/auth-service/internal/core/port"
	"abet/auth-service/internal/core/port/mocks"
)

var _ = Describe("AuthHandler", func() {
	var (
		ctrl     *gomock.Controller
		start    *mocks.MockStartGoogleLoginUseCase
		callback *mocks.MockGoogleCallbackUseCase
		logout   *mocks.MockLogoutUseCase
		verify   *mocks.MockVerifySessionUseCase
		router   *gin.Engine
	)

	const publicBase = "http://localhost:8000"

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		ctrl = gomock.NewController(GinkgoT())
		start = mocks.NewMockStartGoogleLoginUseCase(ctrl)
		callback = mocks.NewMockGoogleCallbackUseCase(ctrl)
		logout = mocks.NewMockLogoutUseCase(ctrl)
		verify = mocks.NewMockVerifySessionUseCase(ctrl)

		h := handler.NewAuthHandler(start, callback, logout, verify, handler.AuthConfig{
			PublicBaseURL: publicBase,
			CookieSecure:  false,
			SessionSecret: "0123456789abcdef0123456789abcdef",
		}, nil)
		router = gin.New()
		h.Register(router.Group("/api/v1"))
		h.RegisterInternal(router.Group("/internal/v1"))
	})

	AfterEach(func() { ctrl.Finish() })

	do := func(method, target string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	cookieFrom := func(rec *httptest.ResponseRecorder, name string) *http.Cookie {
		for _, c := range rec.Result().Cookies() {
			if c.Name == name {
				return c
			}
		}
		return nil
	}

	Describe("GET /google/login", func() {
		It("sets the signed state cookie on the browser path and redirects to Google", func() {
			start.EXPECT().Execute(gomock.Any(), "/portfolios").Return(port.StartLoginResult{
				AuthURL: "https://accounts.google.com/o/oauth2/v2/auth?x=1",
				State:   domain.OAuthState{State: "s1", Verifier: "v1", Redirect: "/portfolios"},
			}, nil)

			rec := do(http.MethodGet, "/api/v1/auth/google/login?redirect=/portfolios")
			Expect(rec.Code).To(Equal(http.StatusFound))
			Expect(rec.Header().Get("Location")).To(Equal("https://accounts.google.com/o/oauth2/v2/auth?x=1"))

			c := cookieFrom(rec, "abet_oauth")
			Expect(c).NotTo(BeNil())
			Expect(c.Path).To(Equal("/public/api/v1/auth"))
			Expect(c.HttpOnly).To(BeTrue())
			Expect(c.SameSite).To(Equal(http.SameSiteLaxMode))
			Expect(c.MaxAge).To(Equal(600))
		})

		It("returns 503 google oauth not configured", func() {
			start.EXPECT().Execute(gomock.Any(), "").Return(port.StartLoginResult{}, domain.ErrOAuthNotConfigured)

			rec := do(http.MethodGet, "/api/v1/auth/google/login")
			Expect(rec.Code).To(Equal(http.StatusServiceUnavailable))
			Expect(rec.Body.String()).To(MatchJSON(`{"error":"google oauth not configured"}`))
		})
	})

	Describe("GET /google/callback", func() {
		// obtainStateCookie runs the login step to get a valid signed cookie.
		obtainStateCookie := func() *http.Cookie {
			start.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(port.StartLoginResult{
				AuthURL: "https://g",
				State:   domain.OAuthState{State: "s1", Verifier: "v1", Redirect: "/portfolios"},
			}, nil)
			return cookieFrom(do(http.MethodGet, "/api/v1/auth/google/login"), "abet_oauth")
		}

		It("redirects to login_failed without the state cookie", func() {
			rec := do(http.MethodGet, "/api/v1/auth/google/callback?code=c&state=s1")
			Expect(rec.Header().Get("Location")).To(Equal(publicBase + "/login?error=login_failed"))
		})

		It("redirects to login_failed for a forged state cookie", func() {
			rec := do(http.MethodGet, "/api/v1/auth/google/callback?code=c&state=s1",
				&http.Cookie{Name: "abet_oauth", Value: "eyJzIjoiczEifQ.AAAA"})
			Expect(rec.Header().Get("Location")).To(Equal(publicBase + "/login?error=login_failed"))
		})

		It("sets the session cookie and redirects to the requested path on success", func() {
			stateCookie := obtainStateCookie()
			callback.EXPECT().Execute(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, cmd port.GoogleCallbackCommand) (port.GoogleCallbackResult, error) {
					Expect(cmd.Code).To(Equal("c"))
					Expect(cmd.State).To(Equal("s1"))
					Expect(cmd.Stored).To(Equal(domain.OAuthState{State: "s1", Verifier: "v1", Redirect: "/portfolios"}))
					return port.GoogleCallbackResult{Token: "jwt", MaxAge: 12 * time.Hour, Redirect: "/portfolios"}, nil
				})

			rec := do(http.MethodGet, "/api/v1/auth/google/callback?code=c&state=s1", stateCookie)
			Expect(rec.Code).To(Equal(http.StatusFound))
			Expect(rec.Header().Get("Location")).To(Equal(publicBase + "/portfolios"))

			session := cookieFrom(rec, "abet_session")
			Expect(session).NotTo(BeNil())
			Expect(session.Value).To(Equal("jwt"))
			Expect(session.Path).To(Equal("/"))
			Expect(session.MaxAge).To(Equal(12 * 3600))
			Expect(session.HttpOnly).To(BeTrue())
			Expect(session.SameSite).To(Equal(http.SameSiteLaxMode))

			cleared := cookieFrom(rec, "abet_oauth")
			Expect(cleared).NotTo(BeNil())
			Expect(cleared.MaxAge).To(BeNumerically("<", 0))
		})

		It("redirects to access_denied when the user is rejected", func() {
			stateCookie := obtainStateCookie()
			callback.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(port.GoogleCallbackResult{}, domain.ErrForbidden)

			rec := do(http.MethodGet, "/api/v1/auth/google/callback?code=c&state=s1", stateCookie)
			Expect(rec.Header().Get("Location")).To(Equal(publicBase + "/login?error=access_denied"))
			Expect(cookieFrom(rec, "abet_session")).To(BeNil())
		})

		It("redirects to login_failed on other errors", func() {
			stateCookie := obtainStateCookie()
			callback.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(port.GoogleCallbackResult{}, domain.ErrServiceUnavailable)

			rec := do(http.MethodGet, "/api/v1/auth/google/callback?code=c&state=s1", stateCookie)
			Expect(rec.Header().Get("Location")).To(Equal(publicBase + "/login?error=login_failed"))
		})
	})

	Describe("POST /logout", func() {
		It("revokes and clears the cookie, always 204", func() {
			logout.EXPECT().Execute(gomock.Any(), "jwt").Return(nil)

			rec := do(http.MethodPost, "/api/v1/auth/logout", &http.Cookie{Name: "abet_session", Value: "jwt"})
			Expect(rec.Code).To(Equal(http.StatusNoContent))
			c := cookieFrom(rec, "abet_session")
			Expect(c).NotTo(BeNil())
			Expect(c.MaxAge).To(BeNumerically("<", 0))
		})

		It("returns 204 without a cookie", func() {
			rec := do(http.MethodPost, "/api/v1/auth/logout")
			Expect(rec.Code).To(Equal(http.StatusNoContent))
		})
	})

	Describe("GET /verify", func() {
		It("returns 200 with X-User-* headers for a valid session", func() {
			uid := uuid.New()
			verify.EXPECT().Execute(gomock.Any(), "jwt").Return(port.VerifiedSession{UserID: uid, Email: "alice@chula.ac.th"}, nil)

			rec := do(http.MethodGet, "/internal/v1/auth/verify", &http.Cookie{Name: "abet_session", Value: "jwt"})
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Header().Get("X-User-Id")).To(Equal(uid.String()))
			Expect(rec.Header().Get("X-User-Email")).To(Equal("alice@chula.ac.th"))
			Expect(rec.Body.String()).To(BeEmpty())
		})

		It("returns 401 without cookie", func() {
			rec := do(http.MethodGet, "/internal/v1/auth/verify")
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
			Expect(rec.Body.String()).To(MatchJSON(`{"error":"unauthenticated"}`))
		})

		It("returns 401 for an invalid session", func() {
			verify.EXPECT().Execute(gomock.Any(), "bad").Return(port.VerifiedSession{}, domain.ErrUnauthorized)

			rec := do(http.MethodGet, "/internal/v1/auth/verify", &http.Cookie{Name: "abet_session", Value: "bad"})
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
			Expect(rec.Header().Get("X-User-Id")).To(BeEmpty())
		})

		It("returns 503 when the repository is down", func() {
			verify.EXPECT().Execute(gomock.Any(), "jwt").Return(port.VerifiedSession{}, domain.ErrServiceUnavailable)

			rec := do(http.MethodGet, "/internal/v1/auth/verify", &http.Cookie{Name: "abet_session", Value: "jwt"})
			Expect(rec.Code).To(Equal(http.StatusServiceUnavailable))
		})
	})
})
