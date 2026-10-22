// Package postgres implements the repository ports on top of pgx + sqlc.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	pgxv5 "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"abet/main-service/db"
	"abet/main-service/internal/adapter/outbound/postgres/sqlc"
	"abet/main-service/internal/core/domain"
)

// Store owns the connection pool and the sqlc Queries. Every repository takes a *Store.
type Store struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

// NewStore connects to Postgres and verifies the connection.
func NewStore(ctx context.Context, dbc DBConfig) (*Store, error) {
	cfg, err := dbc.PoolConfig()
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Store{pool: pool, q: sqlc.New(pool)}, nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

// Ping implements port.Pinger (used by /readyz).
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// Queries returns the pool-backed queries (single statements, no transaction).
func (s *Store) Queries() *sqlc.Queries { return s.q }

// WithTx runs fn inside a transaction; it commits when fn returns nil and rolls back otherwise.
// Reusable by every module that needs several statements to be atomic.
func (s *Store) WithTx(ctx context.Context, fn func(q *sqlc.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if err := fn(s.q.WithTx(tx)); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// Migrate applies the embedded golang-migrate migrations (idempotent). The connection is opened from the structured
// settings (no URL), so the password never needs escaping.
func Migrate(dbc DBConfig) error {
	cfg, err := dbc.PoolConfig()
	if err != nil {
		return err
	}
	sqlDB := stdlib.OpenDB(*cfg.ConnConfig)
	defer func() { _ = sqlDB.Close() }()

	driver, err := pgxv5.WithInstance(sqlDB, &pgxv5.Config{})
	if err != nil {
		return fmt.Errorf("init migrate driver: %w", err)
	}
	src, err := iofs.New(db.FS, db.MigrationsDir)
	if err != nil {
		return fmt.Errorf("open migrations: %w", err)
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

// mapErr translates pgx/PostgreSQL errors into domain sentinels; unknown errors are wrapped with context.
func mapErr(op string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", op, domain.ErrNotFound)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return fmt.Errorf("%s: %w: duplicate value", op, domain.ErrConflict)
		case "23503": // foreign_key_violation
			return fmt.Errorf("%s: %w: row is referenced or reference is missing", op, domain.ErrConflict)
		case "23514", "22P02": // check_violation, invalid_text_representation
			return fmt.Errorf("%s: %w: %s", op, domain.ErrInvalidInput, pgErr.Message)
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
