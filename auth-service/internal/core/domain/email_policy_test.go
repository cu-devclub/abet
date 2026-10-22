package domain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/auth-service/internal/core/domain"
)

var _ = Describe("EmailDomainPolicy", func() {
	It("requires at least one domain", func() {
		_, err := domain.NewEmailDomainPolicy([]string{" ", ""})
		Expect(err).To(MatchError(domain.ErrInvalidInput))
	})

	Context("with chula.ac.th and student.chula.ac.th", func() {
		var policy *domain.EmailDomainPolicy

		BeforeEach(func() {
			var err error
			policy, err = domain.NewEmailDomainPolicy([]string{"Chula.ac.th", " @student.chula.ac.th "})
			Expect(err).NotTo(HaveOccurred())
		})

		DescribeTable("Allows",
			func(email string, expected bool) {
				Expect(policy.Allows(email)).To(Equal(expected))
			},
			Entry("exact domain", "alice@chula.ac.th", true),
			Entry("second domain", "6634473123@student.chula.ac.th", true),
			Entry("case-insensitive", "Alice@CHULA.AC.TH", true),
			Entry("surrounding spaces", "  alice@chula.ac.th ", true),
			Entry("other domain", "alice@gmail.com", false),
			Entry("suffix attack", "alice@evilchula.ac.th", false),
			Entry("subdomain", "alice@eng.chula.ac.th", false),
			Entry("domain as local part", "chula.ac.th@gmail.com", false),
			Entry("trailing domain tricks", "alice@chula.ac.th.evil.com", false),
			Entry("no at sign", "chula.ac.th", false),
			Entry("empty local part", "@chula.ac.th", false),
			Entry("empty domain", "alice@", false),
			Entry("empty", "", false),
		)

		It("returns the first allowed domain lower-cased as primary", func() {
			Expect(policy.PrimaryDomain()).To(Equal("chula.ac.th"))
		})
	})
})
