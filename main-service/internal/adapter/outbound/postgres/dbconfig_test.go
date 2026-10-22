package postgres_test

import (
	"crypto/tls"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/main-service/internal/adapter/outbound/postgres"
)

var _ = Describe("DBConfig", func() {
	const nasty = `p@ss:w/rd?#%&='"\ x`

	It("keeps every field verbatim, whatever characters the password has", func() {
		cfg, err := postgres.DBConfig{Host: "postgres", Port: "5433", User: "abet", Password: nasty, Name: "main_db"}.PoolConfig()
		Expect(err).NotTo(HaveOccurred())
		cc := cfg.ConnConfig
		Expect(cc.Host).To(Equal("postgres"))
		Expect(cc.Port).To(BeEquivalentTo(5433))
		Expect(cc.User).To(Equal("abet"))
		Expect(cc.Password).To(Equal(nasty))
		Expect(cc.Database).To(Equal("main_db"))
	})

	It("quotes non-secret values that contain spaces, quotes and backslashes", func() {
		cfg, err := postgres.DBConfig{Host: "h", Port: "5432", User: `we ird'\user`, Password: "x", Name: `db name'`}.PoolConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.ConnConfig.User).To(Equal(`we ird'\user`))
		Expect(cfg.ConnConfig.Database).To(Equal("db name'"))
	})

	It("defaults sslmode to disable (no TLS) and honours require", func() {
		cfg, err := postgres.DBConfig{Host: "h", Port: "5432", User: "u", Password: "p", Name: "d"}.PoolConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.ConnConfig.TLSConfig).To(BeNil())

		cfg, err = postgres.DBConfig{Host: "h", Port: "5432", User: "u", Password: "p", Name: "d", SSLMode: "require"}.PoolConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.ConnConfig.TLSConfig).To(BeAssignableToTypeOf(&tls.Config{}))
		Expect(cfg.ConnConfig.TLSConfig).NotTo(BeNil())
	})

	It("rejects an unknown sslmode without echoing the password", func() {
		_, err := postgres.DBConfig{Host: "h", Port: "5432", User: "u", Password: nasty, Name: "d", SSLMode: "bogus"}.PoolConfig()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).NotTo(ContainSubstring(nasty))
	})
})
