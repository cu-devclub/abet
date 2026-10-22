package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"abet/main-service/internal/adapter/outbound/postgres/sqlc"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

type CurriculumRepository struct {
	store *Store
}

var _ port.CurriculumRepository = (*CurriculumRepository)(nil)

func NewCurriculumRepository(store *Store) *CurriculumRepository {
	return &CurriculumRepository{store: store}
}

func (r *CurriculumRepository) Create(ctx context.Context, c *domain.Curriculum) error {
	err := r.store.q.CreateCurriculum(ctx, sqlc.CreateCurriculumParams{
		ID: c.ID, Code: c.Code, Name: c.Name, DepartmentID: c.DepartmentID, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	})
	return mapErr("create curriculum", err)
}

func (r *CurriculumRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Curriculum, error) {
	row, err := r.store.q.GetCurriculum(ctx, id)
	if err != nil {
		return nil, mapErr("get curriculum", err)
	}
	return curriculumFromRow(sqlc.ListCurriculaRow(row)), nil
}

func (r *CurriculumRepository) List(ctx context.Context, departmentID *uuid.UUID) ([]domain.Curriculum, error) {
	var filter uuid.NullUUID
	if departmentID != nil {
		filter = uuid.NullUUID{UUID: *departmentID, Valid: true}
	}
	rows, err := r.store.q.ListCurricula(ctx, filter)
	if err != nil {
		return nil, mapErr("list curricula", err)
	}
	out := make([]domain.Curriculum, 0, len(rows))
	for _, row := range rows {
		out = append(out, *curriculumFromRow(row))
	}
	return out, nil
}

func (r *CurriculumRepository) Update(ctx context.Context, c *domain.Curriculum) error {
	n, err := r.store.q.UpdateCurriculum(ctx, sqlc.UpdateCurriculumParams{
		ID: c.ID, Code: c.Code, Name: c.Name, DepartmentID: c.DepartmentID, UpdatedAt: c.UpdatedAt,
	})
	if err != nil {
		return mapErr("update curriculum", err)
	}
	if n == 0 {
		return fmt.Errorf("update curriculum: %w", domain.ErrNotFound)
	}
	return nil
}

func (r *CurriculumRepository) Delete(ctx context.Context, id uuid.UUID) error {
	n, err := r.store.q.DeleteCurriculum(ctx, id)
	if err != nil {
		return mapErr("delete curriculum", err)
	}
	if n == 0 {
		return fmt.Errorf("delete curriculum: %w", domain.ErrNotFound)
	}
	return nil
}

func curriculumFromRow(row sqlc.ListCurriculaRow) *domain.Curriculum {
	c := &domain.Curriculum{
		ID: row.ID, Code: row.Code, Name: row.Name, DepartmentID: row.DepartmentID,
		Department: domain.Ref{ID: row.DepartmentID, Code: row.DepartmentCode, Name: row.DepartmentName},
		CreatedAt:  row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if row.CoverFilename != nil {
		cover := &domain.Cover{Filename: *row.CoverFilename}
		if row.CoverSizeBytes != nil {
			cover.SizeBytes = *row.CoverSizeBytes
		}
		if row.CoverUpdatedAt != nil {
			cover.UpdatedAt = *row.CoverUpdatedAt
		}
		c.Cover = cover
	}
	return c
}
