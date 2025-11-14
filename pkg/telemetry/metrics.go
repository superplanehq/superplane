package telemetry

import (
	"context"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter                      = otel.Meter("github.com/superplanehq/superplane")
	queueWorkerTickHistogram   metric.Float64Histogram
	queueWorkerHistogramReady  atomic.Bool
)

// InitMetrics configures the global meter provider and initializes metrics.
// It uses the standard OTLP metric gRPC exporter and relies on OpenTelemetry
// environment variables (e.g. OTEL_EXPORTER_OTLP_ENDPOINT) for configuration.
func InitMetrics(ctx context.Context) error {
	exporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return err
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter),
		),
	)

	otel.SetMeterProvider(provider)
	meter = provider.Meter("github.com/superplanehq/superplane")

	queueWorkerTickHistogram, err = meter.Float64Histogram(
		"queue_worker.tick.duration.seconds",
		metric.WithDescription("Duration of each WorkflowNodeQueueWorker tick"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	queueWorkerHistogramReady.Store(true)

	return nil
}

// RecordQueueWorkerTickDuration records the duration of a single tick of the
// WorkflowNodeQueueWorker. If metrics are not initialized, this is a no-op.
func RecordQueueWorkerTickDuration(ctx context.Context, d time.Duration) {
	if !queueWorkerHistogramReady.Load() {
		return
	}

	queueWorkerTickHistogram.Record(ctx, d.Seconds())
}

