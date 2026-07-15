package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS allows the API to be called from browser-based clients (e.g. a
// separate frontend app or Postman's browser runtime). Origins are
// configurable so production can lock this down to a known frontend
// domain instead of "*".
func CORS(allowedOrigin string) gin.HandlerFunc {
	if allowedOrigin == "" {
		allowedOrigin = "*"
	}
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, "+RequestIDHeader)

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
