package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"gorm.io/datatypes"
)

func TestListRunnerFleetCountsIgnoresOrganizationFleets(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()
	organization := &models.Organization{
		ID:        uuid.New(),
		Name:      "Runner Fleet Counts",
		Slug:      uuid.NewString(),
		CreatedAt: &now,
		UpdatedAt: &now,
	}
	require.NoError(t, db.Create(organization).Error)

	installation := &models.RunnerFleet{
		ID:            uuid.New(),
		Slug:          "count-installation",
		ScopeType:     models.RunnerFleetScopeInstallation,
		Enabled:       true,
		Spec:          datatypes.NewJSONType(models.RunnerFleetSpec{}),
		RunnerVersion: "0.1.0",
	}
	empty := &models.RunnerFleet{
		ID:            uuid.New(),
		Slug:          "count-empty",
		ScopeType:     models.RunnerFleetScopeInstallation,
		Enabled:       true,
		Spec:          datatypes.NewJSONType(models.RunnerFleetSpec{}),
		RunnerVersion: "0.1.0",
	}
	organizationFleet := &models.RunnerFleet{
		ID:            uuid.New(),
		Slug:          "count-organization",
		ScopeType:     models.RunnerFleetScopeOrganization,
		ScopeID:       &organization.ID,
		Enabled:       true,
		Spec:          datatypes.NewJSONType(models.RunnerFleetSpec{}),
		RunnerVersion: "0.1.0",
	}
	require.NoError(t, installation.Create(db))
	require.NoError(t, empty.Create(db))
	require.NoError(t, organizationFleet.Create(db))

	require.NoError(t, db.Create(&[]models.Runner{
		{ID: uuid.New(), FleetID: installation.ID, State: models.RunnerStateIdle, RunnerVersion: "0.1.0", CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), FleetID: installation.ID, State: models.RunnerStateBusy, RunnerVersion: "0.1.0", CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), FleetID: installation.ID, State: models.RunnerStateTerminated, RunnerVersion: "0.1.0", TerminatedAt: &now, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), FleetID: organizationFleet.ID, State: models.RunnerStateBusy, RunnerVersion: "0.1.0", CreatedAt: now, UpdatedAt: now},
	}).Error)

	rows, err := listRunnerFleetCounts()
	require.NoError(t, err)
	bySlug := map[string]runnerFleetCount{}
	for _, row := range rows {
		bySlug[row.FleetSlug] = row
	}

	assert.Equal(t, int64(1), bySlug["count-installation"].IdleCount)
	assert.Equal(t, int64(1), bySlug["count-installation"].BusyCount)
	assert.Equal(t, int64(0), bySlug["count-empty"].IdleCount)
	assert.Equal(t, int64(0), bySlug["count-empty"].BusyCount)
	_, foundOrganizationFleet := bySlug["count-organization"]
	assert.False(t, foundOrganizationFleet)

	require.NoError(t, db.Delete(empty).Error)
	rows, err = listRunnerFleetCounts()
	require.NoError(t, err)
	for _, row := range rows {
		assert.NotEqual(t, "count-empty", row.FleetSlug)
	}
}

func TestRecordRunnerMetricsWhenReady(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	previousMeter := meter
	previousReady := metricsReady.Load()
	previousHook := models.SetRunnerMetrics(models.RunnerMetrics{})
	meter = provider.Meter("superplane-test")
	t.Cleanup(func() {
		meter = previousMeter
		metricsReady.Store(previousReady)
		models.SetRunnerMetrics(previousHook)
		_ = provider.Shutdown(context.Background())
	})

	require.NoError(t, registerRunnerMetrics())
	metricsReady.Store(true)

	ctx := context.Background()
	RecordRunnerFleetCount(ctx, "e1-large-amd64", models.RunnerStateIdle, 2)
	RecordRunnerFleetCount(ctx, "e1-large-amd64", models.RunnerStateBusy, 1)
	RecordRunnerStateOccupancy(ctx, "e1-large-amd64", models.RunnerStateIdle, 30*time.Second)
	RecordRunnerTaskQueueWait(ctx, "e1-large-amd64", models.RunnerQueueWaitStarted, time.Minute)
	RecordRunnerTaskRun(ctx, "e1-large-amd64", models.RunnerTaskStateSucceeded, 2*time.Minute)
	RecordRunnerTaskLogSize(ctx, "e1-large-amd64", 13)

	var collected metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &collected))

	assert.Equal(t, int64(2), gaugeValue(t, collected, "runner.fleet.count", "idle"))
	assert.Equal(t, int64(1), gaugeValue(t, collected, "runner.fleet.count", "busy"))
	assert.InDelta(t, 30, histogramSum(t, collected, "runner.state.occupancy.seconds"), 0.001)
	assert.InDelta(t, 60, histogramSum(t, collected, "runner_task.queue.wait.seconds"), 0.001)
	assert.InDelta(t, 120, histogramSum(t, collected, "runner_task.run.seconds"), 0.001)
	assert.Equal(t, int64(13), intHistogramSum(t, collected, "runner_task.log.bytes"))
}

func TestRecordRunnerMetricsNoPanicWhenNotReady(t *testing.T) {
	metricsReady.Store(false)
	ctx := context.Background()
	RecordRunnerFleetCount(ctx, "e1-large-amd64", models.RunnerStateIdle, 1)
	RecordRunnerStateOccupancy(ctx, "e1-large-amd64", models.RunnerStateBusy, time.Second)
	RecordRunnerTaskQueueWait(ctx, "e1-large-amd64", models.RunnerQueueWaitCanceled, time.Second)
	RecordRunnerTaskRun(ctx, "e1-large-amd64", models.RunnerTaskStateFailed, time.Second)
	RecordRunnerTaskLogSize(ctx, "e1-large-amd64", 4)
}

func gaugeValue(t *testing.T, collected metricdata.ResourceMetrics, name, state string) int64 {
	t.Helper()
	for _, point := range metricDataPoints(t, collected, name) {
		gauge, ok := point.(metricdata.DataPoint[int64])
		if !ok || attributeString(gauge.Attributes, "state") != state {
			continue
		}
		return gauge.Value
	}
	t.Fatalf("gauge %s state %s not found", name, state)
	return 0
}

func histogramSum(t *testing.T, collected metricdata.ResourceMetrics, name string) float64 {
	t.Helper()
	for _, point := range metricDataPoints(t, collected, name) {
		histogram, ok := point.(metricdata.HistogramDataPoint[float64])
		if ok {
			return histogram.Sum
		}
	}
	t.Fatalf("histogram %s not found", name)
	return 0
}

func intHistogramSum(t *testing.T, collected metricdata.ResourceMetrics, name string) int64 {
	t.Helper()
	for _, point := range metricDataPoints(t, collected, name) {
		histogram, ok := point.(metricdata.HistogramDataPoint[int64])
		if ok {
			return histogram.Sum
		}
	}
	t.Fatalf("histogram %s not found", name)
	return 0
}

func metricDataPoints(t *testing.T, collected metricdata.ResourceMetrics, name string) []any {
	t.Helper()
	for _, scope := range collected.ScopeMetrics {
		for _, item := range scope.Metrics {
			if item.Name != name {
				continue
			}
			switch data := item.Data.(type) {
			case metricdata.Gauge[int64]:
				points := make([]any, len(data.DataPoints))
				for i := range data.DataPoints {
					points[i] = data.DataPoints[i]
				}
				return points
			case metricdata.Histogram[float64]:
				points := make([]any, len(data.DataPoints))
				for i := range data.DataPoints {
					points[i] = data.DataPoints[i]
				}
				return points
			case metricdata.Histogram[int64]:
				points := make([]any, len(data.DataPoints))
				for i := range data.DataPoints {
					points[i] = data.DataPoints[i]
				}
				return points
			default:
				t.Fatalf("metric %s has unexpected data %T", name, item.Data)
			}
		}
	}
	t.Fatalf("metric %s not found", name)
	return nil
}

func attributeString(set attribute.Set, key string) string {
	value, ok := set.Value(attribute.Key(key))
	if !ok {
		return ""
	}
	return value.AsString()
}
