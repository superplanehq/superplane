package models_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestSaveBitbucketForgeDeliveryKeepsTheLaterToken(t *testing.T) {
	support.Setup(t)
	db := database.Conn()
	installationID := uuid.NewString()
	firstDelivery := time.Now().UTC().Truncate(time.Microsecond)
	secondDelivery := firstDelivery.Add(time.Minute)
	laterExpiry := firstDelivery.Add(4 * time.Hour)
	nearerExpiry := firstDelivery.Add(time.Hour)

	_, err := models.SaveBitbucketForgeDelivery(db, models.BitbucketForgeDelivery{
		InstallationID: installationID,
		WorkspaceUUID:  "workspace-1",
		SystemToken:    []byte("later-token"),
		TokenExpiresAt: laterExpiry,
		DeliveredAt:    firstDelivery,
	})
	require.NoError(t, err)

	saved, err := models.SaveBitbucketForgeDelivery(db, models.BitbucketForgeDelivery{
		InstallationID: installationID,
		SystemToken:    []byte("nearer-token"),
		TokenExpiresAt: nearerExpiry,
		DeliveredAt:    secondDelivery,
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("later-token"), saved.SystemToken)
	assert.WithinDuration(t, laterExpiry, saved.TokenExpiresAt, time.Second)
	assert.WithinDuration(t, secondDelivery, saved.LastDeliveryAt, time.Second)

	reloaded, err := models.FindBitbucketForgeInstallation(db, installationID)
	require.NoError(t, err)
	assert.Equal(t, []byte("later-token"), reloaded.SystemToken)
	assert.WithinDuration(t, secondDelivery, reloaded.LastDeliveryAt, time.Second)
}

func TestSaveBitbucketForgeDeliveryHandlesConcurrentFirstDeliveries(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	installationID := uuid.NewString()
	now := time.Now().UTC().Truncate(time.Microsecond)
	const deliveries = 16
	start := make(chan struct{})
	results := make(chan error, deliveries)
	for index := range deliveries {
		go func() {
			<-start
			_, err := models.SaveBitbucketForgeDelivery(db, models.BitbucketForgeDelivery{
				InstallationID: installationID,
				SystemToken:    []byte(fmt.Sprintf("token-%d", index)),
				TokenExpiresAt: now.Add(time.Duration(index+1) * time.Hour),
				DeliveredAt:    now,
			})
			results <- err
		}()
	}
	close(start)
	for range deliveries {
		assert.NoError(t, <-results)
	}
	saved, err := models.FindBitbucketForgeInstallation(db, installationID)
	require.NoError(t, err)
	assert.Equal(t, []byte("token-15"), saved.SystemToken)
	assert.WithinDuration(t, now.Add(deliveries*time.Hour), saved.TokenExpiresAt, time.Microsecond)
}

func TestSaveBitbucketForgeDeliveryDoesNotRestoreAnUninstalledToken(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	installationID := uuid.NewString()
	now := time.Now().UTC().Truncate(time.Microsecond)
	_, err := models.SaveBitbucketForgeDelivery(db, models.BitbucketForgeDelivery{
		InstallationID: installationID,
		Uninstall:      true,
		DeliveredAt:    now,
	})
	require.NoError(t, err)
	_, err = models.SaveBitbucketForgeDelivery(db, models.BitbucketForgeDelivery{
		InstallationID: installationID,
		SystemToken:    []byte("delayed-token"),
		TokenExpiresAt: now.Add(time.Hour),
		DeliveredAt:    now.Add(time.Second),
	})
	require.NoError(t, err)
	saved, err := models.FindBitbucketForgeInstallation(db, installationID)
	require.NoError(t, err)
	assert.NotNil(t, saved.UninstalledAt)
	assert.Empty(t, saved.SystemToken)
}
