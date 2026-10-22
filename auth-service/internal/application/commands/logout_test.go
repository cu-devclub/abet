package commands_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"abet/auth-service/internal/application/commands"
	"abet/auth-service/internal/core/domain"
	"abet/auth-service/internal/core/port/mocks"
)

var _ = Describe("LogoutHandler", func() {
	var (
		ctrl     *gomock.Controller
		tokens   *mocks.MockTokenService
		sessions *mocks.MockSessionRepository
		handler  *commands.LogoutHandler
		now      time.Time
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		tokens = mocks.NewMockTokenService(ctrl)
		sessions = mocks.NewMockSessionRepository(ctrl)
		now = time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
		handler = commands.NewLogoutHandler(tokens, sessions, func() time.Time { return now })
	})

	AfterEach(func() { ctrl.Finish() })

	It("revokes the session of a valid token", func() {
		sid := uuid.New()
		tokens.EXPECT().Parse("tok").Return(domain.SessionClaims{SessionID: sid}, nil)
		sessions.EXPECT().Revoke(gomock.Any(), sid, now).Return(nil)

		Expect(handler.Execute(context.Background(), "tok")).To(Succeed())
	})

	It("does nothing for an invalid token", func() {
		tokens.EXPECT().Parse("bad").Return(domain.SessionClaims{}, domain.ErrUnauthorized)

		Expect(handler.Execute(context.Background(), "bad")).To(Succeed())
	})

	It("does nothing for an empty token", func() {
		Expect(handler.Execute(context.Background(), "")).To(Succeed())
	})
})

var _ = Describe("CleanupSessionsHandler", func() {
	It("deletes sessions expired more than 7 days ago", func() {
		ctrl := gomock.NewController(GinkgoT())
		defer ctrl.Finish()
		sessions := mocks.NewMockSessionRepository(ctrl)
		now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
		handler := commands.NewCleanupSessionsHandler(sessions, func() time.Time { return now })

		sessions.EXPECT().DeleteExpiredBefore(gomock.Any(), now.Add(-7*24*time.Hour)).Return(int64(3), nil)

		deleted, err := handler.Execute(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(deleted).To(Equal(int64(3)))
	})
})
