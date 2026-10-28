package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestPeriodicVCSCatalogMetricsReportDueJobsAndRecovery(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())
	now := time.Now()
	require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{
		Provider:       models.ProviderGitHub,
		InstallationID: 101,
	}))
	require.NoError(t, models.ReplaceVCSProviderRepositories(db, models.ProviderGitHub, 101, []models.VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/old"},
		{RepositoryID: 202, FullName: "acme/new"},
		{RepositoryID: 203, FullName: "acme/future"},
	}))
	for repositoryID, runAt := range map[int64]time.Time{
		201: now.Add(-10 * time.Minute),
		202: now.Add(-time.Minute),
		203: now.Add(time.Hour),
	} {
		require.NoError(t, models.EnqueueVCSProviderRepositorySync(
			db, models.ProviderGitHub, repositoryID, runAt, models.VCSProviderRepositorySyncPriorityBackground,
		))
	}
	reader := useTestVCSCatalogMeter(t)
	periodic := NewPeriodic(t.Context())

	periodic.reportVCSCatalogSyncQueue()
	collected := collectVCSCatalogMetrics(t, reader)
	assert.Equal(t, int64(2), gaugeValue[int64](t, collected, "vcs_catalog.sync_jobs.due", "background"))
	assert.Equal(t, int64(0), gaugeValue[int64](t, collected, "vcs_catalog.sync_jobs.due", "interactive"))
	assert.InDelta(t, (10 * time.Minute).Seconds(), gaugeValue[float64](t, collected, "vcs_catalog.sync_jobs.oldest_due.age.seconds", "background"), 5)

	require.NoError(t, db.Where("1 = 1").Delete(&models.VCSProviderRepositorySyncJob{}).Error)
	periodic.reportVCSCatalogSyncQueue()
	collected = collectVCSCatalogMetrics(t, reader)
	assert.Equal(t, int64(0), gaugeValue[int64](t, collected, "vcs_catalog.sync_jobs.due", "background"),
		"an empty queue must replace the last reported depth")
	assert.Zero(t, gaugeValue[float64](t, collected, "vcs_catalog.sync_jobs.oldest_due.age.seconds", "background"))
}

func TestRecordGitHubCatalogRateLimitCountsLowResponses(t *testing.T) {
	reader := useTestVCSCatalogMeter(t)

	RecordGitHubCatalogRateLimit(t.Context(), 4000, 5000)
	RecordGitHubCatalogRateLimit(t.Context(), 499, 5000)
	RecordGitHubCatalogRateLimit(t.Context(), 0, 5000)
	RecordGitHubCatalogRateLimit(t.Context(), 0, 0)

	counts := map[string]int64{}
	for _, m := range collectVCSCatalogMetrics(t, reader) {
		if m.Name != "vcs_catalog.github.rate_limit.responses.total" {
			continue
		}
		sum, ok := m.Data.(metricdata.Sum[int64])
		require.True(t, ok)
		for _, point := range sum.DataPoints {
			state, _ := point.Attributes.Value(attribute.Key("state"))
			counts[state.AsString()] = point.Value
		}
	}
	assert.Equal(t, map[string]int64{"ok": 1, "low": 2}, counts, "responses without a rate limit must not be counted")
}

func useTestVCSCatalogMeter(t *testing.T) *sdkmetric.ManualReader {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	previousMeter := meter
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		metricsReady.Store(false)
		meter = previousMeter
	})
	meter = provider.Meter("superplane-test")
	require.NoError(t, registerVCSCatalogMetrics())
	metricsReady.Store(true)
	return reader
}

func collectVCSCatalogMetrics(t *testing.T, reader *sdkmetric.ManualReader) []metricdata.Metrics {
	t.Helper()
	var collected metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &collected))
	var metrics []metricdata.Metrics
	for _, scope := range collected.ScopeMetrics {
		metrics = append(metrics, scope.Metrics...)
	}
	return metrics
}

func gaugeValue[N int64 | float64](t *testing.T, metrics []metricdata.Metrics, name, priority string) N {
	t.Helper()
	for _, m := range metrics {
		if m.Name != name {
			continue
		}
		gauge, ok := m.Data.(metricdata.Gauge[N])
		require.True(t, ok)
		for _, point := range gauge.DataPoints {
			value, _ := point.Attributes.Value(attribute.Key("priority"))
			if value.AsString() == priority {
				return point.Value
			}
		}
	}
	t.Fatalf("missing %s for %s priority", name, priority)
	return 0
}
