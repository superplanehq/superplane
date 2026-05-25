package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func TestFindWebhookByScope(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org := newOrg(t)
	integration := newIntegration(t, org.ID)
	webhook := newWebhookWithScope(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeOps, "repo:owner/repo")

	t.Run("returns the webhook when scope matches", func(t *testing.T) {
		found, err := models.FindWebhookByScope(database.Conn(), integration.ID, "repo:owner/repo")
		require.NoError(t, err)
		assert.Equal(t, webhook.ID, found.ID)
	})

	t.Run("returns ErrRecordNotFound for unknown scope", func(t *testing.T) {
		_, err := models.FindWebhookByScope(database.Conn(), integration.ID, "repo:owner/other")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func TestListOrphanedOpsWebhooks(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org := newOrg(t)
	integration := newIntegration(t, org.ID)
	canvas := newCanvas(t, org.ID)

	// Ops-mode webhook with no bindings → should be returned as orphaned.
	orphan := newWebhookWithScope(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeOps, "repo:owner/a")

	// Ops-mode webhook WITH an active binding → not orphaned.
	withBinding := newWebhookWithScope(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeOps, "repo:owner/b")
	newBinding(t, org.ID, integration.ID, canvas.ID, "node-1", "repo:owner/b", &withBinding.ID, true)

	// Ops-mode webhook already in deleting_pending → excluded.
	_ = newWebhookWithScope(t, &integration.ID, models.WebhookStateDeletingPending, models.WebhookProvisioningModeOps, "repo:owner/c")

	// Legacy-mode webhook with no bindings → excluded (not ops-mode).
	_ = newWebhook(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeLegacy)

	result, err := models.ListOrphanedOpsWebhooks()
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, orphan.ID, result[0].ID)
}

func TestListOpsWebhooksWithoutPendingOp(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org := newOrg(t)
	integration := newIntegration(t, org.ID)

	// Ops-mode pending webhook with no queued op → should be returned.
	missing := newWebhookWithScope(t, &integration.ID, models.WebhookStatePending, models.WebhookProvisioningModeOps, "repo:owner/a")

	// Ops-mode pending webhook WITH a queued op → not returned.
	withOp := newWebhookWithScope(t, &integration.ID, models.WebhookStatePending, models.WebhookProvisioningModeOps, "repo:owner/b")
	newWebhookOp(t, withOp.ID, models.WebhookOperationTypeCreate, models.WebhookOperationStateQueued, time.Now())

	// Ops-mode ready webhook → not returned (wrong state).
	_ = newWebhookWithScope(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeOps, "repo:owner/c")

	result, err := models.ListOpsWebhooksWithoutPendingOp()
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, missing.ID, result[0].ID)
}

func TestListPendingWebhooks_ExcludesOpsMode(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org := newOrg(t)
	integration := newIntegration(t, org.ID)

	// Legacy pending → should be returned.
	legacyPending := newWebhook(t, &integration.ID, models.WebhookStatePending, models.WebhookProvisioningModeLegacy)

	// Ops pending → must NOT be returned (coexistence guard).
	_ = newWebhookWithScope(t, &integration.ID, models.WebhookStatePending, models.WebhookProvisioningModeOps, "repo:owner/b")

	// Legacy ready → not pending, so not returned.
	_ = newWebhook(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeLegacy)

	result, err := models.ListPendingWebhooks()
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, legacyPending.ID, result[0].ID)
}
