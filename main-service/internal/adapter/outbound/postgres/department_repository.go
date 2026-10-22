package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"abet/main-service/internal/adapter/outbound/postgres/sqlc"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

type DepartmentRepository struct {
	store *Store
}

var _ port.DepartmentRepository = (*DepartmentRepository)(nil)

func NewDepartmentRepository(store *Store) *DepartmentRepository {
	return &DepartmentRepository{store: store}
}

func (r *DepartmentRepository) Create(ctx context.Context, d *domain.Department) error {
	err := r.store.q.CreateDepartment(ctx, sqlc.CreateDepartmentParams{
		ID: d.ID, Code: d.Code, Name: d.Name, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	})
	return mapErr("create department", err)
}

func (r *DepartmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Department, error) {
	row, err := r.store.q.GetDepartment(ctx, id)
	if err != nil {
		return nil, mapErr("get department", err)
	}
	return &domain.Department{
		ID: row.ID, Code: row.Code, Name: row.Name, CurriculumCount: int(row.CurriculumCount),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *DepartmentRepository) List(ctx context.Context) ([]domain.Department, error) {
	rows, err := r.store.q.ListDepartments(ctx)
	if err != nil {
		return nil, mapErr("list departments", err)
	}
	out := make([]domain.Department, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Department{
			ID: row.ID, Code: row.Code, Name: row.Name, CurriculumCount: int(row.CurriculumCount),
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *DepartmentRepository) Update(ctx context.Context, d *domain.Department) error {
	n, err := r.store.q.UpdateDepartment(ctx, sqlc.UpdateDepartmentParams{
		ID: d.ID, Code: d.Code, Name: d.Name, UpdatedAt: d.UpdatedAt,
	})
	if err != nil {
		return mapErr("update department", err)
	}
	if n == 0 {
		return fmt.Errorf("update department: %w", domain.ErrNotFound)
	}
	return nil
}

func (r *DepartmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	n, err := r.store.q.DeleteDepartment(ctx, id)
	if err != nil {
		return mapErr("delete department", err)
	}
	if n == 0 {
		return fmt.Errorf("delete department: %w", domain.ErrNotFound)
	}
	return nil
}
