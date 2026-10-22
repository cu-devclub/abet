package handler

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"abet/main-service/internal/adapter/inbound/http/dto"
	"abet/main-service/internal/adapter/inbound/http/middleware"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// multipartOverhead is allowed on top of MAX_UPLOAD_MB for the multipart framing of the single file field.
const multipartOverhead = 1 << 20

// SubmissionHandler serves the Course Portfolios endpoints: uploads, history, PDF and bulk downloads, the portfolio list.
type SubmissionHandler struct {
	upload     port.UploadSubmissionUseCase
	get        port.GetSubmissionUseCase
	list       port.ListSubmissionsUseCase
	pdf        port.GetPortfolioPDFUseCase
	portfolios port.ListPortfoliosUseCase
	download   port.DownloadPortfoliosUseCase
	maxUpload  int64 // bytes
}

func NewSubmissionHandler(
	upload port.UploadSubmissionUseCase,
	get port.GetSubmissionUseCase,
	list port.ListSubmissionsUseCase,
	pdf port.GetPortfolioPDFUseCase,
	portfolios port.ListPortfoliosUseCase,
	download port.DownloadPortfoliosUseCase,
	maxUploadMB int,
) *SubmissionHandler {
	return &SubmissionHandler{
		upload: upload, get: get, list: list, pdf: pdf, portfolios: portfolios, download: download,
		maxUpload: int64(maxUploadMB) << 20,
	}
}

// Register mounts the routes on the authenticated /api/v1 group. Gin accepts the distinct sub-routes of /offerings/:id
// next to the offering handler's own ones.
func (h *SubmissionHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/portfolios", h.ListPortfolios)
	rg.POST("/portfolios/download", h.Download)
	rg.POST("/offerings/:id/submissions", h.Upload)
	rg.GET("/offerings/:id/submissions", h.List)
	rg.GET("/offerings/:id/portfolio.pdf", h.PortfolioPDF)
	rg.GET("/submissions/:id", h.Get)
}

func (h *SubmissionHandler) ListPortfolios(c *gin.Context) {
	var f port.PortfolioFilter
	if raw := c.Query("academic_year"); raw != "" {
		y, err := strconv.Atoi(raw)
		if err != nil {
			badRequest(c, "invalid academic_year")
			return
		}
		f.AcademicYear = &y
	}
	if raw := c.Query("semester"); raw != "" {
		s, err := strconv.Atoi(raw)
		if err != nil || s < 1 || s > 3 {
			badRequest(c, "invalid semester")
			return
		}
		f.Semester = &s
	}
	var ok bool
	if f.DepartmentID, ok = optionalUUIDQuery(c, "department_id"); !ok {
		return
	}
	if f.CurriculumID, ok = optionalUUIDQuery(c, "curriculum_id"); !ok {
		return
	}
	f.Search = strings.TrimSpace(c.Query("search"))

	rows, err := h.portfolios.Execute(c.Request.Context(), middleware.ActorFrom(c), f)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Items(dto.PortfolioRowsFromDomain(rows)))
}

func optionalUUIDQuery(c *gin.Context, name string) (*uuid.UUID, bool) {
	raw := c.Query(name)
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		badRequest(c, "invalid "+name)
		return nil, false
	}
	return &id, true
}

// Upload streams the multipart "file" part into a temp file; the use case validates it and only then stores it in the bucket.
func (h *SubmissionHandler) Upload(c *gin.Context) {
	offeringID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	limit := h.maxUpload + multipartOverhead
	if c.Request.ContentLength > limit {
		writeError(c, h.tooLarge())
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	mr, err := c.Request.MultipartReader()
	if err != nil {
		badRequest(c, "expected a multipart/form-data body with a file field")
		return
	}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			badRequest(c, "file is required")
			return
		}
		if err != nil {
			h.writeFormError(c, err)
			return
		}
		if part.FormName() != "file" || part.FileName() == "" {
			_, _ = io.Copy(io.Discard, part)
			continue
		}
		sub, err := h.upload.Execute(c.Request.Context(), middleware.ActorFrom(c), port.UploadSubmissionInput{
			OfferingID: offeringID,
			Filename:   part.FileName(),
			Body:       &sizeLimitReader{r: part, limit: h.maxUpload, tooLarge: h.tooLarge},
		})
		if err != nil {
			var mbe *http.MaxBytesError
			if errors.As(err, &mbe) {
				err = h.tooLarge()
			}
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, dto.SubmissionFromDomain(sub, true))
		return
	}
}

func (h *SubmissionHandler) tooLarge() error {
	return fmt.Errorf("%w: file exceeds %d MB limit", domain.ErrTooLarge, h.maxUpload>>20)
}

func (h *SubmissionHandler) writeFormError(c *gin.Context, err error) {
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		writeError(c, h.tooLarge())
		return
	}
	badRequest(c, "invalid multipart form")
}

// sizeLimitReader fails with the 413 error as soon as more than limit bytes of the file part were read.
type sizeLimitReader struct {
	r        io.Reader
	limit    int64
	read     int64
	tooLarge func() error
}

func (s *sizeLimitReader) Read(p []byte) (int, error) {
	n, err := s.r.Read(p)
	s.read += int64(n)
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) || s.read > s.limit {
		return n, s.tooLarge()
	}
	return n, err
}

func (h *SubmissionHandler) Get(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	sub, err := h.get.Execute(c.Request.Context(), middleware.ActorFrom(c), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SubmissionFromDomain(sub, true))
}

func (h *SubmissionHandler) List(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	items, err := h.list.Execute(c.Request.Context(), middleware.ActorFrom(c), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Items(dto.SubmissionsFromDomain(items)))
}

func (h *SubmissionHandler) PortfolioPDF(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	f, err := h.pdf.Execute(c.Request.Context(), middleware.ActorFrom(c), id)
	if err != nil {
		writeError(c, err)
		return
	}
	defer func() { _ = f.File.Close() }()
	hdr := c.Writer.Header()
	hdr.Set("Content-Type", "application/pdf")
	hdr.Set("Content-Disposition", contentDisposition("attachment", f.Filename))
	hdr.Set("Cache-Control", "no-cache")
	hdr.Set("X-Content-Type-Options", "nosniff")
	streamObject(c, f.File, f.Info)
}

// Download answers with a zip streamed directly from storage. All checks run before the first byte is written.
func (h *SubmissionHandler) Download(c *gin.Context) {
	var req dto.DownloadPortfoliosRequest
	if !bindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	plan, err := h.download.Plan(ctx, middleware.ActorFrom(c), req.OfferingIDs)
	if err != nil {
		writeError(c, err)
		return
	}
	hdr := c.Writer.Header()
	hdr.Set("Content-Type", "application/zip")
	hdr.Set("Content-Disposition", contentDisposition("attachment", plan.ZipName))
	hdr.Set("Cache-Control", "no-store")
	c.Status(http.StatusOK)
	if err := h.download.Write(ctx, plan, c.Writer); err != nil {
		// Headers are gone already; the client sees a truncated archive.
		slog.ErrorContext(c.Request.Context(), "portfolio download aborted", "error", err)
	}
}
