package telemetry

import (
	"context"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var (
	meter        = otel.Meter("superplane")
	metricsReady atomic.Bool

	queueWorkerTickHistogram         metric.Float64Histogram
	queueWorkerNodesCountHistogram   metric.Int64Histogram
	queueWorkerNodesCounter          metric.Int64Counter
	queueWorkerNodeDurationHistogram metric.Float64Histogram
	queueWorkerStuckItems            metric.Int64Histogram

	executorWorkerTickHistogram              metric.Float64Histogram
	executorWorkerNodesCountHistogram        metric.Int64Histogram
	executorWorkerExecutionsCounter          metric.Int64Counter
	executorWorkerExecutionDurationHistogram metric.Float64Histogram

	eventWorkerTickHistogram          metric.Float64Histogram
	eventWorkerEventsCountHistogram   metric.Int64Histogram
	eventWorkerEventsCounter          metric.Int64Counter
	eventWorkerEventDurationHistogram metric.Float64Histogram

	nodeRequestWorkerTickHistogram          metric.Float64Histogram
	nodeRequestWorkerRequestsCountHistogram metric.Int64Histogram

	workflowCleanupWorkerTickHistogram          metric.Float64Histogram
	workflowCleanupWorkerCanvasesCountHistogram metric.Int64Histogram

	dbLocksCountHistogram       metric.Int64Histogram
	dbLongQueriesCountHistogram metric.Int64Histogram

	dbPoolConnectionsMaxGauge   metric.Int64Gauge
	dbPoolConnectionsOpenGauge  metric.Int64Gauge
	dbPoolConnectionsInUseGauge metric.Int64Gauge
	dbPoolConnectionsIdleGauge  metric.Int64Gauge
	dbPoolWaitCountCounter      metric.Int64Counter
	dbPoolWaitDurationHistogram metric.Float64Histogram

	dbRowsAffectedCounter metric.Int64Counter

	integrationSecretWritesCounter metric.Int64Counter

	pendingEventsGauge     metric.Int64Gauge
	pendingExecutionsGauge metric.Int64Gauge
)

// Operation values for integration secret writes.
const (
	IntegrationSecretOperationCreate = "create"
	IntegrationSecretOperationUpdate = "update"
)

func InitMetrics(ctx context.Context) error {
	exporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return err
	}

	res, err := otelResource(ctx)
	if err != nil {
		return err
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
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

	queueWorkerNodesCounter, err = meter.Int64Counter(
		"queue_worker.nodes.total",
		metric.WithDescription("WorkflowNodeQueueWorker node processing outcomes"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	queueWorkerNodeDurationHistogram, err = meter.Float64Histogram(
		"queue_worker.node.duration.seconds",
		metric.WithDescription("Duration of WorkflowNodeQueueWorker node processing"),
		metric.WithUnit("s"),
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

	executorWorkerExecutionsCounter, err = meter.Int64Counter(
		"executor_worker.executions.total",
		metric.WithDescription("WorkflowNodeExecutor execution processing outcomes"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	executorWorkerExecutionDurationHistogram, err = meter.Float64Histogram(
		"executor_worker.execution.duration.seconds",
		metric.WithDescription("Duration of WorkflowNodeExecutor execution processing"),
		metric.WithUnit("s"),
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

	eventWorkerEventsCounter, err = meter.Int64Counter(
		"event_worker.events.total",
		metric.WithDescription("WorkflowEventRouter event processing outcomes"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	eventWorkerEventDurationHistogram, err = meter.Float64Histogram(
		"event_worker.event.duration.seconds",
		metric.WithDescription("Duration of WorkflowEventRouter event processing"),
		metric.WithUnit("s"),
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

	workflowCleanupWorkerCanvasesCountHistogram, err = meter.Int64Histogram(
		"workflow_cleanup_worker.tick.canvases.deleted",
		metric.WithDescription("Number of deleted canvases processed each tick"),
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

	dbPoolConnectionsMaxGauge, err = meter.Int64Gauge(
		"db.pool.connections.max",
		metric.WithDescription("Configured maximum open database connections in the pool"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	dbPoolConnectionsOpenGauge, err = meter.Int64Gauge(
		"db.pool.connections.open",
		metric.WithDescription("Number of open database connections"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	dbPoolConnectionsInUseGauge, err = meter.Int64Gauge(
		"db.pool.connections.in_use",
		metric.WithDescription("Number of database connections currently in use"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	dbPoolConnectionsIdleGauge, err = meter.Int64Gauge(
		"db.pool.connections.idle",
		metric.WithDescription("Number of idle database connections in the pool"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	dbPoolWaitCountCounter, err = meter.Int64Counter(
		"db.pool.wait.count",
		metric.WithDescription("Number of times a request waited for a database connection"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	dbPoolWaitDurationHistogram, err = meter.Float64Histogram(
		"db.pool.wait.duration.seconds",
		metric.WithDescription("Time spent waiting for a database connection"),
		metric.WithUnit("s"),
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

	dbRowsAffectedCounter, err = meter.Int64Counter(
		"db.rows.affected.count",
		metric.WithDescription("Number of database rows affected by operation"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	integrationSecretWritesCounter, err = meter.Int64Counter(
		"integration.secret.writes.total",
		metric.WithDescription("Number of writes to app_installation_secrets, attributed by integration type and operation"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	pendingEventsGauge, err = meter.Int64Gauge(
		"workflow_events.pending.count",
		metric.WithDescription("Current number of pending workflow events"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	pendingExecutionsGauge, err = meter.Int64Gauge(
		"workflow_node_executions.pending.count",
		metric.WithDescription("Current number of pending workflow node executions"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	err = registerDBOperationMetricsCallbacks()
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

func RecordQueueWorkerNodeProcessing(ctx context.Context, d time.Duration, outcome, reason string) {
	if !metricsReady.Load() {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("outcome", outcome),
		attribute.String("reason", reason),
	)

	queueWorkerNodesCounter.Add(ctx, 1, attrs)
	queueWorkerNodeDurationHistogram.Record(
		ctx,
		d.Seconds(),
		metric.WithAttributes(
			attribute.String("outcome", outcome),
		),
	)
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

func RecordExecutorWorkerExecution(ctx context.Context, d time.Duration, outcome, reason, component string) {
	if !metricsReady.Load() {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("outcome", outcome),
		attribute.String("reason", reason),
		attribute.String("component", component),
	)

	executorWorkerExecutionsCounter.Add(ctx, 1, attrs)
	executorWorkerExecutionDurationHistogram.Record(
		ctx,
		d.Seconds(),
		metric.WithAttributes(
			attribute.String("outcome", outcome),
			attribute.String("component", component),
		),
	)
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

func RecordEventWorkerEventProcessing(ctx context.Context, d time.Duration, outcome, reason string) {
	if !metricsReady.Load() {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("outcome", outcome),
		attribute.String("reason", reason),
	)

	eventWorkerEventsCounter.Add(ctx, 1, attrs)
	eventWorkerEventDurationHistogram.Record(
		ctx,
		d.Seconds(),
		metric.WithAttributes(
			attribute.String("outcome", outcome),
		),
	)
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

func RecordWorkflowCleanupWorkerCanvasesCount(ctx context.Context, count int) {
	if !metricsReady.Load() {
		return
	}

	workflowCleanupWorkerCanvasesCountHistogram.Record(ctx, int64(count))
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

func RecordDBPoolStats(ctx context.Context, maxOpen, open, inUse, idle int64) {
	if !metricsReady.Load() {
		return
	}

	dbPoolConnectionsMaxGauge.Record(ctx, maxOpen)
	dbPoolConnectionsOpenGauge.Record(ctx, open)
	dbPoolConnectionsInUseGauge.Record(ctx, inUse)
	dbPoolConnectionsIdleGauge.Record(ctx, idle)
}

func RecordDBPoolWaitCount(ctx context.Context, count int64) {
	if !metricsReady.Load() || count <= 0 {
		return
	}

	dbPoolWaitCountCounter.Add(ctx, count)
}

func RecordDBPoolWaitDuration(ctx context.Context, d time.Duration) {
	if !metricsReady.Load() || d <= 0 {
		return
	}

	dbPoolWaitDurationHistogram.Record(ctx, d.Seconds())
}

func RecordDBRowsAffected(ctx context.Context, count int64, tableName, operation string) {
	if !metricsReady.Load() {
		return
	}

	dbRowsAffectedCounter.Add(
		ctx,
		count,
		metric.WithAttributes(
			attribute.String("table", tableName),
			attribute.String("operation", operation),
		),
	)
}

func RecordIntegrationSecretWrite(ctx context.Context, appName, operation string) {
	if !metricsReady.Load() {
		return
	}

	integrationSecretWritesCounter.Add(
		ctx,
		1,
		metric.WithAttributes(
			attribute.String("app_name", appName),
			attribute.String("operation", operation),
		),
	)
}

func RecordPendingEventsCount(ctx context.Context, count int64) {
	if !metricsReady.Load() {
		return
	}

	pendingEventsGauge.Record(ctx, count)
}

func RecordPendingExecutionsCount(ctx context.Context, count int64) {
	if !metricsReady.Load() {
		return
	}

	pendingExecutionsGauge.Record(ctx, count)
}
