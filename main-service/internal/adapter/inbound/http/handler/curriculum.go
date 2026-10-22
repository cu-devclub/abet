package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"abet/main-service/internal/adapter/inbound/http/dto"
	"abet/main-service/internal/adapter/inbound/http/middleware"
	"abet/main-service/internal/core/port"
)

type CurriculumHandler struct {
	create port.CreateCurriculumUseCase
	update port.UpdateCurriculumUseCase
	del    port.DeleteCurriculumUseCase
	list   port.ListCurriculaUseCase
}

func NewCurriculumHandler(
	create port.CreateCurriculumUseCase,
	update port.UpdateCurriculumUseCase,
	del port.DeleteCurriculumUseCase,
	list port.ListCurriculaUseCase,
) *CurriculumHandler {
	return &CurriculumHandler{create: create, update: update, del: del, list: list}
}

// Register mounts the routes on the authenticated /api/v1 group.
// (Cover endpoints under /curricula/:id/cover are added by their own handler, see README "Extending".)
func (h *CurriculumHandler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/curricula")
	g.GET("", h.List)
	g.POST("", h.Create)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

func (h *CurriculumHandler) List(c *gin.Context) {
	var deptID *uuid.UUID
	if raw := c.Query("department_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			badRequest(c, "invalid department_id")
			return
		}
		deptID = &id
	}
	items, err := h.list.Execute(c.Request.Context(), middleware.ActorFrom(c), deptID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Items(dto.CurriculaFromDomain(items)))
}

func (h *CurriculumHandler) Create(c *gin.Context) {
	var req dto.CreateCurriculumRequest
	if !bindJSON(c, &req) {
		return
	}
	cur, err := h.create.Execute(c.Request.Context(), middleware.ActorFrom(c), req.ToInput())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.CurriculumFromDomain(cur))
}

func (h *CurriculumHandler) Update(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateCurriculumRequest
	if !bindJSON(c, &req) {
		return
	}
	cur, err := h.update.Execute(c.Request.Context(), middleware.ActorFrom(c), req.ToInput(id))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.CurriculumFromDomain(cur))
}

func (h *CurriculumHandler) Delete(c *gin.Context) {
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
