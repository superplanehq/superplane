package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestPeriodicForgeMetricsTrackMissingDeliveriesAndRecovery(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	now := time.Now()
	db := database.DB(t.Context())
	for _, row := range []models.BitbucketForgeInstallation{
		{InstallationID: uuid.NewString(), LastDeliveryAt: now, TokenExpiresAt: now.Add(time.Hour), SystemToken: []byte("encrypted")},
		{InstallationID: uuid.NewString(), LastDeliveryAt: now.Add(-16 * time.Minute), TokenExpiresAt: now.Add(time.Hour), SystemToken: []byte("encrypted")},
		{InstallationID: uuid.NewString(), LastDeliveryAt: now, TokenExpiresAt: now.Add(-time.Minute), SystemToken: []byte("encrypted")},
		{InstallationID: uuid.NewString(), LastDeliveryAt: now},
		{InstallationID: uuid.NewString(), UninstalledAt: &now},
	} {
		require.NoError(t, db.Create(&row).Error)
	}
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	previousMeter := meter
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		metricsReady.Store(false)
		meter = previousMeter
	})
	meter = provider.Meter("superplane-test")
	require.NoError(t, registerBitbucketForgeMetrics())
	metricsReady.Store(true)

	periodic := NewPeriodic(t.Context())
	assertCount := func(expected int64) {
		t.Helper()
		periodic.reportBitbucketForgeDeliveries()
		var collected metricdata.ResourceMetrics
		require.NoError(t, reader.Collect(t.Context(), &collected))
		for _, scope := range collected.ScopeMetrics {
			for _, m := range scope.Metrics {
				if m.Name != "bitbucket.forge.installations.stale" {
					continue
				}
				gauge, ok := m.Data.(metricdata.Gauge[int64])
				require.True(t, ok)
				require.Len(t, gauge.DataPoints, 1)
				require.Equal(t, expected, gauge.DataPoints[0].Value)
				return
			}
		}
		t.Fatal("missing Forge stale installation metric")
	}
	assertCount(3)
	require.NoError(t, db.Model(&models.BitbucketForgeInstallation{}).Where("uninstalled_at IS NULL").Updates(map[string]any{
		"last_delivery_at": now, "token_expires_at": now.Add(time.Hour), "system_token": []byte("encrypted"),
	}).Error)
	assertCount(0)
	var count int64
	require.NoError(t, db.Model(&models.BitbucketForgeInstallation{}).Count(&count).Error)
	require.Equal(t, int64(5), count, "monitoring must preserve installation records")
}
