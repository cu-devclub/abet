package handler

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"abet/main-service/internal/adapter/inbound/http/dto"
	"abet/main-service/internal/adapter/inbound/http/middleware"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// CoverHandler serves the curriculum cover PDF (upload, preview/download, delete).
type CoverHandler struct {
	put      port.PutCoverUseCase
	get      port.GetCoverUseCase
	del      port.DeleteCoverUseCase
	maxBytes int64
}

func NewCoverHandler(put port.PutCoverUseCase, get port.GetCoverUseCase, del port.DeleteCoverUseCase, maxBytes int64) *CoverHandler {
	return &CoverHandler{put: put, get: get, del: del, maxBytes: maxBytes}
}

func (h *CoverHandler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/curricula")
	g.PUT("/:id/cover", h.Put)
	g.GET("/:id/cover", h.Get)
	g.DELETE("/:id/cover", h.Delete)
}

func (h *CoverHandler) Put(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	// 1 MB of multipart overhead on top of the file limit; the use case enforces the exact limit.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxBytes+1<<20)
	if err := c.Request.ParseMultipartForm(8 << 20); err != nil {
		writeUploadFormError(c, err)
		return
	}
	defer func() { _ = c.Request.MultipartForm.RemoveAll() }()
	fh, err := c.FormFile("file")
	if err != nil {
		badRequest(c, "file is required")
		return
	}
	if fh.Size > h.maxBytes {
		writeError(c, fmt.Errorf("%w: the cover is larger than %d MB", domain.ErrTooLarge, h.maxBytes>>20))
		return
	}
	f, err := fh.Open()
	if err != nil {
		badRequest(c, "cannot read file")
		return
	}
	defer func() { _ = f.Close() }()

	cur, err := h.put.Execute(c.Request.Context(), middleware.ActorFrom(c), port.PutCoverInput{CurriculumID: id, Filename: fh.Filename, File: f})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.CurriculumFromDomain(cur))
}

// Get streams the PDF. Query parameters other than download are ignored (the frontend appends ?v=<timestamp>).
// X-Frame-Options is SAMEORIGIN on purpose: the frontend previews the cover in an iframe.
func (h *CoverHandler) Get(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	cf, err := h.get.Execute(c.Request.Context(), middleware.ActorFrom(c), id)
	if err != nil {
		writeError(c, err)
		return
	}
	defer func() { _ = cf.File.Close() }()

	kind := "inline"
	if d := c.Query("download"); d == "1" || strings.EqualFold(d, "true") {
		kind = "attachment"
	}
	hdr := c.Writer.Header()
	hdr.Set("Content-Type", "application/pdf")
	hdr.Set("Content-Disposition", contentDisposition(kind, cf.Cover.Filename))
	hdr.Set("Cache-Control", "no-cache")
	hdr.Set("X-Frame-Options", "SAMEORIGIN")
	hdr.Set("X-Content-Type-Options", "nosniff")
	streamObject(c, cf.File, cf.Info)
}

func (h *CoverHandler) Delete(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.del.Execute(c.Request.Context(), middleware.ActorFrom(c), id); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// contentDisposition builds `inline; filename="x.pdf"`; non-ASCII names get an ASCII fallback plus filename*.
func contentDisposition(kind, name string) string {
	ascii := strings.Map(func(r rune) rune {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, name)
	out := fmt.Sprintf(`%s; filename="%s"`, kind, ascii)
	if ascii != name {
		out += "; filename*=UTF-8''" + url.PathEscape(name)
	}
	return out
}

// streamObject copies an object to the response with its Content-Length (objects are not seekable: no Range support).
func streamObject(c *gin.Context, r io.Reader, info port.ObjectInfo) {
	hdr := c.Writer.Header()
	hdr.Set("Content-Length", strconv.FormatInt(info.Size, 10))
	if !info.LastModified.IsZero() {
		hdr.Set("Last-Modified", info.LastModified.UTC().Format(http.TimeFormat))
	}
	c.Status(http.StatusOK)
	if c.Request.Method == http.MethodHead {
		return
	}
	if _, err := io.Copy(c.Writer, r); err != nil {
		slog.ErrorContext(c.Request.Context(), "streaming object aborted", "key", info.Key, "error", err)
	}
}
