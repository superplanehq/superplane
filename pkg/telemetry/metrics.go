package telemetry

import (
	"context"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var (
	meter                          = otel.Meter("superplane")
	queueWorkerTickHistogram       metric.Float64Histogram
	queueWorkerHistogramReady      atomic.Bool
	queueWorkerNodesCountHistogram metric.Int64Histogram
	queueWorkerNodesHistogramReady atomic.Bool
	dbLocksCountHistogram          metric.Int64Histogram
	dbLocksCountHistogramReady     atomic.Bool
	dbLocksReporterInitializedFlag atomic.Bool
)

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
	meter = provider.Meter("superplane")

	queueWorkerTickHistogram, err = meter.Float64Histogram(
		"queue_worker.tick.duration.seconds",
		metric.WithDescription("Duration of each WorkflowNodeQueueWorker tick"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	queueWorkerHistogramReady.Store(true)

	queueWorkerNodesCountHistogram, err = meter.Int64Histogram(
		"queue_worker.tick.nodes.ready",
		metric.WithDescription("Number of workflow nodes ready to be processed each tick"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	queueWorkerNodesHistogramReady.Store(true)

	dbLocksCountHistogram, err = meter.Int64Histogram(
		"db.locks.count",
		metric.WithDescription("Number of database locks"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	dbLocksCountHistogramReady.Store(true)

	StartDatabaseLocksReporter(ctx)

	return nil
}

func RecordQueueWorkerTickDuration(ctx context.Context, d time.Duration) {
	if !queueWorkerHistogramReady.Load() {
		return
	}

	queueWorkerTickHistogram.Record(ctx, d.Seconds())
}

func RecordQueueWorkerNodesCount(ctx context.Context, count int) {
	if !queueWorkerNodesHistogramReady.Load() {
		return
	}

	queueWorkerNodesCountHistogram.Record(ctx, int64(count))
}
