package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/superplanehq/superplane/pkg/taskbroker/shared/telemetry"
)

func testMeter(t *testing.T) (*BrokerMetrics, *metric.ManualReader) {
	t.Helper()
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	m, err := New(provider.Meter("test"))
	require.NoError(t, err)
	return m, reader
}

func collectCounter(t *testing.T, reader *metric.ManualReader, name string) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			sum := m.Data.(metricdata.Sum[int64])
			var total int64
			for _, dp := range sum.DataPoints {
				total += dp.Value
			}
			return total
		}
	}
	t.Fatalf("metric %q not found", name)
	return 0
}

func TestBrokerMetricsTaskCreated(t *testing.T) {
	m, reader := testMeter(t)
	ctx := context.Background()

	m.TaskCreated(ctx, "fleet-a")
	m.TaskCreated(ctx, "fleet-a")

	require.Equal(t, int64(2), collectCounter(t, reader, telemetry.MetricTasksCreated))
}

func TestBrokerMetricsTaskCompleted(t *testing.T) {
	m, reader := testMeter(t)
	ctx := context.Background()

	m.TaskCompleted(ctx, "fleet-a", "succeeded")
	m.TaskCompleted(ctx, "fleet-a", "failed")

	require.Equal(t, int64(2), collectCounter(t, reader, telemetry.MetricTasksCompleted))
}

func TestBrokerMetricsTaskStartLatency(t *testing.T) {
	m, reader := testMeter(t)
	ctx := context.Background()

	m.TaskStartLatency(ctx, "fleet-a", 250*time.Millisecond)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, met := range sm.Metrics {
			if met.Name != telemetry.MetricTaskStartLatency {
				continue
			}
			hist := met.Data.(metricdata.Histogram[float64])
			require.Len(t, hist.DataPoints, 1)
			require.InDelta(t, 0.25, hist.DataPoints[0].Sum, 0.01)
			found = true
		}
	}
	require.True(t, found)
}

func TestBrokerMetricsWebhookDelivered(t *testing.T) {
	m, reader := testMeter(t)
	ctx := context.Background()

	m.WebhookDelivered(ctx, "fleet-a", "succeeded", 100*time.Millisecond)
	m.WebhookDelivered(ctx, "fleet-a", "failed", 200*time.Millisecond)

	require.Equal(t, int64(2), collectCounter(t, reader, telemetry.MetricWebhookDeliveries))
}
