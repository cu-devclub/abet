package domain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/main-service/internal/core/domain"
)

var _ = Describe("email domain helpers", func() {
	allowed := []string{"chula.ac.th"}

	DescribeTable("IsAllowedEmailDomain",
		func(email string, want bool) {
			Expect(domain.IsAllowedEmailDomain(email, allowed)).To(Equal(want))
		},
		Entry("exact domain", "alice@chula.ac.th", true),
		Entry("case-insensitive", "Alice@CHULA.AC.TH", true),
		Entry("padded with spaces", "  alice@chula.ac.th ", true),
		Entry("sub-domain is not an exact match", "alice@student.chula.ac.th", false),
		Entry("suffix trick", "alice@evilchula.ac.th", false),
		Entry("other domain", "alice@gmail.com", false),
		Entry("no at sign", "alice", false),
		Entry("empty local part", "@chula.ac.th", false),
		Entry("empty domain", "alice@", false),
		Entry("domain in local part", "chula.ac.th@gmail.com", false),
	)

	It("supports several allowed domains", func() {
		Expect(domain.IsAllowedEmailDomain("a@student.chula.ac.th", []string{"chula.ac.th", "Student.Chula.ac.th"})).To(BeTrue())
	})

	It("ValidateEmail lower-cases and trims", func() {
		got, err := domain.ValidateEmail("  Alice@Chula.ac.th ", allowed)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("alice@chula.ac.th"))
	})

	It("ValidateEmail rejects bad syntax and foreign domains with ErrInvalidInput", func() {
		_, err := domain.ValidateEmail("not-an-email", allowed)
		Expect(err).To(MatchError(domain.ErrInvalidInput))
		_, err = domain.ValidateEmail("a@gmail.com", allowed)
		Expect(err).To(MatchError(domain.ErrInvalidInput))
		_, err = domain.ValidateEmail("Alice <a@chula.ac.th>", allowed)
		Expect(err).To(MatchError(domain.ErrInvalidInput))
	})
})
