package postgres

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const metricsScope = "superplane/runner-active-logs/postgres"

type storeMetrics struct {
	operations metric.Int64Counter
	duration   metric.Float64Histogram
}

func newStoreMetrics(provider metric.MeterProvider) (storeMetrics, error) {
	meter := provider.Meter(metricsScope)
	operations, err := meter.Int64Counter(
		"runner_active_log.postgres.operations.total",
		metric.WithDescription("PostgreSQL runner active log storage operation outcomes"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return storeMetrics{}, err
	}
	duration, err := meter.Float64Histogram(
		"runner_active_log.postgres.operation.duration.seconds",
		metric.WithDescription("Duration of PostgreSQL runner active log storage operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return storeMetrics{}, err
	}
	return storeMetrics{
		operations: operations,
		duration:   duration,
	}, nil
}

func (m storeMetrics) recordOperation(
	ctx context.Context,
	startedAt time.Time,
	operation string,
	err error,
) {
	if m.operations == nil || m.duration == nil {
		return
	}
	outcome := "success"
	if err != nil {
		outcome = "error"
	}
	attributes := metric.WithAttributes(
		attribute.String("operation", operation),
		attribute.String("outcome", outcome),
	)
	m.operations.Add(ctx, 1, attributes)
	m.duration.Record(ctx, time.Since(startedAt).Seconds(), attributes)
}
