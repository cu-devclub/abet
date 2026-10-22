package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"abet/main-service/internal/adapter/inbound/http/dto"
	"abet/main-service/internal/adapter/inbound/http/middleware"
	"abet/main-service/internal/core/port"
)

type DepartmentHandler struct {
	create port.CreateDepartmentUseCase
	update port.UpdateDepartmentUseCase
	del    port.DeleteDepartmentUseCase
	list   port.ListDepartmentsUseCase
}

func NewDepartmentHandler(
	create port.CreateDepartmentUseCase,
	update port.UpdateDepartmentUseCase,
	del port.DeleteDepartmentUseCase,
	list port.ListDepartmentsUseCase,
) *DepartmentHandler {
	return &DepartmentHandler{create: create, update: update, del: del, list: list}
}

// Register mounts the routes on the authenticated /api/v1 group.
func (h *DepartmentHandler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/departments")
	g.GET("", h.List)
	g.POST("", h.Create)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

func (h *DepartmentHandler) List(c *gin.Context) {
	items, err := h.list.Execute(c.Request.Context(), middleware.ActorFrom(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Items(dto.DepartmentsFromDomain(items)))
}

func (h *DepartmentHandler) Create(c *gin.Context) {
	var req dto.CreateDepartmentRequest
	if !bindJSON(c, &req) {
		return
	}
	d, err := h.create.Execute(c.Request.Context(), middleware.ActorFrom(c), req.ToInput())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.DepartmentFromDomain(d))
}

func (h *DepartmentHandler) Update(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateDepartmentRequest
	if !bindJSON(c, &req) {
		return
	}
	d, err := h.update.Execute(c.Request.Context(), middleware.ActorFrom(c), req.ToInput(id))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.DepartmentFromDomain(d))
}

func (h *DepartmentHandler) Delete(c *gin.Context) {
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
