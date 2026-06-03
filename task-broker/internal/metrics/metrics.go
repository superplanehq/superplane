package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/metric"

	"github.com/superplane/runner/shared/telemetry"
)

// BrokerMetrics records task-broker OpenTelemetry metrics (see docs/metrics.md).
type BrokerMetrics struct {
	tasksCreated           metric.Int64Counter
	tasksCompleted         metric.Int64Counter
	tasksUnclaimed         metric.Int64Counter
	leaseReaps             metric.Int64Counter
	taskStartLatency       metric.Float64Histogram
	webhookDeliveries      metric.Int64Counter
	webhookDeliveryDur     metric.Float64Histogram
	tasksQueued            metric.Int64Gauge
	tasksClaimed           metric.Int64Gauge
	instanceSpinupDuration metric.Float64Histogram
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
	webhookDeliveries, err := meter.Int64Counter("webhook.deliveries",
		metric.WithDescription("Terminal-state webhook POST attempts that finished"))
	if err != nil {
		return nil, err
	}
	webhookDeliveryDur, err := meter.Float64Histogram("webhook.delivery.duration",
		metric.WithDescription("Wall time for one webhook delivery including retries"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	tasksQueued, err := meter.Int64Gauge("tasks.queued",
		metric.WithDescription("Tasks waiting for a runner to claim them"))
	if err != nil {
		return nil, err
	}
	tasksClaimed, err := meter.Int64Gauge("tasks.claimed",
		metric.WithDescription("Tasks claimed by a runner but not yet terminal"))
	if err != nil {
		return nil, err
	}
	instanceSpinupDuration, err := meter.Float64Histogram("instance.spinup.duration",
		metric.WithDescription("Time from instance request until ready to accept tasks"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	return &BrokerMetrics{
		tasksCreated:           tasksCreated,
		tasksCompleted:         tasksCompleted,
		tasksUnclaimed:         tasksUnclaimed,
		leaseReaps:             leaseReaps,
		taskStartLatency:       taskStartLatency,
		webhookDeliveries:      webhookDeliveries,
		webhookDeliveryDur:     webhookDeliveryDur,
		tasksQueued:            tasksQueued,
		tasksClaimed:           tasksClaimed,
		instanceSpinupDuration: instanceSpinupDuration,
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

func (m *BrokerMetrics) WebhookDelivered(ctx context.Context, fleetID, outcome string, duration time.Duration) {
	attrs := metric.WithAttributes(
		telemetry.FleetAttr(fleetID),
		telemetry.OutcomeAttr(outcome),
	)
	m.webhookDeliveries.Add(ctx, 1, attrs)
	m.webhookDeliveryDur.Record(ctx, duration.Seconds(), attrs)
}

func (m *BrokerMetrics) SetTaskBacklog(ctx context.Context, fleetID string, queued, claimed int) {
	fleet := telemetry.FleetAttr(fleetID)
	m.tasksQueued.Record(ctx, int64(queued), metric.WithAttributes(fleet))
	m.tasksClaimed.Record(ctx, int64(claimed), metric.WithAttributes(fleet))
}

func (m *BrokerMetrics) InstanceSpinupDuration(ctx context.Context, fleetID, phase string, duration time.Duration) {
	m.instanceSpinupDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		telemetry.FleetAttr(fleetID),
		telemetry.PhaseAttr(phase),
	))
}
