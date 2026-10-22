package commands_test

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"abet/auth-service/internal/application/commands"
	"abet/auth-service/internal/core/domain"
	"abet/auth-service/internal/core/port"
	"abet/auth-service/internal/core/port/mocks"
)

var _ = Describe("GoogleCallbackHandler", func() {
	var (
		ctrl     *gomock.Controller
		idp      *mocks.MockIdentityProvider
		main     *mocks.MockMainServiceClient
		sessions *mocks.MockSessionRepository
		tokens   *mocks.MockTokenService
		handler  *commands.GoogleCallbackHandler
		now      time.Time
		cmd      port.GoogleCallbackCommand
		userID   uuid.UUID
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		idp = mocks.NewMockIdentityProvider(ctrl)
		main = mocks.NewMockMainServiceClient(ctrl)
		sessions = mocks.NewMockSessionRepository(ctrl)
		tokens = mocks.NewMockTokenService(ctrl)

		policy, err := domain.NewEmailDomainPolicy([]string{"chula.ac.th"})
		Expect(err).NotTo(HaveOccurred())

		now = time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
		handler = commands.NewGoogleCallbackHandler(idp, main, sessions, tokens, policy, 12*time.Hour, func() time.Time { return now })

		userID = uuid.New()
		cmd = port.GoogleCallbackCommand{
			Code:      "auth-code",
			State:     "state-1",
			Stored:    domain.OAuthState{State: "state-1", Verifier: "verifier-1", Redirect: "/portfolios"},
			UserAgent: "test-agent",
			IP:        "10.0.0.1",
		}
	})

	AfterEach(func() { ctrl.Finish() })

	Context("state validation", func() {
		It("rejects a state mismatch without calling Google", func() {
			cmd.State = "other-state"
			_, err := handler.Execute(context.Background(), cmd)
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("rejects a missing code", func() {
			cmd.Code = ""
			_, err := handler.Execute(context.Background(), cmd)
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})
	})

	Context("identity checks", func() {
		It("rejects an email domain that is not allowed", func() {
			idp.EXPECT().Exchange(gomock.Any(), "auth-code", "verifier-1").
				Return(domain.Identity{Email: "mallory@gmail.com", EmailVerified: true}, nil)

			_, err := handler.Execute(context.Background(), cmd)
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("rejects a suffix-matching domain", func() {
			idp.EXPECT().Exchange(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(domain.Identity{Email: "mallory@evilchula.ac.th", EmailVerified: true}, nil)

			_, err := handler.Execute(context.Background(), cmd)
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("rejects an unverified email", func() {
			idp.EXPECT().Exchange(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(domain.Identity{Email: "alice@chula.ac.th", EmailVerified: false}, nil)

			_, err := handler.Execute(context.Background(), cmd)
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("propagates an exchange failure", func() {
			idp.EXPECT().Exchange(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(domain.Identity{}, domain.ErrUnauthorized)

			_, err := handler.Execute(context.Background(), cmd)
			Expect(err).To(MatchError(domain.ErrUnauthorized))
		})
	})

	Context("main-service login sync", func() {
		BeforeEach(func() {
			idp.EXPECT().Exchange(gomock.Any(), "auth-code", "verifier-1").
				Return(domain.Identity{Email: "Alice@Chula.ac.th", EmailVerified: true, Name: "Alice", PictureURL: "http://pic"}, nil)
		})

		It("maps a not-provisioned user to ErrForbidden and creates no session", func() {
			main.EXPECT().LoginSync(gomock.Any(), "Alice@Chula.ac.th", "Alice", "http://pic").
				Return(port.LoginSyncResult{}, domain.ErrForbidden)

			_, err := handler.Execute(context.Background(), cmd)
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("propagates main-service outages", func() {
			main.EXPECT().LoginSync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(port.LoginSyncResult{}, domain.ErrServiceUnavailable)

			_, err := handler.Execute(context.Background(), cmd)
			Expect(err).To(MatchError(domain.ErrServiceUnavailable))
		})

		It("creates a session and issues a token on success", func() {
			main.EXPECT().LoginSync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(port.LoginSyncResult{UserID: userID, Email: "alice@chula.ac.th", Role: "instructor"}, nil)

			var saved *domain.Session
			sessions.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, s *domain.Session) error {
					saved = s
					return nil
				})
			tokens.EXPECT().Issue(gomock.Any()).DoAndReturn(
				func(c domain.SessionClaims) (string, error) {
					Expect(c.SessionID).To(Equal(saved.ID))
					Expect(c.UserID).To(Equal(userID))
					Expect(c.Email).To(Equal("alice@chula.ac.th"))
					Expect(c.ExpiresAt).To(Equal(now.Add(12 * time.Hour)))
					return "signed-token", nil
				})

			result, err := handler.Execute(context.Background(), cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Token).To(Equal("signed-token"))
			Expect(result.MaxAge).To(Equal(12 * time.Hour))
			Expect(result.Redirect).To(Equal("/portfolios"))
			Expect(saved.UserAgent).To(Equal("test-agent"))
			Expect(saved.IP).To(Equal("10.0.0.1"))
		})

		It("falls back to / when the stored redirect is unsafe", func() {
			cmd.Stored.Redirect = "//evil.example"
			main.EXPECT().LoginSync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(port.LoginSyncResult{UserID: userID, Email: "alice@chula.ac.th"}, nil)
			sessions.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			tokens.EXPECT().Issue(gomock.Any()).Return("t", nil)

			result, err := handler.Execute(context.Background(), cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Redirect).To(Equal("/"))
		})

		It("fails when the session cannot be stored", func() {
			main.EXPECT().LoginSync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(port.LoginSyncResult{UserID: userID, Email: "alice@chula.ac.th"}, nil)
			sessions.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("db down"))

			_, err := handler.Execute(context.Background(), cmd)
			Expect(err).To(HaveOccurred())
		})
	})
})
