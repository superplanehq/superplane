package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__SoftDeleteWebhookIfUnreferenced(t *testing.T) {
	t.Run("soft-deletes when no active nodes remain", func(t *testing.T) {
		r := support.Setup(t)
		webhookID := createWebhook(t)
		canvas, node := createCanvasNodeWithWebhook(t, r, webhookID)

		require.NoError(t, database.Conn().Delete(&node).Error)
		require.NoError(t, models.SoftDeleteWebhookIfUnreferenced(database.Conn(), webhookID))

		_, err := models.FindWebhook(webhookID)
		require.Error(t, err)

		var stored models.Webhook
		require.NoError(t, database.Conn().Unscoped().First(&stored, webhookID).Error)
		assert.True(t, stored.DeletedAt.Valid)
		assert.Equal(t, canvas.ID, node.WorkflowID)
	})

	t.Run("keeps the webhook while an active node still references it", func(t *testing.T) {
		r := support.Setup(t)
		webhookID := createWebhook(t)
		_, _ = createCanvasNodeWithWebhook(t, r, webhookID)

		require.NoError(t, models.SoftDeleteWebhookIfUnreferenced(database.Conn(), webhookID))

		_, err := models.FindWebhook(webhookID)
		require.NoError(t, err)
	})

	t.Run("is a no-op when the webhook is already deleted", func(t *testing.T) {
		r := support.Setup(t)
		_ = r
		webhookID := createWebhook(t)
		require.NoError(t, database.Conn().Delete(&models.Webhook{ID: webhookID}).Error)

		require.NoError(t, models.SoftDeleteWebhookIfUnreferenced(database.Conn(), webhookID))
	})

	t.Run("ignores nodes on a soft-deleted canvas", func(t *testing.T) {
		r := support.Setup(t)
		webhookID := createWebhook(t)
		canvas, _ := createCanvasNodeWithWebhook(t, r, webhookID)
		require.NoError(t, canvas.SoftDelete())

		require.NoError(t, models.SoftDeleteWebhookIfUnreferenced(database.Conn(), webhookID))

		var stored models.Webhook
		require.NoError(t, database.Conn().Unscoped().First(&stored, webhookID).Error)
		assert.True(t, stored.DeletedAt.Valid)
	})
}

func Test__WebhookFail__SetsErrorWhenMetadataIsEmpty(t *testing.T) {
	support.Setup(t)

	webhookID := createWebhook(t)
	webhook, err := models.FindWebhook(webhookID)
	require.NoError(t, err)

	require.NoError(t, webhook.Fail(database.Conn(), "boom"))

	assert.Equal(t, map[string]any{"setupError": "boom"}, webhookMetadata(t, webhookID))
}

func Test__WebhookFail__PreservesExistingMetadata(t *testing.T) {
	support.Setup(t)

	webhookID := createWebhook(t)
	setWebhookMetadata(t, webhookID, map[string]any{"webhookId": "wh-123"})
	webhook, err := models.FindWebhook(webhookID)
	require.NoError(t, err)

	require.NoError(t, webhook.Fail(database.Conn(), "boom"))

	metadata := webhookMetadata(t, webhookID)
	assert.Equal(t, "wh-123", metadata["webhookId"])
	assert.Equal(t, "boom", metadata["setupError"])
}

func Test__WebhookFail__ReplacesPreviousSetupError(t *testing.T) {
	support.Setup(t)

	webhookID := createWebhook(t)
	setWebhookMetadata(t, webhookID, map[string]any{"webhookId": "wh-123", "setupError": "old"})
	webhook, err := models.FindWebhook(webhookID)
	require.NoError(t, err)

	require.NoError(t, webhook.Fail(database.Conn(), "new"))

	metadata := webhookMetadata(t, webhookID)
	assert.Equal(t, "wh-123", metadata["webhookId"])
	assert.Equal(t, "new", metadata["setupError"])
}

func Test__WebhookFail__EmptyReasonLeavesMetadataUnchanged(t *testing.T) {
	support.Setup(t)

	webhookID := createWebhook(t)
	setWebhookMetadata(t, webhookID, map[string]any{"webhookId": "wh-123"})
	webhook, err := models.FindWebhook(webhookID)
	require.NoError(t, err)

	require.NoError(t, webhook.Fail(database.Conn(), ""))

	assert.Equal(t, map[string]any{"webhookId": "wh-123"}, webhookMetadata(t, webhookID))
}

func setWebhookMetadata(t *testing.T, webhookID uuid.UUID, metadata map[string]any) {
	t.Helper()
	require.NoError(t, database.Conn().Model(&models.Webhook{ID: webhookID}).
		Update("metadata", datatypes.NewJSONType[any](metadata)).Error)
}

func webhookMetadata(t *testing.T, webhookID uuid.UUID) map[string]any {
	t.Helper()

	reloaded, err := models.FindWebhook(webhookID)
	require.NoError(t, err)
	assert.Equal(t, models.WebhookStateFailed, reloaded.State)
	metadata, ok := reloaded.Metadata.Data().(map[string]any)
	require.True(t, ok)
	return metadata
}

func createWebhook(t *testing.T) uuid.UUID {
	t.Helper()
	webhookID := uuid.New()
	require.NoError(t, database.Conn().Create(&models.Webhook{
		ID:     webhookID,
		State:  models.WebhookStatePending,
		Secret: []byte("secret"),
	}).Error)
	return webhookID
}

func createCanvasNodeWithWebhook(t *testing.T, r *support.ResourceRegistry, webhookID uuid.UUID) (*models.Canvas, models.CanvasNode) {
	t.Helper()
	canvas, nodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)
	require.Len(t, nodes, 1)
	require.NoError(t, database.Conn().Model(&nodes[0]).Update("webhook_id", webhookID).Error)
	nodes[0].WebhookID = &webhookID
	return canvas, nodes[0]
}
