package queries_test

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"abet/auth-service/internal/application/queries"
	"abet/auth-service/internal/core/domain"
	"abet/auth-service/internal/core/port/mocks"
)

var _ = Describe("VerifySessionHandler", func() {
	var (
		ctrl     *gomock.Controller
		tokens   *mocks.MockTokenService
		sessions *mocks.MockSessionRepository
		handler  *queries.VerifySessionHandler
		now      time.Time
		claims   domain.SessionClaims
		session  *domain.Session
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		tokens = mocks.NewMockTokenService(ctrl)
		sessions = mocks.NewMockSessionRepository(ctrl)
		now = time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
		handler = queries.NewVerifySessionHandler(tokens, sessions, func() time.Time { return now })

		userID, sid := uuid.New(), uuid.New()
		claims = domain.SessionClaims{UserID: userID, Email: "alice@chula.ac.th", SessionID: sid}
		session = &domain.Session{
			ID: sid, UserID: userID, Email: "alice@chula.ac.th",
			CreatedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour),
		}
	})

	AfterEach(func() { ctrl.Finish() })

	It("returns the user for an active session", func() {
		tokens.EXPECT().Parse("tok").Return(claims, nil)
		sessions.EXPECT().GetByID(gomock.Any(), claims.SessionID).Return(session, nil)

		result, err := handler.Execute(context.Background(), "tok")
		Expect(err).NotTo(HaveOccurred())
		Expect(result.UserID).To(Equal(claims.UserID))
		Expect(result.Email).To(Equal("alice@chula.ac.th"))
	})

	It("rejects an empty token", func() {
		_, err := handler.Execute(context.Background(), "")
		Expect(err).To(MatchError(domain.ErrUnauthorized))
	})

	It("rejects an invalid token", func() {
		tokens.EXPECT().Parse("bad").Return(domain.SessionClaims{}, domain.ErrUnauthorized)

		_, err := handler.Execute(context.Background(), "bad")
		Expect(err).To(MatchError(domain.ErrUnauthorized))
	})

	It("rejects a revoked session", func() {
		revoked := now.Add(-time.Minute)
		session.RevokedAt = &revoked
		tokens.EXPECT().Parse("tok").Return(claims, nil)
		sessions.EXPECT().GetByID(gomock.Any(), claims.SessionID).Return(session, nil)

		_, err := handler.Execute(context.Background(), "tok")
		Expect(err).To(MatchError(domain.ErrUnauthorized))
	})

	It("rejects an expired session row", func() {
		session.ExpiresAt = now.Add(-time.Second)
		tokens.EXPECT().Parse("tok").Return(claims, nil)
		sessions.EXPECT().GetByID(gomock.Any(), claims.SessionID).Return(session, nil)

		_, err := handler.Execute(context.Background(), "tok")
		Expect(err).To(MatchError(domain.ErrUnauthorized))
	})

	It("rejects an unknown session", func() {
		tokens.EXPECT().Parse("tok").Return(claims, nil)
		sessions.EXPECT().GetByID(gomock.Any(), claims.SessionID).Return(nil, domain.ErrNotFound)

		_, err := handler.Execute(context.Background(), "tok")
		Expect(err).To(MatchError(domain.ErrUnauthorized))
	})

	It("rejects a session that belongs to another user", func() {
		session.UserID = uuid.New()
		tokens.EXPECT().Parse("tok").Return(claims, nil)
		sessions.EXPECT().GetByID(gomock.Any(), claims.SessionID).Return(session, nil)

		_, err := handler.Execute(context.Background(), "tok")
		Expect(err).To(MatchError(domain.ErrUnauthorized))
	})

	It("propagates repository failures (not 401)", func() {
		boom := errors.New("db down")
		tokens.EXPECT().Parse("tok").Return(claims, nil)
		sessions.EXPECT().GetByID(gomock.Any(), claims.SessionID).Return(nil, boom)

		_, err := handler.Execute(context.Background(), "tok")
		Expect(err).To(MatchError(boom))
	})
})
