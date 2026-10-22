package jwt_test

import (
	"strings"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/auth-service/internal/adapter/outbound/jwt"
	"abet/auth-service/internal/core/domain"
)

const testSecret = "0123456789abcdef0123456789abcdef-test-secret"

var _ = Describe("TokenService", func() {
	var (
		svc    *jwt.TokenService
		claims domain.SessionClaims
	)

	BeforeEach(func() {
		svc = jwt.NewTokenService(testSecret)
		now := time.Now().UTC().Truncate(time.Second)
		claims = domain.SessionClaims{
			UserID:    uuid.New(),
			Email:     "alice@chula.ac.th",
			SessionID: uuid.New(),
			IssuedAt:  now,
			ExpiresAt: now.Add(time.Hour),
		}
	})

	It("issues a token that parses back to the same claims", func() {
		token, err := svc.Issue(claims)
		Expect(err).NotTo(HaveOccurred())

		parsed, err := svc.Parse(token)
		Expect(err).NotTo(HaveOccurred())
		Expect(parsed.UserID).To(Equal(claims.UserID))
		Expect(parsed.SessionID).To(Equal(claims.SessionID))
		Expect(parsed.Email).To(Equal(claims.Email))
		Expect(parsed.ExpiresAt.Unix()).To(Equal(claims.ExpiresAt.Unix()))
		Expect(parsed.IssuedAt.Unix()).To(Equal(claims.IssuedAt.Unix()))
	})

	It("rejects an expired token", func() {
		claims.IssuedAt = time.Now().Add(-2 * time.Hour)
		claims.ExpiresAt = time.Now().Add(-time.Hour)
		token, err := svc.Issue(claims)
		Expect(err).NotTo(HaveOccurred())

		_, err = svc.Parse(token)
		Expect(err).To(MatchError(domain.ErrUnauthorized))
	})

	It("rejects a token signed with another secret", func() {
		token, err := jwt.NewTokenService("another-secret-another-secret-another!!").Issue(claims)
		Expect(err).NotTo(HaveOccurred())

		_, err = svc.Parse(token)
		Expect(err).To(MatchError(domain.ErrUnauthorized))
	})

	It("rejects a tampered token", func() {
		token, err := svc.Issue(claims)
		Expect(err).NotTo(HaveOccurred())
		parts := strings.Split(token, ".")
		parts[1] = parts[1][:len(parts[1])-2] + "AA"

		_, err = svc.Parse(strings.Join(parts, "."))
		Expect(err).To(MatchError(domain.ErrUnauthorized))
	})

	It("rejects the none algorithm", func() {
		unsigned, err := gojwt.NewWithClaims(gojwt.SigningMethodNone, gojwt.MapClaims{
			"sub": claims.UserID.String(), "sid": claims.SessionID.String(),
			"exp": time.Now().Add(time.Hour).Unix(),
		}).SignedString(gojwt.UnsafeAllowNoneSignatureType)
		Expect(err).NotTo(HaveOccurred())

		_, err = svc.Parse(unsigned)
		Expect(err).To(MatchError(domain.ErrUnauthorized))
	})

	It("rejects a token without expiry", func() {
		token, err := gojwt.NewWithClaims(gojwt.SigningMethodHS256, gojwt.MapClaims{
			"sub": claims.UserID.String(), "sid": claims.SessionID.String(),
		}).SignedString([]byte(testSecret))
		Expect(err).NotTo(HaveOccurred())

		_, err = svc.Parse(token)
		Expect(err).To(MatchError(domain.ErrUnauthorized))
	})

	It("rejects garbage", func() {
		_, err := svc.Parse("not-a-jwt")
		Expect(err).To(MatchError(domain.ErrUnauthorized))
	})
})
