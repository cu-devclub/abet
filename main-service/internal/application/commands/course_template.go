package commands

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"sort"
	"strings"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
	"abet/main-service/internal/core/validation"
)

// PutCourseTemplateHandler implements port.PutCourseTemplateUseCase.
type PutCourseTemplateHandler struct {
	storage  port.Storage
	tmpDir   string
	maxBytes int64
}

var _ port.PutCourseTemplateUseCase = (*PutCourseTemplateHandler)(nil)

func NewPutCourseTemplateHandler(storage port.Storage, tmpDir string, maxBytes int64) *PutCourseTemplateHandler {
	return &PutCourseTemplateHandler{storage: storage, tmpDir: tmpDir, maxBytes: maxBytes}
}

func (h *PutCourseTemplateHandler) Execute(ctx context.Context, actor *domain.Actor, file io.Reader) (*port.CourseTemplateInfo, error) {
	if !actor.CanManageSystem() {
		return nil, domain.ErrForbidden
	}
	tmp, err := os.CreateTemp(h.tmpDir, "course-template-*.zip")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = tmp.Close(); _ = os.Remove(tmp.Name()) }()
	n, err := io.Copy(tmp, io.LimitReader(file, h.maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read template: %w", err)
	}
	if n > h.maxBytes {
		return nil, fmt.Errorf("%w: file too large (max %d MB)", domain.ErrTooLarge, h.maxBytes>>20)
	}
	if err := checkCourseTemplateZip(tmp, n); err != nil {
		return nil, err
	}
	meta := map[string]string{port.CourseTemplateMetaUploader: actor.Email}
	if err := h.storage.PutFileMeta(ctx, port.CourseTemplateKey, tmp.Name(), "application/zip", meta); err != nil {
		return nil, fmt.Errorf("store template: %w", err)
	}
	return statCourseTemplate(ctx, h.storage)
}

// checkCourseTemplateZip rejects non-zips, unsafe entry paths and zips without the four core CSV templates.
func checkCourseTemplateZip(r io.ReaderAt, size int64) error {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return fmt.Errorf("%w: the file is not a valid zip archive", domain.ErrInvalidInput)
	}
	names := map[string]bool{}
	tops := map[string]bool{"": true}
	for _, f := range zr.File {
		n := f.Name
		if strings.HasPrefix(n, "/") || strings.Contains(n, "\\") || slices.Contains(strings.Split(n, "/"), "..") {
			return fmt.Errorf("%w: the zip contains an unsafe path (%q)", domain.ErrInvalidInput, n)
		}
		names[n] = true
		if top, _, ok := strings.Cut(n, "/"); ok {
			tops[top+"/"] = true
		}
	}
	// The CSVs sit at the zip root or under one top folder: report the prefix with the fewest missing files.
	var best []string
	found := false
	for prefix := range tops {
		var missing []string
		for _, p := range validation.CoreCSVPaths() {
			if !names[prefix+p] {
				missing = append(missing, path.Join(strings.TrimSuffix(prefix, "/"), p))
			}
		}
		sort.Strings(missing)
		if !found || len(missing) < len(best) || (len(missing) == len(best) && strings.Join(missing, ",") < strings.Join(best, ",")) {
			best, found = missing, true
		}
	}
	if len(best) > 0 {
		return fmt.Errorf("%w: the zip is missing required files: %s", domain.ErrInvalidInput, strings.Join(best, ", "))
	}
	return nil
}

// GetCourseTemplateHandler implements port.GetCourseTemplateUseCase.
type GetCourseTemplateHandler struct{ storage port.Storage }

var _ port.GetCourseTemplateUseCase = (*GetCourseTemplateHandler)(nil)

func NewGetCourseTemplateHandler(storage port.Storage) *GetCourseTemplateHandler {
	return &GetCourseTemplateHandler{storage: storage}
}

func (h *GetCourseTemplateHandler) Execute(ctx context.Context, actor *domain.Actor) (*port.CourseTemplateInfo, error) {
	if !actor.CanManageSystem() {
		return nil, domain.ErrForbidden
	}
	return statCourseTemplate(ctx, h.storage)
}

// DeleteCourseTemplateHandler implements port.DeleteCourseTemplateUseCase.
type DeleteCourseTemplateHandler struct{ storage port.Storage }

var _ port.DeleteCourseTemplateUseCase = (*DeleteCourseTemplateHandler)(nil)

func NewDeleteCourseTemplateHandler(storage port.Storage) *DeleteCourseTemplateHandler {
	return &DeleteCourseTemplateHandler{storage: storage}
}

func (h *DeleteCourseTemplateHandler) Execute(ctx context.Context, actor *domain.Actor) (*port.CourseTemplateInfo, error) {
	if !actor.CanManageSystem() {
		return nil, domain.ErrForbidden
	}
	if err := h.storage.Delete(ctx, port.CourseTemplateKey); err != nil {
		return nil, fmt.Errorf("delete template: %w", err)
	}
	return &port.CourseTemplateInfo{Source: "default"}, nil
}

func statCourseTemplate(ctx context.Context, s port.Storage) (*port.CourseTemplateInfo, error) {
	st, err := s.Stat(ctx, port.CourseTemplateKey)
	if errors.Is(err, domain.ErrNotFound) {
		return &port.CourseTemplateInfo{Source: "default"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat template: %w", err)
	}
	info := &port.CourseTemplateInfo{Source: "custom", SizeBytes: &st.Size, UploadedAt: &st.LastModified}
	for k, v := range st.Meta { // the S3 server canonicalizes the key case
		if strings.EqualFold(k, port.CourseTemplateMetaUploader) {
			by := v
			info.UploadedBy = &by
		}
	}
	return info, nil
}
