package commands_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"abet/auth-service/internal/application/commands"
	"abet/auth-service/internal/core/domain"
	"abet/auth-service/internal/core/port/mocks"
)

var _ = Describe("StartGoogleLoginHandler", func() {
	var (
		ctrl    *gomock.Controller
		idp     *mocks.MockIdentityProvider
		handler *commands.StartGoogleLoginHandler
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		idp = mocks.NewMockIdentityProvider(ctrl)
		policy, err := domain.NewEmailDomainPolicy([]string{"chula.ac.th", "student.chula.ac.th"})
		Expect(err).NotTo(HaveOccurred())
		handler = commands.NewStartGoogleLoginHandler(idp, policy)
	})

	AfterEach(func() { ctrl.Finish() })

	It("creates state + verifier, sanitises the redirect and passes the first domain as hd", func() {
		idp.EXPECT().AuthCodeURL(gomock.Any(), gomock.Any(), "chula.ac.th").DoAndReturn(
			func(state, verifier, _ string) (string, error) {
				Expect(state).NotTo(BeEmpty())
				Expect(len(verifier)).To(BeNumerically(">=", 43))
				return "https://accounts.google.com/auth?state=" + state, nil
			})

		result, err := handler.Execute(context.Background(), "//evil.example")
		Expect(err).NotTo(HaveOccurred())
		Expect(result.State.Redirect).To(Equal("/"))
		Expect(result.AuthURL).To(ContainSubstring(result.State.State))
	})

	It("keeps a safe redirect", func() {
		idp.EXPECT().AuthCodeURL(gomock.Any(), gomock.Any(), gomock.Any()).Return("https://g", nil)

		result, err := handler.Execute(context.Background(), "/portfolios")
		Expect(err).NotTo(HaveOccurred())
		Expect(result.State.Redirect).To(Equal("/portfolios"))
	})

	It("returns ErrOAuthNotConfigured when Google is not configured", func() {
		idp.EXPECT().AuthCodeURL(gomock.Any(), gomock.Any(), gomock.Any()).Return("", domain.ErrOAuthNotConfigured)

		_, err := handler.Execute(context.Background(), "/")
		Expect(err).To(MatchError(domain.ErrOAuthNotConfigured))
	})
})
