package models_test

import (
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
