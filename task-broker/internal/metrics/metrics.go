package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/metric"

	"github.com/superplane/runner/shared/telemetry"
)

// BrokerMetrics records task-broker OpenTelemetry metrics (see docs/metrics.md).
type BrokerMetrics struct {
	tasksCreated     metric.Int64Counter
	tasksCompleted   metric.Int64Counter
	tasksUnclaimed   metric.Int64Counter
	leaseReaps       metric.Int64Counter
	taskStartLatency metric.Float64Histogram
}

// New registers broker metric instruments on meter.
func New(meter metric.Meter) (*BrokerMetrics, error) {
	tasksCreated, err := meter.Int64Counter("tasks.created",
		metric.WithDescription("Tasks submitted to the broker"))
	if err != nil {
		return nil, err
	}
	tasksCompleted, err := meter.Int64Counter("tasks.completed",
		metric.WithDescription("Tasks that reached a terminal state"))
	if err != nil {
		return nil, err
	}
	tasksUnclaimed, err := meter.Int64Counter("tasks.unclaimed",
		metric.WithDescription("Claimed tasks re-queued after failed runner delivery"))
	if err != nil {
		return nil, err
	}
	leaseReaps, err := meter.Int64Counter("lease.reaps",
		metric.WithDescription("Expired task leases reaped"))
	if err != nil {
		return nil, err
	}
	taskStartLatency, err := meter.Float64Histogram("task.start_latency",
		metric.WithDescription("Time from task creation until claim"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	return &BrokerMetrics{
		tasksCreated:     tasksCreated,
		tasksCompleted:   tasksCompleted,
		tasksUnclaimed:   tasksUnclaimed,
		leaseReaps:       leaseReaps,
		taskStartLatency: taskStartLatency,
	}, nil
}

func (m *BrokerMetrics) TaskCreated(ctx context.Context, fleetID string) {
	m.tasksCreated.Add(ctx, 1, metric.WithAttributes(telemetry.FleetAttr(fleetID)))
}

func (m *BrokerMetrics) TaskCompleted(ctx context.Context, fleetID, outcome string) {
	m.tasksCompleted.Add(ctx, 1, metric.WithAttributes(
		telemetry.FleetAttr(fleetID),
		telemetry.OutcomeAttr(outcome),
	))
}

func (m *BrokerMetrics) TaskUnclaimed(ctx context.Context, fleetID string) {
	m.tasksUnclaimed.Add(ctx, 1, metric.WithAttributes(telemetry.FleetAttr(fleetID)))
}

func (m *BrokerMetrics) LeaseReaped(ctx context.Context, fleetID string) {
	m.leaseReaps.Add(ctx, 1, metric.WithAttributes(telemetry.FleetAttr(fleetID)))
}

func (m *BrokerMetrics) TaskStartLatency(ctx context.Context, fleetID string, latency time.Duration) {
	m.taskStartLatency.Record(ctx, latency.Seconds(), metric.WithAttributes(telemetry.FleetAttr(fleetID)))
}
