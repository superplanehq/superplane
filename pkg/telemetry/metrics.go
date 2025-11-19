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
	meter = otel.Meter("superplane")

	// Queue Worker Metrics
	queueWorkerTickHistogram       metric.Float64Histogram
	queueWorkerHistogramReady      atomic.Bool
	queueWorkerNodesCountHistogram metric.Int64Histogram
	queueWorkerNodesHistogramReady atomic.Bool

	// Executor Worker Metrics
	executorWorkerTickHistogram       metric.Float64Histogram
	executorWorkerTickHistogramReady  atomic.Bool
	executorWorkerNodesCountHistogram metric.Int64Histogram
	executorWorkerNodesHistogramReady atomic.Bool

	// Event Worker Metrics
	eventWorkerTickHistogram        metric.Float64Histogram
	eventWorkerTickHistogramReady   atomic.Bool
	eventWorkerEventsCountHistogram metric.Int64Histogram
	eventWorkerEventsHistogramReady atomic.Bool

	// Node Request Worker Metrics
	nodeRequestWorkerTickHistogram          metric.Float64Histogram
	nodeRequestWorkerTickHistogramReady     atomic.Bool
	nodeRequestWorkerRequestsCountHistogram metric.Int64Histogram
	nodeRequestWorkerRequestsHistogramReady atomic.Bool

	// Workflow Cleanup Worker Metrics
	workflowCleanupWorkerTickHistogram           metric.Float64Histogram
	workflowCleanupWorkerTickHistogramReady      atomic.Bool
	workflowCleanupWorkerWorkflowsCountHistogram metric.Int64Histogram
	workflowCleanupWorkerWorkflowsHistogramReady atomic.Bool

	// Database Locks Metrics
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

	executorWorkerTickHistogram, err = meter.Float64Histogram(
		"executor_worker.tick.duration.seconds",
		metric.WithDescription("Duration of each WorkflowNodeExecutor tick"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	executorWorkerTickHistogramReady.Store(true)

	executorWorkerNodesCountHistogram, err = meter.Int64Histogram(
		"executor_worker.tick.nodes.pending",
		metric.WithDescription("Number of pending workflow node executions each tick"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	executorWorkerNodesHistogramReady.Store(true)

	eventWorkerTickHistogram, err = meter.Float64Histogram(
		"event_worker.tick.duration.seconds",
		metric.WithDescription("Duration of each WorkflowEventRouter tick"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	eventWorkerTickHistogramReady.Store(true)

	eventWorkerEventsCountHistogram, err = meter.Int64Histogram(
		"event_worker.tick.events.pending",
		metric.WithDescription("Number of pending workflow events each tick"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	eventWorkerEventsHistogramReady.Store(true)

	nodeRequestWorkerTickHistogram, err = meter.Float64Histogram(
		"node_request_worker.tick.duration.seconds",
		metric.WithDescription("Duration of each NodeRequestWorker tick"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	nodeRequestWorkerTickHistogramReady.Store(true)

	nodeRequestWorkerRequestsCountHistogram, err = meter.Int64Histogram(
		"node_request_worker.tick.requests.pending",
		metric.WithDescription("Number of pending workflow node requests each tick"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	nodeRequestWorkerRequestsHistogramReady.Store(true)

	workflowCleanupWorkerTickHistogram, err = meter.Float64Histogram(
		"workflow_cleanup_worker.tick.duration.seconds",
		metric.WithDescription("Duration of each WorkflowCleanupWorker tick"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	workflowCleanupWorkerTickHistogramReady.Store(true)

	workflowCleanupWorkerWorkflowsCountHistogram, err = meter.Int64Histogram(
		"workflow_cleanup_worker.tick.workflows.deleted",
		metric.WithDescription("Number of deleted workflows processed each tick"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	workflowCleanupWorkerWorkflowsHistogramReady.Store(true)

	dbLocksCountHistogram, err = meter.Int64Histogram(
		"db.locks.count",
		metric.WithDescription("Number of database locks"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	dbLocksCountHistogramReady.Store(true)

	StartDatabaseLocksReporter(context.Background())

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

func RecordExecutorWorkerTickDuration(ctx context.Context, d time.Duration) {
	if !executorWorkerTickHistogramReady.Load() {
		return
	}

	executorWorkerTickHistogram.Record(ctx, d.Seconds())
}

func RecordExecutorWorkerNodesCount(ctx context.Context, count int) {
	if !executorWorkerNodesHistogramReady.Load() {
		return
	}

	executorWorkerNodesCountHistogram.Record(ctx, int64(count))
}

func RecordEventWorkerTickDuration(ctx context.Context, d time.Duration) {
	if !eventWorkerTickHistogramReady.Load() {
		return
	}

	eventWorkerTickHistogram.Record(ctx, d.Seconds())
}

func RecordEventWorkerEventsCount(ctx context.Context, count int) {
	if !eventWorkerEventsHistogramReady.Load() {
		return
	}

	eventWorkerEventsCountHistogram.Record(ctx, int64(count))
}

func RecordNodeRequestWorkerTickDuration(ctx context.Context, d time.Duration) {
	if !nodeRequestWorkerTickHistogramReady.Load() {
		return
	}

	nodeRequestWorkerTickHistogram.Record(ctx, d.Seconds())
}

func RecordNodeRequestWorkerRequestsCount(ctx context.Context, count int) {
	if !nodeRequestWorkerRequestsHistogramReady.Load() {
		return
	}

	nodeRequestWorkerRequestsCountHistogram.Record(ctx, int64(count))
}

func RecordWorkflowCleanupWorkerTickDuration(ctx context.Context, d time.Duration) {
	if !workflowCleanupWorkerTickHistogramReady.Load() {
		return
	}

	workflowCleanupWorkerTickHistogram.Record(ctx, d.Seconds())
}

func RecordWorkflowCleanupWorkerWorkflowsCount(ctx context.Context, count int) {
	if !workflowCleanupWorkerWorkflowsHistogramReady.Load() {
		return
	}

	workflowCleanupWorkerWorkflowsCountHistogram.Record(ctx, int64(count))
}
