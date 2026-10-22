package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"abet/main-service/internal/adapter/inbound/http/dto"
	"abet/main-service/internal/adapter/inbound/http/middleware"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// ActAsHandler serves the dev-only "act as user" endpoints (D28). It is only constructed and registered when the
// feature is on; otherwise the routes do not exist (404).
type ActAsHandler struct {
	list         port.ListActAsUsersUseCase
	start        port.StartActAsUseCase
	cookieSecure bool
}

func NewActAsHandler(list port.ListActAsUsersUseCase, start port.StartActAsUseCase, cookieSecure bool) *ActAsHandler {
	return &ActAsHandler{list: list, start: start, cookieSecure: cookieSecure}
}

// Register mounts the routes on the authenticated /api/v1 group.
func (h *ActAsHandler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/dev/act-as")
	g.GET("/users", h.Users)
	g.POST("", h.Start)
	g.DELETE("", h.Stop)
}

// Users lists the users the real super_admin can switch to.
func (h *ActAsHandler) Users(c *gin.Context) {
	in := port.ListActAsUsersInput{Search: c.Query("q")}
	if raw := c.Query("role"); raw != "" {
		role := domain.Role(raw)
		in.Role = &role
	}
	users, err := h.list.Execute(c.Request.Context(), middleware.RealActorFrom(c), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ActAsUsersFromDomain(users))
}

// Start validates the target and sets the abet_act_as cookie.
func (h *ActAsHandler) Start(c *gin.Context) {
	real := middleware.RealActorFrom(c)
	// Authorization first so a non-super_admin learns nothing from validation errors.
	if real == nil || real.Role != domain.RoleSuperAdmin {
		writeError(c, domain.ErrForbidden)
		return
	}
	var req dto.ActAsRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.start.Execute(c.Request.Context(), real, req.UserID); err != nil {
		writeError(c, err)
		return
	}
	h.setCookie(c, req.UserID.String(), 0)
	c.Status(http.StatusNoContent)
}

// Stop clears the cookie (idempotent).
func (h *ActAsHandler) Stop(c *gin.Context) {
	real := middleware.RealActorFrom(c)
	if real == nil || real.Role != domain.RoleSuperAdmin {
		writeError(c, domain.ErrForbidden)
		return
	}
	h.setCookie(c, "", -1)
	c.Status(http.StatusNoContent)
}

// setCookie writes the act-as cookie: HttpOnly, SameSite=Lax, session lifetime (maxAge 0) or cleared (maxAge -1).
func (h *ActAsHandler) setCookie(c *gin.Context, value string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.ActAsCookieName, value, maxAge, middleware.ActAsCookiePath, "", h.cookieSecure, true)
}
