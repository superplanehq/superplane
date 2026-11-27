package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/gorm"
)

func Test__IntegrationCleanupWorker_DrainsUnregisteredIntegration(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	cleanupCalls := 0
	webhookWorker, webhookID := setupWebhookCleanupWorker(t, r, func(ctx core.WebhookHandlerContext) error {
		cleanupCalls++
		return nil
	})
	delete(r.Registry.Integrations, "dummy")
	integrationID := integrationIDForWebhook(t, webhookID)
	integration, err := models.FindIntegration(r.Organization.ID, integrationID)
	require.NoError(t, err)
	require.NoError(t, integration.SoftDelete())
	worker := NewIntegrationCleanupWorker(r.Registry, r.Encryptor, "https://example.com")

	require.NoError(t, worker.LockAndProcessIntegration(*integration))
	var stored models.Integration
	require.NoError(t, database.Conn().Unscoped().First(&stored, integrationID).Error)
	assert.True(t, stored.DeletedAt.Valid)

	logger := logrus.NewEntry(logrus.New())
	require.NoError(t, webhookWorker.LockAndProcessWebhook(logger, models.Webhook{ID: webhookID}))
	assertWebhookHardDeleted(t, webhookID)
	require.NoError(t, worker.LockAndProcessIntegration(*integration))
	require.ErrorIs(t, database.Conn().Unscoped().First(&models.Integration{}, integrationID).Error, gorm.ErrRecordNotFound)

	webhooks, err := models.ListDeletedWebhooks()
	require.NoError(t, err)
	assert.Empty(t, webhooks)
	integrations, err := models.ListDeletedIntegrations()
	require.NoError(t, err)
	assert.Empty(t, integrations)
	assert.Zero(t, cleanupCalls)
}

func Test__IntegrationCleanupWorker_PreservesActiveIntegrations(t *testing.T) {
	for _, registered := range []bool{true, false} {
		name := "unregistered"
		if registered {
			name = "registered"
		}
		t.Run(name, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()

			cleanupCalls := 0
			if registered {
				r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
					OnCleanup: func(ctx core.IntegrationCleanupContext) error {
						cleanupCalls++
						return nil
					},
				})
			}
			integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "active", nil)
			require.NoError(t, err)
			var before models.Integration
			require.NoError(t, database.Conn().First(&before, integration.ID).Error)
			stale := *integration
			stale.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
			worker := NewIntegrationCleanupWorker(r.Registry, r.Encryptor, "https://example.com")

			require.NoError(t, worker.LockAndProcessIntegration(stale))

			var after models.Integration
			require.NoError(t, database.Conn().First(&after, integration.ID).Error)
			assert.Equal(t, before, after)
			assert.Zero(t, cleanupCalls)
		})
	}
}
