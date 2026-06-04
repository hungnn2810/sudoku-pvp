// Package logger provides a zerolog-based global logger setup with
// OpenTelemetry trace-ID injection from span context.
package logger

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

// Setup configures the global zerolog logger with JSON output to stdout.
// Call once at process start before any other logging occurs.
// JSON-only output (no console writer) per production logging standards (REQ-026).
func Setup(level zerolog.Level) {
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.SetGlobalLevel(level)
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
}

// ParseLevel converts a string log level to a zerolog.Level.
// Defaults to zerolog.InfoLevel on parse error.
func ParseLevel(s string) zerolog.Level {
	level, err := zerolog.ParseLevel(s)
	if err != nil {
		return zerolog.InfoLevel
	}
	return level
}

// FromCtx returns a zerolog.Logger enriched with trace_id and span_id fields
// when an active OTel span is present in ctx. If no valid span context exists,
// the global logger is returned unchanged.
//
// CRITICAL: This must be called with c.Request.Context() in Gin handlers (not
// c.Copy()) to preserve the OTel trace context injected by otelgin middleware.
// See Research Pitfall 4: otelgin context preservation.
func FromCtx(ctx context.Context) *zerolog.Logger {
	l := log.Ctx(ctx)
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
		sub := l.With().
			Str("trace_id", sc.TraceID().String()).
			Str("span_id", sc.SpanID().String()).
			Logger()
		return &sub
	}
	return l
}
