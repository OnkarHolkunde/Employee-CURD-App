package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"
const requestIDContextKey = "request_id"

// RequestID assigns a unique ID to every incoming request (reusing one
// supplied by an upstream proxy/load balancer if present) so that a single
// request can be traced through logs, error responses, and downstream
// calls.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(RequestIDHeader)
		if reqID == "" {
			reqID = uuid.NewString()
		}
		c.Set(requestIDContextKey, reqID)
		c.Writer.Header().Set(RequestIDHeader, reqID)
		c.Next()
	}
}

// GetRequestID retrieves the request ID stashed by the RequestID middleware.
func GetRequestID(c *gin.Context) string {
	if v, ok := c.Get(requestIDContextKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
