package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"abet/main-service/internal/adapter/outbound/postgres/sqlc"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

type CoverRepository struct {
	store *Store
}

var _ port.CoverRepository = (*CoverRepository)(nil)

func NewCoverRepository(store *Store) *CoverRepository {
	return &CoverRepository{store: store}
}

func (r *CoverRepository) Set(ctx context.Context, curriculumID uuid.UUID, filename string, sizeBytes int64, at time.Time) error {
	n, err := r.store.q.SetCurriculumCover(ctx, sqlc.SetCurriculumCoverParams{
		ID: curriculumID, CoverFilename: &filename, CoverSizeBytes: &sizeBytes, CoverUpdatedAt: &at,
	})
	if err != nil {
		return mapErr("set cover", err)
	}
	if n == 0 {
		return fmt.Errorf("set cover: %w", domain.ErrNotFound)
	}
	return nil
}

func (r *CoverRepository) Clear(ctx context.Context, curriculumID uuid.UUID, at time.Time) error {
	_, err := r.store.q.ClearCurriculumCover(ctx, sqlc.ClearCurriculumCoverParams{ID: curriculumID, UpdatedAt: at})
	return mapErr("clear cover", err)
}
