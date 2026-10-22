package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"abet/main-service/internal/adapter/inbound/http/dto"
	"abet/main-service/internal/adapter/inbound/http/middleware"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// ImportHandler serves the mapping CSV import (preview + apply). The file is read into memory and never stored.
type ImportHandler struct {
	preview  port.PreviewImportUseCase
	apply    port.ApplyImportUseCase
	maxBytes int64
}

func NewImportHandler(preview port.PreviewImportUseCase, apply port.ApplyImportUseCase, maxBytes int64) *ImportHandler {
	return &ImportHandler{preview: preview, apply: apply, maxBytes: maxBytes}
}

func (h *ImportHandler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/offerings/import")
	g.POST("/preview", h.Preview)
	g.POST("/apply", h.Apply)
}

func (h *ImportHandler) Preview(c *gin.Context) {
	in, ok := readImportForm(c, h.maxBytes)
	if !ok {
		return
	}
	plan, err := h.preview.Execute(c.Request.Context(), middleware.ActorFrom(c), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ImportPlanFromDomain(plan))
}

func (h *ImportHandler) Apply(c *gin.Context) {
	in, ok := readImportForm(c, h.maxBytes)
	if !ok {
		return
	}
	plan, err := h.apply.Execute(c.Request.Context(), middleware.ActorFrom(c), in)
	if err != nil {
		if plan != nil && errors.Is(err, domain.ErrUnprocessable) {
			c.JSON(http.StatusUnprocessableEntity, dto.ImportPlanFromDomain(plan))
			return
		}
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ImportPlanFromDomain(plan))
}

// readImportForm parses the multipart body (file, curriculum_id, academic_year); on failure it answers and returns false.
func readImportForm(c *gin.Context, maxBytes int64) (port.ImportInput, bool) {
	var in port.ImportInput
	// Allow 1 MB of multipart overhead on top of the file limit (MAX_IMPORT_MB).
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes+1<<20)
	if err := c.Request.ParseMultipartForm(8 << 20); err != nil {
		writeUploadFormError(c, err)
		return in, false
	}
	defer func() { _ = c.Request.MultipartForm.RemoveAll() }()

	curriculumID, err := uuid.Parse(c.PostForm("curriculum_id"))
	if err != nil {
		badRequest(c, "invalid curriculum_id")
		return in, false
	}
	year, err := strconv.Atoi(c.PostForm("academic_year"))
	if err != nil {
		badRequest(c, "invalid academic_year")
		return in, false
	}
	fh, err := c.FormFile("file")
	if err != nil {
		badRequest(c, "file is required")
		return in, false
	}
	if fh.Size > maxBytes {
		writeError(c, domain.ErrTooLarge)
		return in, false
	}
	f, err := fh.Open()
	if err != nil {
		badRequest(c, "cannot read file")
		return in, false
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		badRequest(c, "cannot read file")
		return in, false
	}
	return port.ImportInput{CurriculumID: curriculumID, AcademicYear: year, Data: data}, true
}

// writeUploadFormError answers a multipart parsing failure: 413 when the body limit was hit, else 400.
func writeUploadFormError(c *gin.Context, err error) {
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		writeError(c, domain.ErrTooLarge)
		return
	}
	badRequest(c, "invalid multipart form")
}
