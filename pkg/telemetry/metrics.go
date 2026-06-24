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
	queueWorkerNodesCountGauge       metric.Int64Gauge
	queueWorkerNodesCounter          metric.Int64Counter
	queueWorkerNodeDurationHistogram metric.Float64Histogram
	queueWorkerStuckItems            metric.Int64Histogram

	executorWorkerTickHistogram              metric.Float64Histogram
	executorWorkerNodesCountGauge            metric.Int64Gauge
	executorWorkerExecutionsCounter          metric.Int64Counter
	executorWorkerExecutionDurationHistogram metric.Float64Histogram

	eventWorkerTickHistogram          metric.Float64Histogram
	eventWorkerEventsCountGauge       metric.Int64Gauge
	eventWorkerEventsCounter          metric.Int64Counter
	eventWorkerEventDurationHistogram metric.Float64Histogram

	nodeRequestWorkerTickHistogram          metric.Float64Histogram
	nodeRequestWorkerRequestsCountHistogram metric.Int64Histogram

	webhookProvisionerWorkerTickHistogram            metric.Float64Histogram
	webhookProvisionerWorkerWebhooksCountGauge       metric.Int64Gauge
	webhookProvisionerWorkerWebhooksCounter          metric.Int64Counter
	webhookProvisionerWorkerWebhookDurationHistogram metric.Float64Histogram

	webhookCleanupWorkerTickHistogram            metric.Float64Histogram
	webhookCleanupWorkerWebhooksCountHistogram   metric.Int64Histogram
	webhookCleanupWorkerWebhooksCounter          metric.Int64Counter
	webhookCleanupWorkerWebhookDurationHistogram metric.Float64Histogram

	workflowCleanupWorkerTickHistogram          metric.Float64Histogram
	workflowCleanupWorkerCanvasesCountHistogram metric.Int64Histogram

	runFinalizerTickHistogram        metric.Float64Histogram
	runFinalizerRunsCountGauge       metric.Int64Gauge
	runFinalizerRunsCounter          metric.Int64Counter
	runFinalizerRunDurationHistogram metric.Float64Histogram

	emailWorkerEmailsCounter        metric.Int64Counter
	emailWorkerEmailDurationSeconds metric.Float64Histogram

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

// durationSecondsHistogramBoundaries matches Prometheus DefBuckets and is
// appropriate for latency histograms recorded in seconds. The OTel SDK default
// boundaries assume milliseconds, which collapses sub-second values into the
// (0, 5] bucket and makes histogram_quantile estimates misleading.
var durationSecondsHistogramBoundaries = []float64{
	0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

func durationSecondsHistogramView() sdkmetric.Option {
	return sdkmetric.WithView(sdkmetric.NewView(
		sdkmetric.Instrument{
			Name: "*duration.seconds",
			Unit: "s",
		},
		sdkmetric.Stream{
			Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
				Boundaries: durationSecondsHistogramBoundaries,
			},
		},
	))
}

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
		durationSecondsHistogramView(),
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

	queueWorkerNodesCountGauge, err = meter.Int64Gauge(
		"queue_worker.tick.nodes.ready",
		metric.WithDescription("Number of workflow nodes ready to be processed on the last queue worker tick"),
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

	executorWorkerNodesCountGauge, err = meter.Int64Gauge(
		"executor_worker.tick.nodes.pending",
		metric.WithDescription("Number of pending workflow node executions on the last executor worker tick"),
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

	eventWorkerEventsCountGauge, err = meter.Int64Gauge(
		"event_worker.tick.events.pending",
		metric.WithDescription("Number of pending workflow events on the last event worker tick"),
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

	webhookProvisionerWorkerTickHistogram, err = meter.Float64Histogram(
		"webhook_provisioner_worker.tick.duration.seconds",
		metric.WithDescription("Duration of each WebhookProvisioner tick"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	webhookProvisionerWorkerWebhooksCountGauge, err = meter.Int64Gauge(
		"webhook_provisioner_worker.tick.webhooks.pending",
		metric.WithDescription("Number of pending webhooks on the last webhook provisioner tick"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	webhookProvisionerWorkerWebhooksCounter, err = meter.Int64Counter(
		"webhook_provisioner_worker.webhooks.total",
		metric.WithDescription("WebhookProvisioner webhook processing outcomes"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	webhookProvisionerWorkerWebhookDurationHistogram, err = meter.Float64Histogram(
		"webhook_provisioner_worker.webhook.duration.seconds",
		metric.WithDescription("Duration of WebhookProvisioner webhook processing"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	webhookCleanupWorkerTickHistogram, err = meter.Float64Histogram(
		"webhook_cleanup_worker.tick.duration.seconds",
		metric.WithDescription("Duration of each WebhookCleanupWorker tick"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	webhookCleanupWorkerWebhooksCountHistogram, err = meter.Int64Histogram(
		"webhook_cleanup_worker.tick.webhooks.pending",
		metric.WithDescription("Number of deleted webhooks awaiting cleanup each tick"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	webhookCleanupWorkerWebhooksCounter, err = meter.Int64Counter(
		"webhook_cleanup_worker.webhooks.total",
		metric.WithDescription("WebhookCleanupWorker webhook processing outcomes"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	webhookCleanupWorkerWebhookDurationHistogram, err = meter.Float64Histogram(
		"webhook_cleanup_worker.webhook.duration.seconds",
		metric.WithDescription("Duration of WebhookCleanupWorker webhook processing"),
		metric.WithUnit("s"),
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

	runFinalizerTickHistogram, err = meter.Float64Histogram(
		"run_finalizer.tick.duration.seconds",
		metric.WithDescription("Duration of each RunFinalizer sweep tick"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	runFinalizerRunsCountGauge, err = meter.Int64Gauge(
		"run_finalizer.tick.runs.started",
		metric.WithDescription("Number of started workflow runs on the last run finalizer sweep tick"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	runFinalizerRunsCounter, err = meter.Int64Counter(
		"run_finalizer.runs.total",
		metric.WithDescription("RunFinalizer run processing outcomes"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	runFinalizerRunDurationHistogram, err = meter.Float64Histogram(
		"run_finalizer.run.duration.seconds",
		metric.WithDescription("Duration of RunFinalizer run processing"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	emailWorkerEmailsCounter, err = meter.Int64Counter(
		"email_worker.emails.total",
		metric.WithDescription("Email worker processing outcomes"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	emailWorkerEmailDurationSeconds, err = meter.Float64Histogram(
		"email_worker.email.duration.seconds",
		metric.WithDescription("Duration of email worker processing"),
		metric.WithUnit("s"),
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

	queueWorkerNodesCountGauge.Record(ctx, int64(count))
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

	executorWorkerNodesCountGauge.Record(ctx, int64(count))
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

	eventWorkerEventsCountGauge.Record(ctx, int64(count))
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

func RecordWebhookProvisionerWorkerTickDuration(ctx context.Context, d time.Duration) {
	if !metricsReady.Load() {
		return
	}

	webhookProvisionerWorkerTickHistogram.Record(ctx, d.Seconds())
}

func RecordWebhookProvisionerWorkerWebhooksCount(ctx context.Context, count int) {
	if !metricsReady.Load() {
		return
	}

	webhookProvisionerWorkerWebhooksCountGauge.Record(ctx, int64(count))
}

func RecordWebhookProvisionerWorkerWebhookProcessing(ctx context.Context, d time.Duration, outcome, reason, appName string) {
	if !metricsReady.Load() {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("outcome", outcome),
		attribute.String("reason", reason),
		attribute.String("app_name", appName),
	)

	webhookProvisionerWorkerWebhooksCounter.Add(ctx, 1, attrs)
	webhookProvisionerWorkerWebhookDurationHistogram.Record(
		ctx,
		d.Seconds(),
		metric.WithAttributes(
			attribute.String("outcome", outcome),
			attribute.String("app_name", appName),
		),
	)
}

func RecordWebhookCleanupWorkerTickDuration(ctx context.Context, d time.Duration) {
	if !metricsReady.Load() {
		return
	}

	webhookCleanupWorkerTickHistogram.Record(ctx, d.Seconds())
}

func RecordWebhookCleanupWorkerWebhooksCount(ctx context.Context, count int) {
	if !metricsReady.Load() {
		return
	}

	webhookCleanupWorkerWebhooksCountHistogram.Record(ctx, int64(count))
}

func RecordWebhookCleanupWorkerWebhookProcessing(ctx context.Context, d time.Duration, outcome, reason string) {
	if !metricsReady.Load() {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("outcome", outcome),
		attribute.String("reason", reason),
	)

	webhookCleanupWorkerWebhooksCounter.Add(ctx, 1, attrs)
	webhookCleanupWorkerWebhookDurationHistogram.Record(
		ctx,
		d.Seconds(),
		metric.WithAttributes(
			attribute.String("outcome", outcome),
		),
	)
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

func RecordRunFinalizerTickDuration(ctx context.Context, d time.Duration) {
	if !metricsReady.Load() {
		return
	}

	runFinalizerTickHistogram.Record(ctx, d.Seconds())
}

func RecordRunFinalizerRunsCount(ctx context.Context, count int) {
	if !metricsReady.Load() {
		return
	}

	runFinalizerRunsCountGauge.Record(ctx, int64(count))
}

func RecordRunFinalizerRunProcessing(ctx context.Context, d time.Duration, trigger, outcome, reason string) {
	if !metricsReady.Load() {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("trigger", trigger),
		attribute.String("outcome", outcome),
		attribute.String("reason", reason),
	)

	runFinalizerRunsCounter.Add(ctx, 1, attrs)
	runFinalizerRunDurationHistogram.Record(
		ctx,
		d.Seconds(),
		metric.WithAttributes(
			attribute.String("trigger", trigger),
			attribute.String("outcome", outcome),
		),
	)
}

func RecordEmailWorkerEmailProcessing(ctx context.Context, d time.Duration, emailType, outcome, reason string) {
	if !metricsReady.Load() {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("email_type", emailType),
		attribute.String("outcome", outcome),
		attribute.String("reason", reason),
	)

	emailWorkerEmailsCounter.Add(ctx, 1, attrs)
	emailWorkerEmailDurationSeconds.Record(
		ctx,
		d.Seconds(),
		metric.WithAttributes(
			attribute.String("email_type", emailType),
			attribute.String("outcome", outcome),
		),
	)
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
