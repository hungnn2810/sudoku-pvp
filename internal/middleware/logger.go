package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"sudoku-pvp/internal/logger"
)

// ZerologLogger returns a Gin middleware that logs one structured JSON access
// log entry per HTTP request after the handler chain completes.
//
// The log entry contains: method, path, status code, latency, and client IP.
// When an OTel span is active (set by Tracing() middleware), trace_id and
// span_id are injected automatically via logger.FromCtx.
//
// CRITICAL (Research Pitfall 4): Uses c.Request.Context() — NOT c.Copy() —
// to preserve the OTel trace context injected by otelgin.Middleware. Using
// c.Copy() would create a new context without the span, resulting in all
// trace IDs being the zero value in logs.
func ZerologLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		logger.FromCtx(c.Request.Context()).Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency", time.Since(start)).
			Str("ip", c.ClientIP()).
			Msg("request")
	}
}
