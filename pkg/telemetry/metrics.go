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
	meter        = otel.Meter("superplane")
	metricsReady atomic.Bool

	queueWorkerTickHistogram       metric.Float64Histogram
	queueWorkerNodesCountHistogram metric.Int64Histogram
	queueWorkerStuckItems          metric.Int64Histogram

	executorWorkerTickHistogram       metric.Float64Histogram
	executorWorkerNodesCountHistogram metric.Int64Histogram

	eventWorkerTickHistogram        metric.Float64Histogram
	eventWorkerEventsCountHistogram metric.Int64Histogram

	nodeRequestWorkerTickHistogram          metric.Float64Histogram
	nodeRequestWorkerRequestsCountHistogram metric.Int64Histogram

	workflowCleanupWorkerTickHistogram           metric.Float64Histogram
	workflowCleanupWorkerWorkflowsCountHistogram metric.Int64Histogram

	dbLocksCountHistogram       metric.Int64Histogram
	dbLongQueriesCountHistogram metric.Int64Histogram
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

	queueWorkerNodesCountHistogram, err = meter.Int64Histogram(
		"queue_worker.tick.nodes.ready",
		metric.WithDescription("Number of workflow nodes ready to be processed each tick"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	executorWorkerTickHistogram, err = meter.Float64Histogram(
		"executor_worker.tick.duration.seconds",
		metric.WithDescription("Duration of each WorkflowNodeExecutor tick"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	executorWorkerNodesCountHistogram, err = meter.Int64Histogram(
		"executor_worker.tick.nodes.pending",
		metric.WithDescription("Number of pending workflow node executions each tick"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	eventWorkerTickHistogram, err = meter.Float64Histogram(
		"event_worker.tick.duration.seconds",
		metric.WithDescription("Duration of each WorkflowEventRouter tick"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	eventWorkerEventsCountHistogram, err = meter.Int64Histogram(
		"event_worker.tick.events.pending",
		metric.WithDescription("Number of pending workflow events each tick"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	nodeRequestWorkerTickHistogram, err = meter.Float64Histogram(
		"node_request_worker.tick.duration.seconds",
		metric.WithDescription("Duration of each NodeRequestWorker tick"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	nodeRequestWorkerRequestsCountHistogram, err = meter.Int64Histogram(
		"node_request_worker.tick.requests.pending",
		metric.WithDescription("Number of pending workflow node requests each tick"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	workflowCleanupWorkerTickHistogram, err = meter.Float64Histogram(
		"workflow_cleanup_worker.tick.duration.seconds",
		metric.WithDescription("Duration of each WorkflowCleanupWorker tick"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	workflowCleanupWorkerWorkflowsCountHistogram, err = meter.Int64Histogram(
		"workflow_cleanup_worker.tick.workflows.deleted",
		metric.WithDescription("Number of deleted workflows processed each tick"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	dbLocksCountHistogram, err = meter.Int64Histogram(
		"db.locks.count",
		metric.WithDescription("Number of database locks"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	dbLongQueriesCountHistogram, err = meter.Int64Histogram(
		"db.long_queries.count",
		metric.WithDescription("Number of long-running database queries"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	queueWorkerStuckItems, err = meter.Int64Histogram(
		"queue_items.stuck.count",
		metric.WithDescription("Number of stuck workflow node queue items"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	StartPeriodicMetricsReporter()

	metricsReady.Store(true)

	return nil
}

func StartPeriodicMetricsReporter() {
	p := NewPeriodic(context.Background())
	p.Start()
}

func RecordQueueWorkerTickDuration(ctx context.Context, d time.Duration) {
	if !metricsReady.Load() {
		return
	}

	queueWorkerTickHistogram.Record(ctx, d.Seconds())
}

func RecordQueueWorkerNodesCount(ctx context.Context, count int) {
	if !metricsReady.Load() {
		return
	}

	queueWorkerNodesCountHistogram.Record(ctx, int64(count))
}

func RecordExecutorWorkerTickDuration(ctx context.Context, d time.Duration) {
	if !metricsReady.Load() {
		return
	}

	executorWorkerTickHistogram.Record(ctx, d.Seconds())
}

func RecordExecutorWorkerNodesCount(ctx context.Context, count int) {
	if !metricsReady.Load() {
		return
	}

	executorWorkerNodesCountHistogram.Record(ctx, int64(count))
}

func RecordEventWorkerTickDuration(ctx context.Context, d time.Duration) {
	if !metricsReady.Load() {
		return
	}

	eventWorkerTickHistogram.Record(ctx, d.Seconds())
}

func RecordEventWorkerEventsCount(ctx context.Context, count int) {
	if !metricsReady.Load() {
		return
	}

	eventWorkerEventsCountHistogram.Record(ctx, int64(count))
}

func RecordNodeRequestWorkerTickDuration(ctx context.Context, d time.Duration) {
	if !metricsReady.Load() {
		return
	}

	nodeRequestWorkerTickHistogram.Record(ctx, d.Seconds())
}

func RecordNodeRequestWorkerRequestsCount(ctx context.Context, count int) {
	if !metricsReady.Load() {
		return
	}

	nodeRequestWorkerRequestsCountHistogram.Record(ctx, int64(count))
}

func RecordWorkflowCleanupWorkerTickDuration(ctx context.Context, d time.Duration) {
	if !metricsReady.Load() {
		return
	}

	workflowCleanupWorkerTickHistogram.Record(ctx, d.Seconds())
}

func RecordWorkflowCleanupWorkerWorkflowsCount(ctx context.Context, count int) {
	if !metricsReady.Load() {
		return
	}

	workflowCleanupWorkerWorkflowsCountHistogram.Record(ctx, int64(count))
}

func RecordDBLocksCount(ctx context.Context, count int64) {
	if !metricsReady.Load() {
		return
	}

	dbLocksCountHistogram.Record(ctx, count)
}

func RecordStuckQueueItemsCount(ctx context.Context, count int) {
	if !metricsReady.Load() {
		return
	}

	queueWorkerStuckItems.Record(ctx, int64(count))
}

func RecordDBLongQueriesCount(ctx context.Context, count int64) {
	if !metricsReady.Load() {
		return
	}

	dbLongQueriesCountHistogram.Record(ctx, count)
}
