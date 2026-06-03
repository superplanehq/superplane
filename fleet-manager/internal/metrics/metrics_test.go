package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/superplane/runner/shared/telemetry"
)

func testMeter(t *testing.T) (*PoolMetrics, *metric.ManualReader) {
	t.Helper()
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	m, err := New(provider.Meter("test"))
	require.NoError(t, err)
	return m, reader
}

func TestPoolMetricsSetHotInstances(t *testing.T) {
	m, reader := testMeter(t)
	ctx := context.Background()

	m.SetHotInstances(ctx, "fleet-a", 3)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, met := range sm.Metrics {
			if met.Name != telemetry.MetricPoolHotInstances {
				continue
			}
			gauge := met.Data.(metricdata.Gauge[int64])
			require.Len(t, gauge.DataPoints, 1)
			require.Equal(t, int64(3), gauge.DataPoints[0].Value)
			found = true
		}
	}
	require.True(t, found)
}

func TestPoolMetricsReconcileDuration(t *testing.T) {
	m, reader := testMeter(t)
	ctx := context.Background()

	m.ReconcileDuration(ctx, "fleet-a", 500*time.Millisecond)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, met := range sm.Metrics {
			if met.Name != telemetry.MetricPoolReconcileDuration {
				continue
			}
			hist := met.Data.(metricdata.Histogram[float64])
			require.Len(t, hist.DataPoints, 1)
			require.InDelta(t, 0.5, hist.DataPoints[0].Sum, 0.01)
			found = true
		}
	}
	require.True(t, found)
}
