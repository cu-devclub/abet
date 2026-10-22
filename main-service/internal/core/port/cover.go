package port

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
)

// ---- outbound ----

// CoverRepository stores the cover metadata columns of a curriculum (the PDF itself lives in Storage).
type CoverRepository interface {
	// Set records the cover; domain.ErrNotFound if the curriculum does not exist.
	Set(ctx context.Context, curriculumID uuid.UUID, filename string, sizeBytes int64, at time.Time) error
	// Clear removes the metadata; a curriculum without a cover is not an error.
	Clear(ctx context.Context, curriculumID uuid.UUID, at time.Time) error
}

// ---- inbound ----

type PutCoverInput struct {
	CurriculumID uuid.UUID
	Filename     string
	File         io.Reader
}

// CoverFile is an opened cover PDF; the caller must Close it.
type CoverFile struct {
	Cover domain.Cover
	File  io.ReadCloser
	Info  ObjectInfo
}

type PutCoverUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, in PutCoverInput) (*domain.Curriculum, error)
}

type GetCoverUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, curriculumID uuid.UUID) (*CoverFile, error)
}

type DeleteCoverUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, curriculumID uuid.UUID) error
}
