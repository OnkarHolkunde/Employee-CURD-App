package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"excel-crud-app/internal/apperrors"
	"excel-crud-app/internal/response"

	"github.com/gin-gonic/gin"
)

// Recovery catches panics in handlers and converts them into a clean JSON
// 500 response (instead of Gin's default plaintext dump), while still
// logging the stack trace server-side for debugging. The client only ever
// sees apperrors.NewInternal()'s fixed, generic message — never the panic
// value itself, which could easily contain sensitive internal detail.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered",
					"request_id", GetRequestID(c),
					"error", fmt.Sprintf("%v", rec),
					"stack", string(debug.Stack()),
				)
				response.Error(c, apperrors.NewInternal())
				c.Abort()
			}
		}()
		c.Next()
	}
}
