package domain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/auth-service/internal/core/domain"
)

var _ = Describe("SanitizeRedirect", func() {
	DescribeTable("returns a safe relative path",
		func(input, expected string) {
			Expect(domain.SanitizeRedirect(input)).To(Equal(expected))
		},
		Entry("empty", "", "/"),
		Entry("root", "/", "/"),
		Entry("simple path", "/portfolios", "/portfolios"),
		Entry("path with query", "/portfolios?year=2026&x=1", "/portfolios?year=2026&x=1"),
		Entry("absolute url", "https://evil.example/", "/"),
		Entry("scheme-relative url", "//evil.example/path", "/"),
		Entry("backslash trick", "/\\evil.example", "/"),
		Entry("backslash anywhere", "/a\\b", "/"),
		Entry("no leading slash", "portfolios", "/"),
		Entry("javascript scheme", "javascript:alert(1)", "/"),
		Entry("newline injection", "/ok\r\nSet-Cookie: x=1", "/"),
		Entry("tab character", "/a\tb", "/"),
	)
})
