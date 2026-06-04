package telemetry

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"sudoku-pvp/internal/config"
)

// Bootstrap initialises the global OTel TracerProvider and MeterProvider.
//
// Trace exporter: stdout for Phase 1 (OTLP support deferred to Phase 7 when
// the trace backend — Grafana Tempo / Jaeger — is confirmed per open question
// in 01-RESEARCH.md). OTLPEndpoint is accepted but currently ignored; passing
// a non-empty value is a no-op in Phase 1.
//
// Metrics exporter: Prometheus (scrape-based). The exporter registers with the
// default prometheus.DefaultRegisterer, which promhttp.Handler() exposes at /metrics.
//
// The returned shutdown function must be deferred in main() to flush and close
// both providers on graceful shutdown (Pitfall 6 — avoid dangling connections).
func Bootstrap(ctx context.Context, cfg config.TelemetryConfig) (func(context.Context) error, error) {
	res, err := newResource(ctx)
	if err != nil {
		return nil, fmt.Errorf("telemetry resource: %w", err)
	}

	// --- Metrics: Prometheus exporter ---
	mp, err := newMeterProvider(res)
	if err != nil {
		return nil, fmt.Errorf("meter provider: %w", err)
	}
	otel.SetMeterProvider(mp)

	// --- Traces: stdout exporter (Phase 1) ---
	// OTLP exporter will be added in Phase 7 when trace backend is confirmed.
	// cfg.OTLPEndpoint is preserved in the config struct for future use.
	_ = cfg.OTLPEndpoint // explicitly acknowledge; stdout always used in Phase 1
	tp, err := newStdoutTracerProvider(res)
	if err != nil {
		return nil, fmt.Errorf("tracer provider: %w", err)
	}
	otel.SetTracerProvider(tp)

	// --- Propagator: W3C TraceContext + Baggage ---
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdown := func(ctx context.Context) error {
		var errs []error
		if err := mp.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter provider shutdown: %w", err))
		}
		if err := tp.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer provider shutdown: %w", err))
		}
		return errors.Join(errs...)
	}
	return shutdown, nil
}
