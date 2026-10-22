package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"abet/auth-service/internal/adapter/outbound/postgres/sqlc"
	"abet/auth-service/internal/core/domain"
	"abet/auth-service/internal/core/port"
)

// SessionRepository stores sessions in Postgres using sqlc generated queries.
type SessionRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool, queries: sqlc.New(pool)}
}

// Ping checks database connectivity (used by /readyz).
func (r *SessionRepository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *SessionRepository) Create(ctx context.Context, s *domain.Session) error {
	err := r.queries.CreateSession(ctx, sqlc.CreateSessionParams{
		ID:        pgtype.UUID{Bytes: s.ID, Valid: true},
		UserID:    pgtype.UUID{Bytes: s.UserID, Valid: true},
		Email:     s.Email,
		UserAgent: s.UserAgent,
		Ip:        s.IP,
		CreatedAt: pgtype.Timestamptz{Time: s.CreatedAt, Valid: true},
		ExpiresAt: pgtype.Timestamptz{Time: s.ExpiresAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("%w: create session: %v", domain.ErrServiceUnavailable, err)
	}
	return nil
}

func (r *SessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	row, err := r.queries.GetSession(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("%w: get session: %v", domain.ErrServiceUnavailable, err)
	}
	s := &domain.Session{
		ID:        uuid.UUID(row.ID.Bytes),
		UserID:    uuid.UUID(row.UserID.Bytes),
		Email:     row.Email,
		UserAgent: row.UserAgent,
		IP:        row.Ip,
		CreatedAt: row.CreatedAt.Time,
		ExpiresAt: row.ExpiresAt.Time,
	}
	if row.RevokedAt.Valid {
		t := row.RevokedAt.Time
		s.RevokedAt = &t
	}
	return s, nil
}

func (r *SessionRepository) Revoke(ctx context.Context, id uuid.UUID, at time.Time) error {
	err := r.queries.RevokeSession(ctx, sqlc.RevokeSessionParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		RevokedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("%w: revoke session: %v", domain.ErrServiceUnavailable, err)
	}
	return nil
}

func (r *SessionRepository) DeleteExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	n, err := r.queries.DeleteExpiredSessions(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("%w: delete expired sessions: %v", domain.ErrServiceUnavailable, err)
	}
	return n, nil
}

var _ port.SessionRepository = (*SessionRepository)(nil)
