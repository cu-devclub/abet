package postgres

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/exaring/otelpgx"
	"github.com/golang-migrate/migrate/v4"
	pgxv5 "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"abet/auth-service/config"
	migrations "abet/auth-service/db"
)

// poolConfig builds the pgx config from separate parameters (the password is never put into a URL/DSN,
// so any character is safe).
func poolConfig(c config.DBConfig) (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, fmt.Errorf("init pgx config: %w", err)
	}
	cfg.ConnConfig.Host = c.Host
	cfg.ConnConfig.Port = c.Port
	cfg.ConnConfig.User = c.User
	cfg.ConnConfig.Password = c.Password
	cfg.ConnConfig.Database = c.Name
	cfg.ConnConfig.Tracer = otelpgx.NewTracer(otelpgx.WithTrimSQLInSpanName()) // no SQL parameter values are recorded

	switch c.SSLMode {
	case "disable", "":
		cfg.ConnConfig.TLSConfig = nil
		cfg.ConnConfig.Fallbacks = nil
	case "require", "verify-ca", "verify-full", "allow", "prefer":
		tlsCfg := &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}
		if c.SSLMode != "verify-full" {
			// require/verify-ca/prefer/allow do not verify the host name (libpq semantics, certificates are not verified either for require)
			tlsCfg.InsecureSkipVerify = true //nolint:gosec // mirrors libpq sslmode semantics
		}
		cfg.ConnConfig.TLSConfig = tlsCfg
		cfg.ConnConfig.Fallbacks = nil
	default:
		return nil, fmt.Errorf("unsupported sslmode %q", c.SSLMode)
	}
	return cfg, nil
}

// Connect opens a pgx connection pool and verifies connectivity.
func Connect(ctx context.Context, c config.DBConfig) (*pgxpool.Pool, error) {
	cfg, err := poolConfig(c)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

// Migrate applies all embedded migrations (idempotent).
func Migrate(c config.DBConfig) error {
	cfg, err := poolConfig(c)
	if err != nil {
		return err
	}
	db := stdlib.OpenDB(*cfg.ConnConfig)
	defer db.Close()

	src, err := iofs.New(migrations.FS, migrations.Dir)
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	driver, err := pgxv5.WithInstance(db, &pgxv5.Config{})
	if err != nil {
		return fmt.Errorf("init migrate driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
