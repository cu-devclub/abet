package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"abet/main-service/internal/core/port"
)

// TemplateHandler serves the downloadable templates (any authenticated user).
type TemplateHandler struct {
	templates port.TemplateService
}

func NewTemplateHandler(templates port.TemplateService) *TemplateHandler {
	return &TemplateHandler{templates: templates}
}

func (h *TemplateHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/templates/instructor-package.zip", h.InstructorPackage)
	rg.GET("/templates/course-instructor-mapping.csv", h.MappingCSV)
}

func (h *TemplateHandler) InstructorPackage(c *gin.Context) {
	hdr := c.Writer.Header()
	hdr.Set("Content-Type", "application/zip")
	hdr.Set("Content-Disposition", contentDisposition("attachment", "course-template.zip"))
	hdr.Set("Cache-Control", "no-cache")
	c.Status(http.StatusOK)
	if err := h.templates.WriteInstructorPackage(c.Writer); err != nil {
		slog.ErrorContext(c.Request.Context(), "instructor package download failed", "error", err)
	}
}

func (h *TemplateHandler) MappingCSV(c *gin.Context) {
	hdr := c.Writer.Header()
	hdr.Set("Content-Disposition", contentDisposition("attachment", "course-instructor-mapping.csv"))
	hdr.Set("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", h.templates.CourseMappingCSV())
}
