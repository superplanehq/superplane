package telemetry

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// longDurationSecondsBoundaries cover runner occupancy and task waits.
// The latency view for *.duration.seconds stops at 10s, which collapses
// minute- and hour-scale samples into one bucket.
var longDurationSecondsBoundaries = []float64{
	1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600, 10800, 21600, 43200, 86400,
}

// logByteBoundaries cover retained task logs up to the 10 MiB active-log limit.
var logByteBoundaries = []float64{
	1024,
	4 * 1024,
	16 * 1024,
	64 * 1024,
	256 * 1024,
	1024 * 1024,
	4 * 1024 * 1024,
	10 * 1024 * 1024,
}

var (
	runnerFleetCountGauge       metric.Int64Gauge
	runnerStateOccupancySeconds metric.Float64Histogram
	runnerTaskQueueWaitSeconds  metric.Float64Histogram
	runnerTaskRunSeconds        metric.Float64Histogram
	runnerTaskLogBytes          metric.Int64Histogram
)

func registerRunnerMetrics() error {
	var err error

	runnerFleetCountGauge, err = meter.Int64Gauge(
		"runner.fleet.count",
		metric.WithDescription("Idle and busy runners in each installation fleet"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	runnerStateOccupancySeconds, err = meter.Float64Histogram(
		"runner.state.occupancy.seconds",
		metric.WithDescription("Time a runner spent idle or busy before leaving that state"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	runnerTaskQueueWaitSeconds, err = meter.Float64Histogram(
		"runner_task.queue.wait.seconds",
		metric.WithDescription("Time from enqueue until the task starts or is abandoned before it starts"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	runnerTaskRunSeconds, err = meter.Float64Histogram(
		"runner_task.run.seconds",
		metric.WithDescription("Time from task start until the task reaches a terminal state"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	runnerTaskLogBytes, err = meter.Int64Histogram(
		"runner_task.log.bytes",
		metric.WithDescription("Uncompressed task log size when the log is compacted"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return err
	}

	models.SetRunnerMetrics(models.RunnerMetrics{
		StateOccupancy: func(labels models.RunnerFleetLabels, state string, d time.Duration) {
			RecordRunnerStateOccupancy(context.Background(), labels.FleetSlug, state, d)
		},
		TaskQueueWait: func(labels models.RunnerFleetLabels, outcome string, d time.Duration) {
			RecordRunnerTaskQueueWait(context.Background(), labels.FleetSlug, outcome, d)
		},
		TaskRun: func(labels models.RunnerFleetLabels, state string, d time.Duration) {
			RecordRunnerTaskRun(context.Background(), labels.FleetSlug, state, d)
		},
	})
	return nil
}

func RecordRunnerFleetCount(ctx context.Context, fleetSlug, state string, count int64) {
	if !metricsReady.Load() || runnerFleetCountGauge == nil || fleetSlug == "" {
		return
	}

	runnerFleetCountGauge.Record(ctx, count, fleetAttributes(fleetSlug,
		attribute.String("state", state),
	))
}

func RecordRunnerStateOccupancy(ctx context.Context, fleetSlug, state string, d time.Duration) {
	if !metricsReady.Load() || runnerStateOccupancySeconds == nil || fleetSlug == "" || d < 0 {
		return
	}

	runnerStateOccupancySeconds.Record(ctx, d.Seconds(), fleetAttributes(fleetSlug,
		attribute.String("state", state),
	))
}

func RecordRunnerTaskQueueWait(ctx context.Context, fleetSlug, outcome string, d time.Duration) {
	if !metricsReady.Load() || runnerTaskQueueWaitSeconds == nil || fleetSlug == "" || d < 0 {
		return
	}

	runnerTaskQueueWaitSeconds.Record(ctx, d.Seconds(), fleetAttributes(fleetSlug,
		attribute.String("outcome", outcome),
	))
}

func RecordRunnerTaskRun(ctx context.Context, fleetSlug, state string, d time.Duration) {
	if !metricsReady.Load() || runnerTaskRunSeconds == nil || fleetSlug == "" || d < 0 {
		return
	}

	runnerTaskRunSeconds.Record(ctx, d.Seconds(), fleetAttributes(fleetSlug,
		attribute.String("state", state),
	))
}

func RecordRunnerTaskLogSize(ctx context.Context, fleetSlug string, size int64) {
	if !metricsReady.Load() || runnerTaskLogBytes == nil || fleetSlug == "" || size < 0 {
		return
	}

	runnerTaskLogBytes.Record(ctx, size, fleetAttributes(fleetSlug))
}

func fleetAttributes(fleetSlug string, extra ...attribute.KeyValue) metric.MeasurementOption {
	attrs := make([]attribute.KeyValue, 0, 1+len(extra))
	attrs = append(attrs, attribute.String("fleet", fleetSlug))
	attrs = append(attrs, extra...)
	return metric.WithAttributes(attrs...)
}
