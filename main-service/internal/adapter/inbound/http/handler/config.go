package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AppConfig is the part of the .env settings the frontend needs.
type AppConfig struct {
	MaxUploadMB         int `json:"max_upload_mb"`
	MaxCoverMB          int `json:"max_cover_mb"`
	MaxImportMB         int `json:"max_import_mb"`
	BuildTimeoutMinutes int `json:"build_timeout_minutes"`
}

// ConfigHandler serves GET /config to every authenticated user.
type ConfigHandler struct {
	cfg AppConfig
}

func NewConfigHandler(cfg AppConfig) *ConfigHandler { return &ConfigHandler{cfg: cfg} }

func (h *ConfigHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/config", h.Get)
}

func (h *ConfigHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, h.cfg)
}
