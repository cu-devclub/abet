package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HeaderInternalToken carries the shared service-to-service secret.
const HeaderInternalToken = "X-Internal-Token"

// InternalToken guards /internal/v1: constant-time comparison with the configured token.
// An empty configured token rejects every request.
func InternalToken(token string) gin.HandlerFunc {
	want := []byte(token)
	return func(c *gin.Context) {
		got := []byte(c.GetHeader(HeaderInternalToken))
		if len(want) == 0 || subtle.ConstantTimeCompare(got, want) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}
		c.Next()
	}
}
