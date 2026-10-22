package postgres

import (
	"testing"

	"abet/auth-service/config"
)

func TestPoolConfigKeepsSpecialPasswordVerbatim(t *testing.T) {
	pw := `p@ss:w/rd?#%&='"\ x`
	cfg, err := poolConfig(config.DBConfig{Host: "db", Port: 5433, User: "u@x", Password: pw, Name: "auth_db", SSLMode: "disable"})
	if err != nil {
		t.Fatal(err)
	}
	cc := cfg.ConnConfig
	if cc.Host != "db" || cc.Port != 5433 || cc.User != "u@x" || cc.Password != pw || cc.Database != "auth_db" {
		t.Fatalf("unexpected conn config: %+v", cc)
	}
	if cc.TLSConfig != nil {
		t.Fatal("sslmode=disable must not use TLS")
	}
}

func TestPoolConfigSSLModes(t *testing.T) {
	base := config.DBConfig{Host: "db", Port: 5432, User: "u", Name: "n"}
	base.SSLMode = "require"
	cfg, err := poolConfig(base)
	if err != nil || cfg.ConnConfig.TLSConfig == nil {
		t.Fatalf("require: err=%v tls=%v", err, cfg)
	}
	base.SSLMode = "bogus"
	if _, err := poolConfig(base); err == nil {
		t.Fatal("expected error for unknown sslmode")
	}
}
