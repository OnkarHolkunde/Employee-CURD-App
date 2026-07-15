package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// StructuredLogger replaces gin's default text logger with structured
// (slog) log lines that include the request ID, making requests easy to
// grep/aggregate in a real deployment (e.g. shipped to ELK/Loki).
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		attrs := []any{
			"request_id", GetRequestID(c),
			"method", c.Request.Method,
			"path", path,
			"query", raw,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		switch {
		case status >= 500:
			slog.Error("request", attrs...)
		case status >= 400:
			slog.Warn("request", attrs...)
		default:
			slog.Info("request", attrs...)
		}
	}
}
