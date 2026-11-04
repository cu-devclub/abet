package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"abet/main-service/internal/adapter/inbound/http/middleware"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// CourseTemplateHandler serves the System Config endpoints for the course template (super_admin only).
type CourseTemplateHandler struct {
	get      port.GetCourseTemplateUseCase
	put      port.PutCourseTemplateUseCase
	del      port.DeleteCourseTemplateUseCase
	maxBytes int64
}

func NewCourseTemplateHandler(get port.GetCourseTemplateUseCase, put port.PutCourseTemplateUseCase, del port.DeleteCourseTemplateUseCase, maxBytes int64) *CourseTemplateHandler {
	return &CourseTemplateHandler{get: get, put: put, del: del, maxBytes: maxBytes}
}

func (h *CourseTemplateHandler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/config/course-template")
	g.GET("", h.Get)
	g.PUT("", h.Put)
	g.DELETE("", h.Delete)
}

type courseTemplateResponse struct {
	Source     string     `json:"source"`
	SizeBytes  *int64     `json:"size_bytes"`
	UploadedAt *time.Time `json:"uploaded_at"`
	UploadedBy *string    `json:"uploaded_by"`
}

func respondCourseTemplate(c *gin.Context, i *port.CourseTemplateInfo, err error) {
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, courseTemplateResponse{Source: i.Source, SizeBytes: i.SizeBytes, UploadedAt: i.UploadedAt, UploadedBy: i.UploadedBy})
}

func (h *CourseTemplateHandler) Get(c *gin.Context) {
	i, err := h.get.Execute(c.Request.Context(), middleware.ActorFrom(c))
	respondCourseTemplate(c, i, err)
}

func (h *CourseTemplateHandler) Delete(c *gin.Context) {
	i, err := h.del.Execute(c.Request.Context(), middleware.ActorFrom(c))
	respondCourseTemplate(c, i, err)
}

func (h *CourseTemplateHandler) Put(c *gin.Context) {
	if !middleware.ActorFrom(c).CanManageSystem() { // before reading the body
		writeError(c, domain.ErrForbidden)
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
	f, err := fh.Open()
	if err != nil {
		badRequest(c, "cannot read file")
		return
	}
	defer func() { _ = f.Close() }()
	i, err := h.put.Execute(c.Request.Context(), middleware.ActorFrom(c), f)
	respondCourseTemplate(c, i, err)
}
