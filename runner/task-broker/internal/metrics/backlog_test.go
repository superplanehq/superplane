package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/superplane/runner/shared/models"
	"github.com/superplane/runner/shared/telemetry"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	"github.com/superplane/runner/task-broker/internal/store"
	"github.com/superplane/runner/task-broker/internal/store/testdb"
)

func gaugeValue(t *testing.T, reader *metric.ManualReader, name, fleetID, canvasName, nodeName string) (int64, bool) {
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
				attrs := map[string]string{}
				for _, attr := range dp.Attributes.ToSlice() {
					attrs[string(attr.Key)] = attr.Value.AsString()
				}
				if attrs["fleet_id"] == fleetID &&
					attrs[models.LabelCanvasName] == canvasName &&
					attrs[models.LabelNodeName] == nodeName {
					return dp.Value, true
				}
			}
		}
	}
	return 0, false
}

func floatGaugeValue(t *testing.T, reader *metric.ManualReader, name, fleetID string) (float64, bool) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			gauge := m.Data.(metricdata.Gauge[float64])
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

	create := func(id, fleetID string, status models.TaskStatus, createdAt time.Time, labels map[string]string) {
		t.Helper()
		require.NoError(t, st.CreateTask(ctx, &models.Task{
			ID: id, FleetID: fleetID, Command: []string{"echo"},
			WebhookURL: "https://example.com/hook", Status: status, CreatedAt: createdAt,
			Labels: labels,
		}))
	}
	create("a-q1", "fleet-a", models.StatusQueued, now.Add(-75*time.Second), map[string]string{
		models.LabelCanvasName: "release-train", models.LabelNodeName: "Run tests",
	})
	create("a-q2", "fleet-a", models.StatusQueued, now.Add(-30*time.Second), map[string]string{
		models.LabelCanvasName: "release-train", models.LabelNodeName: "Deploy",
	})
	create("a-c1", "fleet-a", models.StatusClaimed, now.Add(-90*time.Second), map[string]string{
		models.LabelCanvasName: "release-train", models.LabelNodeName: "Run tests",
	})
	create("b-c1", "fleet-b", models.StatusClaimed, now.Add(-60*time.Second), nil)

	m, reader := testMeter(t)
	require.NoError(t, sampleTaskBacklogAt(ctx, st, m, now))

	q, ok := gaugeValue(t, reader, telemetry.MetricTasksQueued, "fleet-a", "release-train", "Run tests")
	require.True(t, ok)
	require.Equal(t, int64(1), q)

	q, ok = gaugeValue(t, reader, telemetry.MetricTasksQueued, "fleet-a", "release-train", "Deploy")
	require.True(t, ok)
	require.Equal(t, int64(1), q)

	c, ok := gaugeValue(t, reader, telemetry.MetricTasksClaimed, "fleet-a", "release-train", "Run tests")
	require.True(t, ok)
	require.Equal(t, int64(1), c)

	q, ok = gaugeValue(t, reader, telemetry.MetricTasksQueued, "fleet-b", "", "")
	require.True(t, ok)
	require.Equal(t, int64(0), q)

	c, ok = gaugeValue(t, reader, telemetry.MetricTasksClaimed, "fleet-b", "", "")
	require.True(t, ok)
	require.Equal(t, int64(1), c)

	oldestAge, ok := floatGaugeValue(t, reader, telemetry.MetricOldestQueuedTaskAge, "fleet-a")
	require.True(t, ok)
	require.InDelta(t, 75, oldestAge, 0.01)

	oldestAge, ok = floatGaugeValue(t, reader, telemetry.MetricOldestQueuedTaskAge, "fleet-b")
	require.True(t, ok)
	require.Equal(t, float64(0), oldestAge)
}

func TestSampleTaskBacklogZerosDrainedSeries(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Now().UTC()

	require.NoError(t, st.CreateFleet(ctx, &brokermodels.Fleet{
		ID: "fleet-a", Provisioner: "local", Arch: "amd64", Size: "local", CreatedAt: now,
	}))
	require.NoError(t, st.CreateTask(ctx, &models.Task{
		ID: "a-q1", FleetID: "fleet-a", Command: []string{"echo"},
		WebhookURL: "https://example.com/hook", Status: models.StatusQueued, CreatedAt: now,
		Labels: map[string]string{models.LabelCanvasName: "c1", models.LabelNodeName: "n1"},
	}))

	m, reader := testMeter(t)
	require.NoError(t, sampleTaskBacklogAt(ctx, st, m, now))
	q, ok := gaugeValue(t, reader, telemetry.MetricTasksQueued, "fleet-a", "c1", "n1")
	require.True(t, ok)
	require.Equal(t, int64(1), q)

	claimed, err := st.ClaimTask(ctx, "r1", "fleet-a", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, claimed)
	_, err = st.CompleteTask(ctx, store.CompleteTaskRequest{ID: claimed.ID, RunnerID: "r1", ExitCode: 0})
	require.NoError(t, err)

	require.NoError(t, sampleTaskBacklogAt(ctx, st, m, now))
	q, ok = gaugeValue(t, reader, telemetry.MetricTasksQueued, "fleet-a", "c1", "n1")
	require.True(t, ok)
	require.Equal(t, int64(0), q)
}

func TestOldestQueuedAgeNeverNegative(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	futureQueuedAt := now.Add(time.Minute)

	require.Equal(t, time.Duration(0), oldestQueuedAge(now, nil))
	require.Equal(t, time.Duration(0), oldestQueuedAge(now, &futureQueuedAt))
}
