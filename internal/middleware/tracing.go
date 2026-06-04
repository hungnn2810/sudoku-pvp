package middleware

import (
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"github.com/gin-gonic/gin"
)

// serviceName is the constant service name embedded in every OTel span.
const serviceName = "sudoku-pvp-api"

// Tracing returns a Gin middleware that creates an OTel span per HTTP request
// and propagates W3C TraceContext headers from inbound requests.
//
// This is a thin wrapper around otelgin.Middleware that pins the service name
// constant. Must be registered BEFORE ZerologLogger so trace context is
// available when the access log is written (Research Pitfall 4).
func Tracing() gin.HandlerFunc {
	return otelgin.Middleware(serviceName)
}
