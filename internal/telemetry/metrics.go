package telemetry

import (
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

// newMeterProvider creates an OTel MeterProvider backed by a Prometheus exporter.
// The Prometheus exporter registers metrics with the default Prometheus registry,
// which is served by promhttp.Handler() on the /metrics endpoint.
func newMeterProvider(res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	promExporter, err := otelprom.New()
	if err != nil {
		return nil, err
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(promExporter),
		sdkmetric.WithResource(res),
	)
	return mp, nil
}
