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
	fleet := &models.RunnerFleet{
		ID:            uuid.New(),
		Slug:          "metrics-amd64",
		ScopeType:     models.RunnerFleetScopeInstallation,
		Enabled:       true,
		Spec:          datatypes.NewJSONType(models.RunnerFleetSpec{}),
		RunnerVersion: "0.1.0",
	}
	require.NoError(t, fleet.Create(db))

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
	}, gaugeStateCounts(t, resourceMetrics, runnerCountMetricName, fleet))

	require.Equal(t, map[string]int64{
		models.RunnerTaskStateQueued:   2,
		models.RunnerTaskStateReserved: 0,
		models.RunnerTaskStateRunning:  1,
	}, gaugeStateCounts(t, resourceMetrics, runnerTaskCountMetricName, fleet))
}

func gaugeStateCounts(
	t *testing.T,
	resourceMetrics metricdata.ResourceMetrics,
	metricName string,
	fleet *models.RunnerFleet,
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
				state, hasState := point.Attributes.Value(attribute.Key("state"))
				require.True(t, hasFleetID)
				require.True(t, hasFleetSlug)
				require.True(t, hasState)
				require.Equal(t, fleet.ID.String(), fleetID.AsString())
				require.Equal(t, fleet.Slug, fleetSlug.AsString())
				counts[state.AsString()] = point.Value
			}
			return counts
		}
	}

	t.Fatalf("metric %q was not collected", metricName)
	return nil
}
