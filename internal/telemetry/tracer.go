// Package telemetry provides OpenTelemetry bootstrap helpers for the
// sudoku-pvp API service.
package telemetry

import (
	"context"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// newResource creates an OTel resource with the service name attribute.
func newResource(ctx context.Context) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName("sudoku-pvp-api")),
	)
}

// newStdoutTracerProvider creates a TracerProvider that exports spans to stdout.
// Used for Phase 1 dev/test; OTLP exporter will be wired in Phase 7 when the
// trace backend (Grafana Tempo / Jaeger) is confirmed.
func newStdoutTracerProvider(res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exporter, err := stdouttrace.New()
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	return tp, nil
}
