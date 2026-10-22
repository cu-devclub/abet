package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"abet/main-service/internal/adapter/inbound/http/dto"
	"abet/main-service/internal/core/port"
)

// InternalAuthHandler serves the service-to-service endpoints under /internal/v1 (X-Internal-Token protected).
type InternalAuthHandler struct {
	loginSync port.LoginSyncUseCase
}

func NewInternalAuthHandler(loginSync port.LoginSyncUseCase) *InternalAuthHandler {
	return &InternalAuthHandler{loginSync: loginSync}
}

// Register mounts the routes on the /internal/v1 group.
func (h *InternalAuthHandler) Register(rg *gin.RouterGroup) {
	rg.POST("/auth/login-sync", h.LoginSync)
}

func (h *InternalAuthHandler) LoginSync(c *gin.Context) {
	var req dto.LoginSyncRequest
	if !bindJSON(c, &req) {
		return
	}
	u, err := h.loginSync.Execute(c.Request.Context(), req.ToInput())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.LoginSyncFromDomain(u))
}
