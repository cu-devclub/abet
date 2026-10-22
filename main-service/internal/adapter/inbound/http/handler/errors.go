// Package handler holds the Gin controllers. Each module has one file exposing Register(rg *gin.RouterGroup).
package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"abet/main-service/internal/adapter/inbound/http/dto"
	"abet/main-service/internal/core/domain"
)

// sentinelStatus maps domain sentinels to HTTP statuses, most specific first.
var sentinelStatus = []struct {
	err    error
	status int
	def    string // body message when the error carries no extra text
}{
	{domain.ErrInvalidInput, http.StatusBadRequest, "invalid input"},
	{domain.ErrUnauthorized, http.StatusUnauthorized, "unauthenticated"},
	{domain.ErrForbidden, http.StatusForbidden, "forbidden"},
	{domain.ErrNotFound, http.StatusNotFound, "not found"},
	{domain.ErrConflict, http.StatusConflict, "conflict"},
	{domain.ErrTooLarge, http.StatusRequestEntityTooLarge, "payload too large"},
	{domain.ErrUnprocessable, http.StatusUnprocessableEntity, "unprocessable"},
	{domain.ErrServiceUnavailable, http.StatusServiceUnavailable, "service unavailable"},
}

// writeError is the single place where errors become HTTP answers. Unknown errors are logged and answered
// with a generic 500; the error text never reaches the client in that case.
func writeError(c *gin.Context, err error) {
	for _, s := range sentinelStatus {
		if errors.Is(err, s.err) {
			c.JSON(s.status, dto.ErrorResponse{Error: publicMessage(err, s.err, s.def)})
			return
		}
	}
	slog.ErrorContext(c.Request.Context(), "unhandled error", "error", err, "method", c.Request.Method, "path", c.Request.URL.Path)
	c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "internal error"})
}

// publicMessage extracts the human-readable part that use cases attach after the sentinel
// ("invalid input: code cannot be empty" -> "code cannot be empty"). 401/403/503 keep fixed texts unless a
// use case deliberately added one.
func publicMessage(err, sentinel error, def string) string {
	marker := sentinel.Error() + ": "
	if i := strings.LastIndex(err.Error(), marker); i >= 0 {
		return err.Error()[i+len(marker):]
	}
	return def
}

func badRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: msg})
}

// parseUUIDParam reads a path parameter as UUID; on failure it answers 400 and returns false.
func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		badRequest(c, "invalid "+name)
		return uuid.Nil, false
	}
	return id, true
}

// bindJSON decodes the body into req and validates it; on failure it answers 400 and returns false.
func bindJSON[T interface{ Validate() error }](c *gin.Context, req T) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		badRequest(c, "invalid json: "+err.Error())
		return false
	}
	if err := req.Validate(); err != nil {
		writeError(c, err)
		return false
	}
	return true
}
