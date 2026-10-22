package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLog writes one structured slog line per request (health probes and metrics are skipped).
func RequestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/healthz" || path == "/readyz" || path == "/metrics" {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
		}
		// Only set while a super_admin is acting as someone else (dev act-as).
		if real, acting := RealActorFrom(c), ActorFrom(c); real != nil && acting != nil && real.UserID != acting.UserID {
			attrs = append(attrs, "real_user_id", real.UserID.String())
		}
		slog.InfoContext(c.Request.Context(), "request", attrs...)
	}
}

// Recovery turns panics into a logged error and a generic JSON 500 (no panic text leaks to the client).
func Recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
		slog.ErrorContext(c.Request.Context(), "panic recovered", "panic", recovered, "path", c.Request.URL.Path)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	})
}
