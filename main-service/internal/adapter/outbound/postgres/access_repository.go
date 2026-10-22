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

type AccessRepository struct {
	store *Store
}

var _ port.AccessRepository = (*AccessRepository)(nil)

func NewAccessRepository(store *Store) *AccessRepository {
	return &AccessRepository{store: store}
}

func (r *AccessRepository) List(ctx context.Context, offeringID uuid.UUID) ([]domain.AccessEntry, error) {
	rows, err := r.store.q.ListOfferingAccess(ctx, offeringID)
	if err != nil {
		return nil, mapErr("list access", err)
	}
	out := make([]domain.AccessEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.AccessEntry{
			UserID: row.UserID, Name: row.Name, Email: row.Email, Role: domain.Role(row.Role),
			CanUpload: row.CanUpload, CanDownload: row.CanDownload, Source: row.Source,
		})
	}
	return out, nil
}

func (r *AccessRepository) ListDefaultStaff(ctx context.Context, offeringID, departmentID uuid.UUID) ([]port.DefaultStaff, error) {
	rows, err := r.store.q.ListDefaultStaffForOffering(ctx, sqlc.ListDefaultStaffForOfferingParams{DepartmentID: departmentID, OfferingID: offeringID})
	if err != nil {
		return nil, mapErr("list default staff", err)
	}
	out := make([]port.DefaultStaff, 0, len(rows))
	for _, row := range rows {
		out = append(out, port.DefaultStaff{UserID: row.UserID, Name: row.Name, Email: row.Email})
	}
	return out, nil
}

func (r *AccessRepository) Upsert(ctx context.Context, offeringID, userID uuid.UUID, canUpload, canDownload bool) error {
	err := r.store.q.UpsertAccess(ctx, sqlc.UpsertAccessParams{
		ID: uuid.New(), OfferingID: offeringID, UserID: userID, CanUpload: canUpload, CanDownload: canDownload,
		Source: domain.AccessSourceManual, At: time.Now().UTC(),
	})
	return mapErr("upsert access", err)
}

func (r *AccessRepository) Delete(ctx context.Context, offeringID, userID uuid.UUID) error {
	n, err := r.store.q.DeleteAccess(ctx, sqlc.DeleteAccessParams{OfferingID: offeringID, UserID: userID})
	if err != nil {
		return mapErr("delete access", err)
	}
	if n == 0 {
		return fmt.Errorf("delete access: %w", domain.ErrNotFound)
	}
	return nil
}
