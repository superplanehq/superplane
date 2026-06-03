package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/metric"

	"github.com/superplane/runner/shared/telemetry"
)

// PoolMetrics records fleet-manager OpenTelemetry metrics (see docs/metrics.md).
type PoolMetrics struct {
	hotInstances           metric.Int64Gauge
	reconcileDuration      metric.Float64Histogram
	instanceSpinupDuration metric.Float64Histogram
}

// New registers pool metric instruments on meter.
func New(meter metric.Meter) (*PoolMetrics, error) {
	hotInstances, err := meter.Int64Gauge("hot.instances",
		metric.WithDescription("Live EC2 instances in the warm pool"))
	if err != nil {
		return nil, err
	}
	reconcileDuration, err := meter.Float64Histogram("reconcile.duration",
		metric.WithDescription("Time taken by one fleet-manager reconcile tick"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	instanceSpinupDuration, err := meter.Float64Histogram("instance.spinup.duration",
		metric.WithDescription("Time from instance request until ready to accept tasks"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	return &PoolMetrics{
		hotInstances:           hotInstances,
		reconcileDuration:      reconcileDuration,
		instanceSpinupDuration: instanceSpinupDuration,
	}, nil
}

func (m *PoolMetrics) SetHotInstances(ctx context.Context, fleetID string, count int) {
	m.hotInstances.Record(ctx, int64(count), metric.WithAttributes(telemetry.FleetAttr(fleetID)))
}

func (m *PoolMetrics) ReconcileDuration(ctx context.Context, fleetID string, duration time.Duration) {
	m.reconcileDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(telemetry.FleetAttr(fleetID)))
}

func (m *PoolMetrics) InstanceSpinupDuration(ctx context.Context, fleetID, phase string, duration time.Duration) {
	m.instanceSpinupDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		telemetry.FleetAttr(fleetID),
		telemetry.PhaseAttr(phase),
	))
}
