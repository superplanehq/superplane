package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"gorm.io/datatypes"
)

func TestPeriodicRunnerMetricsReportEveryNonTerminalStateForEachFleet(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	organization, err := models.CreateOrganization("Runner Metrics Organization", "")
	require.NoError(t, err)

	db := database.DB(t.Context())
	createFleet := func(slug string) *models.RunnerFleet {
		fleet := &models.RunnerFleet{
			ID:            uuid.New(),
			Slug:          slug,
			ScopeType:     models.RunnerFleetScopeInstallation,
			Enabled:       true,
			Spec:          datatypes.NewJSONType(models.RunnerFleetSpec{}),
			RunnerVersion: "0.1.0",
		}
		require.NoError(t, fleet.Create(db))
		return fleet
	}
	fleet := createFleet("metrics-amd64")
	secondFleet := createFleet("metrics-arm64")

	now := time.Now()
	for _, state := range []string{
		models.RunnerStatePending,
		models.RunnerStatePending,
		models.RunnerStateBusy,
	} {
		require.NoError(t, db.Create(&models.Runner{
			ID:            uuid.New(),
			FleetID:       fleet.ID,
			State:         state,
			RunnerVersion: fleet.RunnerVersion,
			CreatedAt:     now,
			UpdatedAt:     now,
		}).Error)
	}
	require.NoError(t, db.Create(&models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStateTerminated,
		RunnerVersion: fleet.RunnerVersion,
		CreatedAt:     now,
		UpdatedAt:     now,
		TerminatedAt:  &now,
	}).Error)
	for range 2 {
		require.NoError(t, db.Create(&models.Runner{
			ID:            uuid.New(),
			FleetID:       secondFleet.ID,
			State:         models.RunnerStateIdle,
			RunnerVersion: secondFleet.RunnerVersion,
			CreatedAt:     now,
			UpdatedAt:     now,
		}).Error)
	}

	for _, state := range []string{
		models.RunnerTaskStateQueued,
		models.RunnerTaskStateQueued,
		models.RunnerTaskStateRunning,
	} {
		require.NoError(t, db.Create(&models.RunnerTask{
			ID:                uuid.New(),
			OrganizationID:    organization.ID,
			FleetID:           fleet.ID,
			Backend:           models.RunnerTaskBackendIntegrated,
			State:             state,
			PayloadCiphertext: []byte("ciphertext"),
			QueuedAt:          now,
			CreatedAt:         now,
			UpdatedAt:         now,
		}).Error)
	}
	require.NoError(t, db.Create(&models.RunnerTask{
		ID:                uuid.New(),
		OrganizationID:    organization.ID,
		FleetID:           fleet.ID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateSucceeded,
		PayloadCiphertext: []byte("ciphertext"),
		QueuedAt:          now,
		FinishedAt:        &now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}).Error)
	for range 2 {
		require.NoError(t, db.Create(&models.RunnerTask{
			ID:                uuid.New(),
			OrganizationID:    organization.ID,
			FleetID:           secondFleet.ID,
			Backend:           models.RunnerTaskBackendIntegrated,
			State:             models.RunnerTaskStateReserved,
			PayloadCiphertext: []byte("ciphertext"),
			QueuedAt:          now,
			CreatedAt:         now,
			UpdatedAt:         now,
		}).Error)
	}

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		metricsReady.Store(false)
	})

	meter = provider.Meter("superplane-test")
	require.NoError(t, registerRunnerMetrics())
	metricsReady.Store(true)

	periodic := NewPeriodic(t.Context())
	periodic.reportRunnerCounts()
	periodic.reportRunnerTaskCounts()

	var resourceMetrics metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &resourceMetrics))

	require.Equal(t, map[string]int64{
		models.RunnerStatePending: 2,
		models.RunnerStateIdle:    0,
		models.RunnerStateBusy:    1,
	}, gaugeStateCounts(t, resourceMetrics, runnerCountMetricName, fleet, models.RunnerFleetScopeInstallation))

	require.Equal(t, map[string]int64{
		models.RunnerTaskStateQueued:   2,
		models.RunnerTaskStateReserved: 0,
		models.RunnerTaskStateRunning:  1,
	}, gaugeStateCounts(t, resourceMetrics, runnerTaskCountMetricName, fleet, models.RunnerFleetScopeInstallation))

	require.Equal(t, map[string]int64{
		models.RunnerStatePending: 0,
		models.RunnerStateIdle:    2,
		models.RunnerStateBusy:    0,
	}, gaugeStateCounts(t, resourceMetrics, runnerCountMetricName, secondFleet, models.RunnerFleetScopeInstallation))

	require.Equal(t, map[string]int64{
		models.RunnerTaskStateQueued:   0,
		models.RunnerTaskStateReserved: 2,
		models.RunnerTaskStateRunning:  0,
	}, gaugeStateCounts(t, resourceMetrics, runnerTaskCountMetricName, secondFleet, models.RunnerFleetScopeInstallation))
}

func TestRunnerMetricsSeparateOrganizationsSharingFleetSlug(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	firstOrganization, err := models.CreateOrganization("First shared fleet organization", "")
	require.NoError(t, err)
	secondOrganization, err := models.CreateOrganization("Second shared fleet organization", "")
	require.NoError(t, err)

	db := database.DB(t.Context())
	createFleet := func(organization *models.Organization) *models.RunnerFleet {
		fleet := &models.RunnerFleet{
			ID:            uuid.New(),
			Slug:          "shared-runner-fleet",
			ScopeType:     models.RunnerFleetScopeOrganization,
			ScopeID:       &organization.ID,
			Enabled:       true,
			Spec:          datatypes.NewJSONType(models.RunnerFleetSpec{}),
			RunnerVersion: "0.1.0",
		}
		require.NoError(t, fleet.Create(db))
		return fleet
	}
	firstFleet := createFleet(firstOrganization)
	secondFleet := createFleet(secondOrganization)

	now := time.Now()
	for _, fleet := range []*models.RunnerFleet{firstFleet, secondFleet} {
		require.NoError(t, db.Create(&models.Runner{
			ID:            uuid.New(),
			FleetID:       fleet.ID,
			State:         models.RunnerStatePending,
			RunnerVersion: fleet.RunnerVersion,
			CreatedAt:     now,
			UpdatedAt:     now,
		}).Error)
	}

	reader := setupRunnerMetricsReader(t)
	startedAt := now.Add(-time.Minute)
	firstScope := "organization/" + firstOrganization.Slug
	secondScope := "organization/" + secondOrganization.Slug
	RecordRunnerTaskExecutionDuration(t.Context(), &models.RunnerTask{
		State:      models.RunnerTaskStateSucceeded,
		StartedAt:  &startedAt,
		FinishedAt: &now,
	}, firstFleet.Slug, firstScope)
	RecordRunnerTaskExecutionDuration(t.Context(), &models.RunnerTask{
		State:      models.RunnerTaskStateSucceeded,
		StartedAt:  &startedAt,
		FinishedAt: &now,
	}, secondFleet.Slug, secondScope)
	NewPeriodic(t.Context()).reportRunnerCounts()

	var resourceMetrics metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &resourceMetrics))
	require.Equal(t, int64(1), gaugeStateCounts(t, resourceMetrics, runnerCountMetricName, firstFleet, firstScope)[models.RunnerStatePending])
	require.Equal(t, int64(1), gaugeStateCounts(t, resourceMetrics, runnerCountMetricName, secondFleet, secondScope)[models.RunnerStatePending])

	for _, scope := range resourceMetrics.ScopeMetrics {
		for _, collected := range scope.Metrics {
			if collected.Name != runnerTaskExecutionDurationMetricName {
				continue
			}
			histogram, ok := collected.Data.(metricdata.Histogram[float64])
			require.True(t, ok)
			require.Len(t, histogram.DataPoints, 2)
			seen := map[string]bool{}
			for _, point := range histogram.DataPoints {
				value, found := point.Attributes.Value(attribute.Key("fleet_scope"))
				require.True(t, found)
				seen[value.AsString()] = true
			}
			require.Equal(t, map[string]bool{firstScope: true, secondScope: true}, seen)
			return
		}
	}
	t.Fatalf("metric %q was not collected", runnerTaskExecutionDurationMetricName)
}

func TestRunnerTaskQueueDurationRecordsSecondsByFleet(t *testing.T) {
	reader := setupRunnerMetricsReader(t)

	queuedAt := time.Now().Add(-3 * time.Minute)
	reservedAt := queuedAt.Add(2 * time.Minute)
	fleetSlug := "e1-large-amd64"
	RecordRunnerTaskQueueDuration(t.Context(), &models.RunnerTask{
		QueuedAt:   queuedAt,
		ReservedAt: &reservedAt,
	}, fleetSlug, models.RunnerFleetScopeInstallation)

	var resourceMetrics metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &resourceMetrics))
	for _, scope := range resourceMetrics.ScopeMetrics {
		for _, collected := range scope.Metrics {
			if collected.Name != runnerTaskQueueDurationMetricName {
				continue
			}
			histogram, ok := collected.Data.(metricdata.Histogram[float64])
			require.True(t, ok)
			require.Len(t, histogram.DataPoints, 1)
			point := histogram.DataPoints[0]
			require.Equal(t, uint64(1), point.Count)
			require.InDelta(t, 120, point.Sum, 0.001)
			fleet, found := point.Attributes.Value(attribute.Key("fleet_id"))
			require.True(t, found)
			require.Equal(t, fleetSlug, fleet.AsString())
			fleetScope, found := point.Attributes.Value(attribute.Key("fleet_scope"))
			require.True(t, found)
			require.Equal(t, models.RunnerFleetScopeInstallation, fleetScope.AsString())
			require.Contains(t, point.Bounds, float64(120))
			return
		}
	}
	t.Fatalf("metric %q was not collected", runnerTaskQueueDurationMetricName)
}

func TestRunnerTaskExecutionDurationRecordsSecondsByFleetAndTerminalState(t *testing.T) {
	reader := setupRunnerMetricsReader(t)

	startedAt := time.Now().Add(-5 * time.Minute)
	finishedAt := startedAt.Add(3 * time.Minute)
	lostAt := startedAt.Add(4 * time.Minute)
	fleetSlug := "e1-large-amd64"
	RecordRunnerTaskExecutionDuration(t.Context(), &models.RunnerTask{
		State:      models.RunnerTaskStateSucceeded,
		StartedAt:  &startedAt,
		FinishedAt: &finishedAt,
	}, fleetSlug, models.RunnerFleetScopeInstallation)
	RecordRunnerTaskExecutionDuration(t.Context(), &models.RunnerTask{
		State:      models.RunnerTaskStateLost,
		StartedAt:  &startedAt,
		FinishedAt: &lostAt,
	}, fleetSlug, models.RunnerFleetScopeInstallation)

	var resourceMetrics metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &resourceMetrics))
	for _, scope := range resourceMetrics.ScopeMetrics {
		for _, collected := range scope.Metrics {
			if collected.Name != runnerTaskExecutionDurationMetricName {
				continue
			}
			histogram, ok := collected.Data.(metricdata.Histogram[float64])
			require.True(t, ok)
			require.Len(t, histogram.DataPoints, 2)
			durations := make(map[string]float64, 2)
			for _, point := range histogram.DataPoints {
				require.Equal(t, uint64(1), point.Count)
				fleet, found := point.Attributes.Value(attribute.Key("fleet_id"))
				require.True(t, found)
				require.Equal(t, fleetSlug, fleet.AsString())
				fleetScope, found := point.Attributes.Value(attribute.Key("fleet_scope"))
				require.True(t, found)
				require.Equal(t, models.RunnerFleetScopeInstallation, fleetScope.AsString())
				state, found := point.Attributes.Value(attribute.Key("state"))
				require.True(t, found)
				require.Contains(t, point.Bounds, float64(300))
				durations[state.AsString()] = point.Sum
			}
			require.Equal(t, map[string]float64{
				models.RunnerTaskStateSucceeded: 180,
				models.RunnerTaskStateLost:      240,
			}, durations)
			return
		}
	}
	t.Fatalf("metric %q was not collected", runnerTaskExecutionDurationMetricName)
}

func TestRunnerTaskLogSizeRecordsRetainedBytesByFleet(t *testing.T) {
	reader := setupRunnerMetricsReader(t)
	fleetSlug := "e1-large-amd64"
	RecordRunnerTaskLogSize(t.Context(), fleetSlug, models.RunnerFleetScopeInstallation, 13, false)
	RecordRunnerTaskLogSize(t.Context(), fleetSlug, models.RunnerFleetScopeInstallation, 0, false)

	var resourceMetrics metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &resourceMetrics))
	for _, scope := range resourceMetrics.ScopeMetrics {
		for _, collected := range scope.Metrics {
			if collected.Name != runnerTaskLogSizeMetricName {
				continue
			}
			histogram, ok := collected.Data.(metricdata.Histogram[int64])
			require.True(t, ok)
			require.Len(t, histogram.DataPoints, 1)
			point := histogram.DataPoints[0]
			require.Equal(t, uint64(2), point.Count)
			require.Equal(t, int64(13), point.Sum)
			fleet, found := point.Attributes.Value(attribute.Key("fleet_id"))
			require.True(t, found)
			require.Equal(t, fleetSlug, fleet.AsString())
			fleetScope, found := point.Attributes.Value(attribute.Key("fleet_scope"))
			require.True(t, found)
			require.Equal(t, models.RunnerFleetScopeInstallation, fleetScope.AsString())
			truncated, found := point.Attributes.Value(attribute.Key("truncated"))
			require.True(t, found)
			require.False(t, truncated.AsBool())
			require.Contains(t, point.Bounds, float64(1024))
			return
		}
	}
	t.Fatalf("metric %q was not collected", runnerTaskLogSizeMetricName)
}

func setupRunnerMetricsReader(t *testing.T) *sdkmetric.ManualReader {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		metricsReady.Store(false)
	})
	meter = provider.Meter("superplane-test")
	require.NoError(t, registerRunnerMetrics())
	metricsReady.Store(true)
	return reader
}

func gaugeStateCounts(
	t *testing.T,
	resourceMetrics metricdata.ResourceMetrics,
	metricName string,
	fleet *models.RunnerFleet,
	fleetScope string,
) map[string]int64 {
	t.Helper()

	for _, scopeMetrics := range resourceMetrics.ScopeMetrics {
		for _, collectedMetric := range scopeMetrics.Metrics {
			if collectedMetric.Name != metricName {
				continue
			}

			gauge, ok := collectedMetric.Data.(metricdata.Gauge[int64])
			require.True(t, ok, "%s must be an int64 gauge", metricName)

			counts := make(map[string]int64, len(gauge.DataPoints))
			for _, point := range gauge.DataPoints {
				fleetID, hasFleetID := point.Attributes.Value(attribute.Key("fleet_id"))
				fleetSlug, hasFleetSlug := point.Attributes.Value(attribute.Key("fleet_slug"))
				scope, hasScope := point.Attributes.Value(attribute.Key("fleet_scope"))
				state, hasState := point.Attributes.Value(attribute.Key("state"))
				require.True(t, hasFleetID)
				require.True(t, hasFleetSlug)
				require.True(t, hasScope)
				require.True(t, hasState)
				if fleetID.AsString() != fleet.Slug || scope.AsString() != fleetScope {
					continue
				}
				require.Equal(t, fleet.Slug, fleetSlug.AsString())
				counts[state.AsString()] = point.Value
			}
			return counts
		}
	}

	t.Fatalf("metric %q was not collected", metricName)
	return nil
}
