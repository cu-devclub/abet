package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"abet/main-service/internal/adapter/inbound/http/dto"
	"abet/main-service/internal/adapter/inbound/http/middleware"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

type UserHandler struct {
	create port.CreateUserUseCase
	update port.UpdateUserUseCase
	del    port.DeleteUserUseCase
	list   port.ListUsersUseCase
}

func NewUserHandler(
	create port.CreateUserUseCase,
	update port.UpdateUserUseCase,
	del port.DeleteUserUseCase,
	list port.ListUsersUseCase,
) *UserHandler {
	return &UserHandler{create: create, update: update, del: del, list: list}
}

// Register mounts the routes on the authenticated /api/v1 group.
func (h *UserHandler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/users")
	g.GET("", h.List)
	g.POST("", h.Create)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

func (h *UserHandler) List(c *gin.Context) {
	in := port.ListUsersInput{Search: c.Query("search")}
	if raw := c.Query("role"); raw != "" {
		role := domain.Role(raw)
		in.Role = &role
	}
	if raw := c.Query("department_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			badRequest(c, "invalid department_id")
			return
		}
		in.DepartmentID = &id
	}
	actor := middleware.ActorFrom(c)
	items, err := h.list.Execute(c.Request.Context(), actor, in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Items(dto.UsersFromDomain(items, actor)))
}

func (h *UserHandler) Create(c *gin.Context) {
	var req dto.CreateUserRequest
	if !bindJSON(c, &req) {
		return
	}
	actor := middleware.ActorFrom(c)
	u, err := h.create.Execute(c.Request.Context(), actor, req.ToInput())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.UserFromDomain(u, actor))
}

func (h *UserHandler) Update(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateUserRequest
	if !bindJSON(c, &req) {
		return
	}
	actor := middleware.ActorFrom(c)
	u, err := h.update.Execute(c.Request.Context(), actor, req.ToInput(id))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.UserFromDomain(u, actor))
}

func (h *UserHandler) Delete(c *gin.Context) {
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
