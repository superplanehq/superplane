package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/superplanehq/superplane/pkg/runnerbroker/models"
	"github.com/superplanehq/superplane/pkg/runnerbroker/store/testdb"
	brokermodels "github.com/superplanehq/superplane/pkg/runnerbroker/storemodels"
	"github.com/superplanehq/superplane/pkg/runnerbroker/telemetry"
)

func gaugeValue(t *testing.T, reader *metric.ManualReader, name, fleetID string) (int64, bool) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			gauge := m.Data.(metricdata.Gauge[int64])
			for _, dp := range gauge.DataPoints {
				for _, attr := range dp.Attributes.ToSlice() {
					if attr.Key == "fleet_id" && attr.Value.AsString() == fleetID {
						return dp.Value, true
					}
				}
			}
		}
	}
	return 0, false
}

func TestSampleTaskBacklog(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Now().UTC()

	require.NoError(t, st.CreateFleet(ctx, &brokermodels.Fleet{
		ID: "fleet-a", Provisioner: "local", Arch: "amd64", Size: "local", CreatedAt: now,
	}))
	require.NoError(t, st.CreateFleet(ctx, &brokermodels.Fleet{
		ID: "fleet-b", Provisioner: "local", Arch: "arm64", Size: "local", CreatedAt: now,
	}))

	create := func(id, fleetID string, status models.TaskStatus) {
		t.Helper()
		require.NoError(t, st.CreateTask(ctx, &models.Task{
			ID: id, FleetID: fleetID, Command: []string{"echo"},
			WebhookURL: "https://example.com/hook", Status: status, CreatedAt: now,
		}))
	}
	create("a-q1", "fleet-a", models.StatusQueued)
	create("a-q2", "fleet-a", models.StatusQueued)
	create("a-c1", "fleet-a", models.StatusClaimed)
	create("b-c1", "fleet-b", models.StatusClaimed)

	m, reader := testMeter(t)
	require.NoError(t, SampleTaskBacklog(ctx, st, m))

	q, ok := gaugeValue(t, reader, telemetry.MetricTasksQueued, "fleet-a")
	require.True(t, ok)
	require.Equal(t, int64(2), q)

	c, ok := gaugeValue(t, reader, telemetry.MetricTasksClaimed, "fleet-a")
	require.True(t, ok)
	require.Equal(t, int64(1), c)

	q, ok = gaugeValue(t, reader, telemetry.MetricTasksQueued, "fleet-b")
	require.True(t, ok)
	require.Equal(t, int64(0), q)

	c, ok = gaugeValue(t, reader, telemetry.MetricTasksClaimed, "fleet-b")
	require.True(t, ok)
	require.Equal(t, int64(1), c)
}
