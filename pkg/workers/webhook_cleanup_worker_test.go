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
