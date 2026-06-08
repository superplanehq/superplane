package workers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__WebhookDedupeJob_NoOpWhenNoDuplicates(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "test-integration", nil)
	require.NoError(t, err)

	wMakeWebhookWithScope(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeOps, "repo:owner/a")
	wMakeWebhookWithScope(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeOps, "repo:owner/b")

	job := NewWebhookDedupeJob()
	require.NoError(t, job.Run())
}

func Test__WebhookDedupeJob_RebindsNodesToCanonical(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "test-integration", nil)
	require.NoError(t, err)

	canonical := wMakeWebhookWithScope(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeOps, "repo:owner/canon")
	duplicate := wMakeWebhookWithScope(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeOps, "repo:owner/dup")

	canvas := wMakeCanvas(t, r.Organization.ID)
	wMakeNode(t, canvas.ID, "node-1", &duplicate.ID, &integration.ID)
	wMakeBinding(t, r.Organization.ID, integration.ID, canvas.ID, "node-1", "repo:owner/dup", &duplicate.ID)

	job := NewWebhookDedupeJob()
	require.NoError(t, job.rebindAndMark(database.Conn(), duplicate.ID, canonical.ID))

	// Node must point to canonical.
	var node models.CanvasNode
	require.NoError(t, database.Conn().Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").First(&node).Error)
	assert.Equal(t, canonical.ID, *node.WebhookID)

	// Binding must point to canonical.
	var binding models.WebhookSubscriptionBinding
	require.NoError(t, database.Conn().
		Where("workflow_id = ? AND node_id = ? AND active = true", canvas.ID, "node-1").
		First(&binding).Error)
	assert.Equal(t, canonical.ID, *binding.WebhookID)

	// Duplicate must now be ops-mode so the reconciler orphan path deletes it.
	var dup models.Webhook
	require.NoError(t, database.Conn().First(&dup, duplicate.ID).Error)
	assert.Equal(t, models.WebhookProvisioningModeOps, dup.ProvisioningMode)
}

func Test__WebhookDedupeJob_PickCanonical_PrefersReady(t *testing.T) {
	job := NewWebhookDedupeJob()

	oldPending := models.Webhook{ID: uuid.New(), State: models.WebhookStatePending}
	oldReady := models.Webhook{ID: uuid.New(), State: models.WebhookStateReady}
	newReady := models.Webhook{ID: uuid.New(), State: models.WebhookStateReady}

	// Slice is ordered oldest-first (as the DB query returns it).
	webhooks := []models.Webhook{oldPending, oldReady, newReady}
	canonical := job.pickCanonical(webhooks)

	// Must prefer the oldest 'ready' entry.
	assert.Equal(t, oldReady.ID, canonical.ID)
}

func Test__WebhookDedupeJob_PickCanonical_FallsBackToOldest(t *testing.T) {
	job := NewWebhookDedupeJob()

	oldest := models.Webhook{ID: uuid.New(), State: models.WebhookStatePending}
	newer := models.Webhook{ID: uuid.New(), State: models.WebhookStateFailed}

	webhooks := []models.Webhook{oldest, newer}
	canonical := job.pickCanonical(webhooks)

	assert.Equal(t, oldest.ID, canonical.ID)
}
