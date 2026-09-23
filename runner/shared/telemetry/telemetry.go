// Package telemetry configures OpenTelemetry metrics export for runner binaries.
// When OTEL_EXPORTER_OTLP_ENDPOINT is unset, Init installs a no-op meter provider
// so local dev and tests behave as today.
package telemetry

import (
	"context"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

// Init configures the global OpenTelemetry meter provider.
// Returns enabled=false when metrics export is disabled (no OTLP endpoint).
func Init(ctx context.Context) (shutdown func(context.Context) error, enabled bool, err error) {
	if !metricsExportEnabled() {
		otel.SetMeterProvider(metric.NewMeterProvider())
		return func(context.Context) error { return nil }, false, nil
	}

	reader, err := autoexport.NewMetricReader(ctx)
	if err != nil {
		return nil, false, err
	}

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(semconv.ServiceName(serviceName())),
	)
	if err != nil {
		return nil, false, err
	}

	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(reader),
	)
	otel.SetMeterProvider(mp)
	return mp.Shutdown, true, nil
}

func metricsExportEnabled() bool {
	if strings.ToLower(strings.TrimSpace(os.Getenv("OTEL_METRICS_EXPORTER"))) == "none" {
		return false
	}
	return strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")) != ""
}

func serviceName() string {
	if v := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME")); v != "" {
		return v
	}
	return "unknown"
}
