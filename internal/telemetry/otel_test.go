//go:build !integration

package telemetry_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"sudoku-pvp/internal/config"
	"sudoku-pvp/internal/telemetry"
)

// TestBootstrap verifies that Bootstrap sets global OTel providers and returns
// a non-nil shutdown function. Does not require any external infrastructure.
func TestBootstrap(t *testing.T) {
	ctx := context.Background()

	cfg := config.TelemetryConfig{
		OTLPEndpoint: "", // stdout exporter always used in Phase 1
	}

	shutdown, err := telemetry.Bootstrap(ctx, cfg)
	if err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("Bootstrap returned nil shutdown function")
	}

	// Verify global providers were registered.
	if otel.GetTracerProvider() == nil {
		t.Error("TracerProvider not set after Bootstrap")
	}
	if otel.GetMeterProvider() == nil {
		t.Error("MeterProvider not set after Bootstrap")
	}

	// Graceful shutdown must not error.
	if err := shutdown(ctx); err != nil {
		t.Errorf("shutdown returned error: %v", err)
	}
}
