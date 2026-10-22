package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"abet/main-service/internal/core/port"
)

// HealthHandler serves /healthz (liveness) and /readyz (readiness, pings the DB).
type HealthHandler struct {
	startTime time.Time
	db        port.Pinger
	storage   port.Pinger
	queue     port.QueueReporter
}

func NewHealthHandler(db port.Pinger) *HealthHandler {
	return &HealthHandler{startTime: time.Now(), db: db}
}

// WithStorage makes /readyz also depend on the object store.
func (h *HealthHandler) WithStorage(s port.Pinger) *HealthHandler {
	h.storage = s
	return h
}

// WithQueue adds the build queue state (workers, waiting jobs) to the /readyz details.
func (h *HealthHandler) WithQueue(q port.QueueReporter) *HealthHandler {
	h.queue = q
	return h
}

// LivenessProbe always answers 200 while the process runs.
func (h *HealthHandler) LivenessProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive", "time": time.Now().UTC().Format(time.RFC3339)})
}

// ReadinessProbe answers 200 only when the database (and the object store) is reachable.
func (h *HealthHandler) ReadinessProbe(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "database": "down"})
			return
		}
	}
	if h.storage != nil {
		if err := h.storage.Ping(ctx); err != nil {
			slog.Warn("readiness: object storage unavailable", "error", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "database": "up", "storage": "down"})
			return
		}
	}
	body := gin.H{
		"status":   "ready",
		"database": "up",
		"uptime":   time.Since(h.startTime).Round(time.Second).String(),
	}
	if h.storage != nil {
		body["storage"] = "up"
	}
	// Informational only: uploads stay possible without workers (jobs wait in the queue, the timeout is the backstop).
	if h.queue != nil {
		if s := h.queue.Snapshot(); s.Known {
			body["build_queue"] = gin.H{
				"consumers":  s.Consumers,
				"messages":   s.Messages,
				"checked_at": s.CheckedAt.UTC().Format(time.RFC3339),
			}
		} else {
			body["build_queue"] = gin.H{"status": "unknown"}
		}
	}
	c.JSON(http.StatusOK, body)
}
