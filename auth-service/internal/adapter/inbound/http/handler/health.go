package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Checker interface {
	Ping(ctx context.Context) error
}

// HealthHandler handles health, liveness (/healthz), and readiness (/readyz) probes.
type HealthHandler struct {
	startTime time.Time
	dbChecker Checker
}

// NewHealthHandler creates a new instance of HealthHandler.
func NewHealthHandler(dbChecker Checker) *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
		dbChecker: dbChecker,
	}
}

// LivenessProbe (/healthz) - Always returns 200 OK if the process is running.
func (h *HealthHandler) LivenessProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadinessProbe (/readyz) - Returns 200 OK only if all dependencies (DB) are ready.
func (h *HealthHandler) ReadinessProbe(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if h.dbChecker != nil {
		if err := h.dbChecker.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "not_ready",
				"database": "down",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ready",
		"database": "up",
		"uptime":   time.Since(h.startTime).Round(time.Second).String(),
	})
}
