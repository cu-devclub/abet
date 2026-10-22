package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"abet/auth-service/internal/adapter/inbound/http/dto"
	"abet/auth-service/internal/core/domain"
)

// writeError maps domain sentinels to HTTP status codes. Unknown errors are logged and
// reported as a generic 500 so internal details never leak.
func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrOAuthNotConfigured):
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{Error: "google oauth not configured"})
	case errors.Is(err, domain.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request"})
	case errors.Is(err, domain.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "unauthenticated"})
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Error: "forbidden"})
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "not found"})
	case errors.Is(err, domain.ErrConflict):
		c.JSON(http.StatusConflict, dto.ErrorResponse{Error: "conflict"})
	case errors.Is(err, domain.ErrServiceUnavailable), errors.Is(err, domain.ErrCircuitOpen):
		slog.ErrorContext(c.Request.Context(), "service unavailable", "path", c.FullPath(), "error", err)
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{Error: "service unavailable"})
	default:
		slog.ErrorContext(c.Request.Context(), "unhandled error", "path", c.FullPath(), "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "internal error"})
	}
}
