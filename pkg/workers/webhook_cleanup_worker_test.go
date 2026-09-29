package workers

import (
	"errors"
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
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__WebhookCleanupWorker_DeletesWebhookWhenProviderCleanupFails(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	logger := logrus.NewEntry(logrus.New())
	cleanupCalls := 0
	worker, webhookID := setupWebhookCleanupWorker(t, r, func(ctx core.WebhookHandlerContext) error {
		cleanupCalls++
		return errors.New("provider unavailable")
	})

	err := worker.LockAndProcessWebhook(logger, models.Webhook{ID: webhookID})
	require.NoError(t, err)

	assertWebhookHardDeleted(t, webhookID)
	assert.Equal(t, 1, cleanupCalls)
}

func Test__WebhookCleanupWorker_PersistsIntegrationMetadataClearedByCleanup(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	logger := logrus.NewEntry(logrus.New())
	worker, webhookID := setupWebhookCleanupWorker(t, r, func(ctx core.WebhookHandlerContext) error {
		ctx.Integration.SetMetadata(map[string]any{})
		return nil
	})

	integrationID := integrationIDForWebhook(t, webhookID)
	require.NoError(t, database.Conn().Model(&models.Integration{}).
		Where("id = ?", integrationID).
		Update("metadata", datatypes.NewJSONType(map[string]any{"webhookId": 34})).Error)

	require.NoError(t, worker.LockAndProcessWebhook(logger, models.Webhook{ID: webhookID}))

	var stored models.Integration
	require.NoError(t, database.Conn().Where("id = ?", integrationID).First(&stored).Error)
	assert.NotContains(t, stored.Metadata.Data(), "webhookId")
}

func integrationIDForWebhook(t *testing.T, webhookID uuid.UUID) uuid.UUID {
	t.Helper()

	var webhook models.Webhook
	require.NoError(t, database.Conn().Unscoped().Where("id = ?", webhookID).First(&webhook).Error)
	require.NotNil(t, webhook.AppInstallationID)
	return *webhook.AppInstallationID
}

func setupWebhookCleanupWorker(
	t *testing.T,
	r *support.ResourceRegistry,
	cleanupFunc func(ctx core.WebhookHandlerContext) error,
) (*WebhookCleanupWorker, uuid.UUID) {
	t.Helper()

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.WebhookHandlers["dummy"] = impl.NewDummyWebhookHandler(impl.DummyWebhookHandlerOptions{
		CleanupFunc: cleanupFunc,
	})

	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"dummy",
		support.RandomName("integration"),
		nil,
	)
	require.NoError(t, err)

	now := time.Now()
	webhook := models.Webhook{
		ID:                uuid.New(),
		State:             models.WebhookStateReady,
		Secret:            []byte("encrypted-secret"),
		AppInstallationID: &integration.ID,
		RetryCount:        3,
		MaxRetries:        3,
		CreatedAt:         &now,
	}
	require.NoError(t, database.Conn().Create(&webhook).Error)
	require.NoError(t, database.Conn().Delete(&webhook).Error)

	return NewWebhookCleanupWorker(r.Encryptor, r.Registry, "https://example.com"), webhook.ID
}

func assertWebhookHardDeleted(t *testing.T, webhookID uuid.UUID) {
	t.Helper()

	var webhook models.Webhook
	err := database.Conn().
		Unscoped().
		Where("id = ?", webhookID).
		First(&webhook).
		Error
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
