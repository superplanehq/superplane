package telemetry

// OpenTelemetry instrument names for the runner platform (see docs/metrics.md).
// All metrics use the runner. prefix so they stay distinct in shared backends like Dash0.
const (
	MetricTasksCreated           = "runner.tasks.created"
	MetricTasksCompleted         = "runner.tasks.completed"
	MetricTaskStartLatency       = "runner.task.start_latency"
	MetricTasksQueued            = "runner.tasks.queued"
	MetricTasksClaimed           = "runner.tasks.claimed"
	MetricTasksUnclaimed         = "runner.tasks.unclaimed"
	MetricLeaseReaps             = "runner.lease.reaps"
	MetricWebhookDeliveries      = "runner.webhook.deliveries"
	MetricWebhookDeliveryDur     = "runner.webhook.delivery.duration"
	MetricPoolHotInstances       = "runner.pool.hot_instances"
	MetricPoolReconcileDuration  = "runner.pool.reconcile.duration"
	MetricInstanceSpinupDuration = "runner.instance.spinup.duration"
)
