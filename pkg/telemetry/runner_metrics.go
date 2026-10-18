package telemetry

import (
	"context"

	"github.com/superplanehq/superplane/pkg/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	runnerCountMetricName                 = "runners.count"
	runnerTaskCountMetricName             = "runner_tasks.count"
	runnerTaskQueueDurationMetricName     = "runner_tasks.queue_wait.seconds"
	runnerTaskExecutionDurationMetricName = "runner_tasks.execution_time.seconds"
	runnerTaskLogSizeMetricName           = "runner_tasks.log_size.bytes"
)

var (
	runnerCountGauge             metric.Int64Gauge
	runnerTaskCountGauge         metric.Int64Gauge
	runnerTaskQueueDuration      metric.Float64Histogram
	runnerTaskExecutionDuration  metric.Float64Histogram
	runnerTaskLogSize            metric.Int64Histogram
	runnerTaskDurationBoundaries = []float64{1, 5, 10, 30, 60, 120, 300, 600, 900, 1800, 3600, 7200, 14400, 28800, 86400}
)

func MetricsEnabled() bool {
	return metricsReady.Load()
}

func registerRunnerMetrics() error {
	var err error

	runnerCountGauge, err = meter.Int64Gauge(
		runnerCountMetricName,
		metric.WithDescription("Current number of non-terminal runners by fleet and state"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	runnerTaskCountGauge, err = meter.Int64Gauge(
		runnerTaskCountMetricName,
		metric.WithDescription("Current number of non-terminal runner tasks by fleet and state"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	runnerTaskQueueDuration, err = meter.Float64Histogram(
		runnerTaskQueueDurationMetricName,
		metric.WithDescription("Time from runner task queueing to reservation"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(runnerTaskDurationBoundaries...),
	)
	if err != nil {
		return err
	}

	runnerTaskExecutionDuration, err = meter.Float64Histogram(
		runnerTaskExecutionDurationMetricName,
		metric.WithDescription("Time from runner task start to terminal state"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(runnerTaskDurationBoundaries...),
	)
	if err != nil {
		return err
	}

	runnerTaskLogSize, err = meter.Int64Histogram(
		runnerTaskLogSizeMetricName,
		metric.WithDescription("Retained uncompressed runner task log size after archiving"),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(0, 1024, 4096, 16384, 65536, 262144, 1048576, 4194304, 8388608, 10485760),
	)
	return err
}

func RecordRunnerTaskQueueDuration(ctx context.Context, task *models.RunnerTask, fleetSlug, fleetScope string) {
	if !metricsReady.Load() || fleetScope == "" || task.ReservedAt == nil || task.QueuedAt.IsZero() {
		return
	}

	duration := task.ReservedAt.Sub(task.QueuedAt)
	if duration < 0 {
		return
	}

	runnerTaskQueueDuration.Record(
		ctx,
		duration.Seconds(),
		metric.WithAttributes(
			attribute.String("fleet_id", fleetSlug),
			attribute.String("fleet_scope", fleetScope),
		),
	)
}

func RecordRunnerTaskExecutionDuration(ctx context.Context, task *models.RunnerTask, fleetSlug, fleetScope string) {
	if !metricsReady.Load() || fleetScope == "" || task.StartedAt == nil || task.FinishedAt == nil {
		return
	}

	duration := task.FinishedAt.Sub(*task.StartedAt)
	if duration < 0 {
		return
	}

	runnerTaskExecutionDuration.Record(
		ctx,
		duration.Seconds(),
		metric.WithAttributes(
			attribute.String("fleet_id", fleetSlug),
			attribute.String("fleet_scope", fleetScope),
			attribute.String("state", task.State),
		),
	)
}

func RecordRunnerTaskLogSize(ctx context.Context, fleetSlug, fleetScope string, size int64, truncated bool) {
	if !metricsReady.Load() || fleetScope == "" || size < 0 {
		return
	}

	runnerTaskLogSize.Record(
		ctx,
		size,
		metric.WithAttributes(
			attribute.String("fleet_id", fleetSlug),
			attribute.String("fleet_scope", fleetScope),
			attribute.Bool("truncated", truncated),
		),
	)
}

func recordRunnerCount(ctx context.Context, count models.FleetStateCount) {
	if !metricsReady.Load() {
		return
	}

	runnerCountGauge.Record(ctx, count.Count, fleetStateAttributes(count))
}

func recordRunnerTaskCount(ctx context.Context, count models.FleetStateCount) {
	if !metricsReady.Load() {
		return
	}

	runnerTaskCountGauge.Record(ctx, count.Count, fleetStateAttributes(count))
}

func fleetStateAttributes(count models.FleetStateCount) metric.RecordOption {
	return metric.WithAttributes(
		attribute.String("fleet_id", count.FleetSlug),
		attribute.String("fleet_slug", count.FleetSlug),
		attribute.String("fleet_scope", count.FleetScope),
		attribute.String("state", count.State),
	)
}
