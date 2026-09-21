package telemetry

import (
	"context"
	"testing"
	"time"
)

// Every Record* function in metrics.go guards on metricsReady before touching
// its instrument. The instruments are package-level vars that stay nil until
// InitMetrics runs, so a dropped guard is a nil dereference on a hot path in
// any process that never initialized telemetry (CLI commands, tests, workers
// started before InitMetrics). This pins the guard on all of them at once.
func TestRecordMetrics_NoPanicWhenNotReady(t *testing.T) {
	previous := metricsReady.Load()
	metricsReady.Store(false)
	t.Cleanup(func() {
		metricsReady.Store(previous)
	})

	tests := []struct {
		name   string
		record func(context.Context)
	}{
		{"RecordQueueWorkerTickDuration", func(ctx context.Context) {
			RecordQueueWorkerTickDuration(ctx, time.Second)
		}},
		{"RecordQueueWorkerNodesCount", func(ctx context.Context) {
			RecordQueueWorkerNodesCount(ctx, 1)
		}},
		{"RecordQueueWorkerNodeProcessing", func(ctx context.Context) {
			RecordQueueWorkerNodeProcessing(ctx, time.Second, "outcome", "reason")
		}},
		{"RecordExecutorWorkerTickDuration", func(ctx context.Context) {
			RecordExecutorWorkerTickDuration(ctx, time.Second)
		}},
		{"RecordExecutorWorkerNodesCount", func(ctx context.Context) {
			RecordExecutorWorkerNodesCount(ctx, 1)
		}},
		{"RecordExecutorWorkerExecution", func(ctx context.Context) {
			RecordExecutorWorkerExecution(ctx, time.Second, "outcome", "reason", "component")
		}},
		{"RecordEventWorkerTickDuration", func(ctx context.Context) {
			RecordEventWorkerTickDuration(ctx, time.Second)
		}},
		{"RecordEventWorkerEventsCount", func(ctx context.Context) {
			RecordEventWorkerEventsCount(ctx, 1)
		}},
		{"RecordEventWorkerEventProcessing", func(ctx context.Context) {
			RecordEventWorkerEventProcessing(ctx, time.Second, "outcome", "reason")
		}},
		{"RecordNodeRequestWorkerTickDuration", func(ctx context.Context) {
			RecordNodeRequestWorkerTickDuration(ctx, time.Second)
		}},
		{"RecordNodeRequestWorkerRequestsCount", func(ctx context.Context) {
			RecordNodeRequestWorkerRequestsCount(ctx, 1)
		}},
		{"RecordWebhookProvisionerWorkerTickDuration", func(ctx context.Context) {
			RecordWebhookProvisionerWorkerTickDuration(ctx, time.Second)
		}},
		{"RecordWebhookProvisionerWorkerWebhooksCount", func(ctx context.Context) {
			RecordWebhookProvisionerWorkerWebhooksCount(ctx, 1)
		}},
		{"RecordWebhookProvisionerWorkerWebhookProcessing", func(ctx context.Context) {
			RecordWebhookProvisionerWorkerWebhookProcessing(ctx, time.Second, "outcome", "reason", "app")
		}},
		{"RecordWebhookCleanupWorkerTickDuration", func(ctx context.Context) {
			RecordWebhookCleanupWorkerTickDuration(ctx, time.Second)
		}},
		{"RecordWebhookCleanupWorkerWebhooksCount", func(ctx context.Context) {
			RecordWebhookCleanupWorkerWebhooksCount(ctx, 1)
		}},
		{"RecordWebhookCleanupWorkerWebhookProcessing", func(ctx context.Context) {
			RecordWebhookCleanupWorkerWebhookProcessing(ctx, time.Second, "outcome", "reason")
		}},
		{"RecordWorkflowCleanupWorkerTickDuration", func(ctx context.Context) {
			RecordWorkflowCleanupWorkerTickDuration(ctx, time.Second)
		}},
		{"RecordWorkflowCleanupWorkerCanvasesCount", func(ctx context.Context) {
			RecordWorkflowCleanupWorkerCanvasesCount(ctx, 1)
		}},
		{"RecordRunFinalizerTickDuration", func(ctx context.Context) {
			RecordRunFinalizerTickDuration(ctx, time.Second)
		}},
		{"RecordRunFinalizerRunsCount", func(ctx context.Context) {
			RecordRunFinalizerRunsCount(ctx, 1)
		}},
		{"RecordRunFinalizerRunProcessing", func(ctx context.Context) {
			RecordRunFinalizerRunProcessing(ctx, time.Second, "trigger", "outcome", "reason")
		}},
		{"RecordEmailWorkerEmailProcessing", func(ctx context.Context) {
			RecordEmailWorkerEmailProcessing(ctx, time.Second, "type", "outcome", "reason")
		}},
		{"RecordDBLocksCount", func(ctx context.Context) {
			RecordDBLocksCount(ctx, 1)
		}},
		{"RecordStuckQueueItemsCount", func(ctx context.Context) {
			RecordStuckQueueItemsCount(ctx, 1)
		}},
		{"RecordDBLongQueriesCount", func(ctx context.Context) {
			RecordDBLongQueriesCount(ctx, 1)
		}},
		{"RecordDBPoolStats", func(ctx context.Context) {
			RecordDBPoolStats(ctx, 1, 2, 3, 4)
		}},
		{"RecordDBPoolWaitCount", func(ctx context.Context) {
			RecordDBPoolWaitCount(ctx, 1)
		}},
		{"RecordDBPoolWaitDuration", func(ctx context.Context) {
			RecordDBPoolWaitDuration(ctx, time.Second)
		}},
		{"RecordDBRowsAffected", func(ctx context.Context) {
			RecordDBRowsAffected(ctx, 1, "table", "operation")
		}},
		{"RecordIntegrationSecretWrite", func(ctx context.Context) {
			RecordIntegrationSecretWrite(ctx, "app", IntegrationSecretOperationCreate)
		}},
		{"RecordPendingEventsCount", func(ctx context.Context) {
			RecordPendingEventsCount(ctx, 1)
		}},
		{"RecordPendingExecutionsCount", func(ctx context.Context) {
			RecordPendingExecutionsCount(ctx, 1)
		}},
	}

	ctx := context.Background()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.record(ctx)
		})
	}
}
