package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"abet/main-service/internal/adapter/inbound/http/dto"
	"abet/main-service/internal/adapter/inbound/http/middleware"
	"abet/main-service/internal/core/port"
)

type MeHandler struct {
	getMe port.GetMeUseCase
	actAs port.GetActAsStateUseCase // optional; nil = act_as is always {available:false}
}

func NewMeHandler(getMe port.GetMeUseCase) *MeHandler { return &MeHandler{getMe: getMe} }

// WithActAs adds the dev act-as state to GET /me.
func (h *MeHandler) WithActAs(uc port.GetActAsStateUseCase) *MeHandler {
	h.actAs = uc
	return h
}

// Register mounts the routes on the authenticated /api/v1 group.
func (h *MeHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/me", h.Get)
}

func (h *MeHandler) Get(c *gin.Context) {
	actor := middleware.ActorFrom(c)
	res, err := h.getMe.Execute(c.Request.Context(), actor)
	if err != nil {
		writeError(c, err)
		return
	}
	out := dto.MeFromResult(res, actor)
	if h.actAs != nil {
		state, err := h.actAs.Execute(c.Request.Context(), middleware.RealActorFrom(c), actor)
		if err != nil {
			writeError(c, err)
			return
		}
		out.ActAs = dto.ActAsStateFromPort(state)
	}
	c.JSON(http.StatusOK, out)
}
