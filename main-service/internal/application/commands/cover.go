package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

const pdfMagic = "%PDF-"

// PutCoverHandler implements port.PutCoverUseCase: validates the PDF, stores it at covers/<id>.pdf and records the metadata.
type PutCoverHandler struct {
	currs    port.CurriculumRepository
	covers   port.CoverRepository
	storage  port.Storage
	maxBytes int64
}

var _ port.PutCoverUseCase = (*PutCoverHandler)(nil)

func NewPutCoverHandler(currs port.CurriculumRepository, covers port.CoverRepository, storage port.Storage, maxBytes int64) *PutCoverHandler {
	return &PutCoverHandler{currs: currs, covers: covers, storage: storage, maxBytes: maxBytes}
}

func (h *PutCoverHandler) Execute(ctx context.Context, actor *domain.Actor, in port.PutCoverInput) (*domain.Curriculum, error) {
	if !actor.CanManageCurriculum(in.CurriculumID) {
		return nil, domain.ErrForbidden
	}
	if _, err := h.currs.GetByID(ctx, in.CurriculumID); err != nil {
		return nil, notFoundAs(err, "curriculum not found")
	}

	// The whole file is read (bounded) before anything is stored, so a rejected upload never replaces a good cover.
	data, err := io.ReadAll(io.LimitReader(in.File, h.maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read cover: %w", err)
	}
	if int64(len(data)) > h.maxBytes {
		return nil, fmt.Errorf("%w: the cover is larger than %d MB", domain.ErrTooLarge, h.maxBytes>>20)
	}
	if !bytes.HasPrefix(data, []byte(pdfMagic)) {
		return nil, fmt.Errorf("%w: the cover must be a PDF file", domain.ErrInvalidInput)
	}

	size := int64(len(data))
	if err := h.storage.Put(ctx, CoverStorageKey(in.CurriculumID), bytes.NewReader(data), size, "application/pdf"); err != nil {
		return nil, fmt.Errorf("store cover: %w", err)
	}
	if err := h.covers.Set(ctx, in.CurriculumID, SanitizeCoverFilename(in.Filename), size, time.Now().UTC()); err != nil {
		return nil, notFoundAs(err, "curriculum not found")
	}
	return h.currs.GetByID(ctx, in.CurriculumID)
}

// SanitizeCoverFilename keeps the base name, drops control characters and quotes (it ends up in a header)
// and falls back to cover.pdf.
func SanitizeCoverFilename(name string) string {
	name = path.Base(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '"' || r == ';' {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == "/" {
		return "cover.pdf"
	}
	if len(name) > 200 {
		name = name[len(name)-200:]
	}
	return name
}

// GetCoverHandler implements port.GetCoverUseCase.
type GetCoverHandler struct {
	currs   port.CurriculumRepository
	storage port.Storage
}

var _ port.GetCoverUseCase = (*GetCoverHandler)(nil)

func NewGetCoverHandler(currs port.CurriculumRepository, storage port.Storage) *GetCoverHandler {
	return &GetCoverHandler{currs: currs, storage: storage}
}

func (h *GetCoverHandler) Execute(ctx context.Context, actor *domain.Actor, curriculumID uuid.UUID) (*port.CoverFile, error) {
	if !actor.CanManageCurriculum(curriculumID) {
		return nil, domain.ErrForbidden
	}
	cur, err := h.currs.GetByID(ctx, curriculumID)
	if err != nil {
		return nil, notFoundAs(err, "curriculum not found")
	}
	if cur.Cover == nil {
		return nil, fmt.Errorf("%w: this curriculum has no cover", domain.ErrNotFound)
	}
	f, info, err := h.storage.Get(ctx, CoverStorageKey(curriculumID))
	if err != nil {
		return nil, notFoundAs(err, "cover file not found")
	}
	return &port.CoverFile{Cover: *cur.Cover, File: f, Info: info}, nil
}

// DeleteCoverHandler implements port.DeleteCoverUseCase.
type DeleteCoverHandler struct {
	currs   port.CurriculumRepository
	covers  port.CoverRepository
	storage port.Storage
}

var _ port.DeleteCoverUseCase = (*DeleteCoverHandler)(nil)

func NewDeleteCoverHandler(currs port.CurriculumRepository, covers port.CoverRepository, storage port.Storage) *DeleteCoverHandler {
	return &DeleteCoverHandler{currs: currs, covers: covers, storage: storage}
}

func (h *DeleteCoverHandler) Execute(ctx context.Context, actor *domain.Actor, curriculumID uuid.UUID) error {
	if !actor.CanManageCurriculum(curriculumID) {
		return domain.ErrForbidden
	}
	if _, err := h.currs.GetByID(ctx, curriculumID); err != nil {
		return notFoundAs(err, "curriculum not found")
	}
	if err := h.covers.Clear(ctx, curriculumID, time.Now().UTC()); err != nil {
		return err
	}
	if err := h.storage.Delete(ctx, CoverStorageKey(curriculumID)); err != nil {
		slog.WarnContext(ctx, "delete cover file failed", "curriculum_id", curriculumID, "error", err)
	}
	return nil
}
