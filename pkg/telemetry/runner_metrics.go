package telemetry

import (
	"context"

	"github.com/superplanehq/superplane/pkg/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	runnerCountMetricName     = "runners.count"
	runnerTaskCountMetricName = "runner_tasks.count"
)

var (
	runnerCountGauge     metric.Int64Gauge
	runnerTaskCountGauge metric.Int64Gauge
)

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
	return err
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
		attribute.String("fleet_id", count.FleetID.String()),
		attribute.String("fleet_slug", count.FleetSlug),
		attribute.String("state", count.State),
	)
}
