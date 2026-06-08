package workers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
)

func Test__WebhookBackfillJob_CreatesBindingForScopeKeyedHandler(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.WebhookHandlers["dummy"] = newScopedHandler(testScopeKey, impl.DummyWebhookHandlerOptions{})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "test-integration", nil)
	require.NoError(t, err)

	webhook := wMakeWebhook(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeLegacy)
	canvas := wMakeCanvas(t, r.Organization.ID)
	wMakeNode(t, canvas.ID, "node-1", &webhook.ID, &integration.ID)

	job := NewWebhookBackfillJob(r.Registry)
	require.NoError(t, job.Run())

	var bindings []models.WebhookSubscriptionBinding
	require.NoError(t, database.Conn().
		Where("workflow_id = ? AND node_id = ? AND active = true", canvas.ID, "node-1").
		Find(&bindings).Error)
	require.Len(t, bindings, 1)
	assert.Equal(t, testScopeKey, bindings[0].ScopeKey)
	assert.Equal(t, &webhook.ID, bindings[0].WebhookID)
}

func Test__WebhookBackfillJob_SkipsNodeWithExistingBinding(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.WebhookHandlers["dummy"] = newScopedHandler(testScopeKey, impl.DummyWebhookHandlerOptions{})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "test-integration", nil)
	require.NoError(t, err)

	webhook := wMakeWebhook(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeLegacy)
	canvas := wMakeCanvas(t, r.Organization.ID)
	wMakeNode(t, canvas.ID, "node-1", &webhook.ID, &integration.ID)

	// Pre-existing binding for this node.
	wMakeBinding(t, r.Organization.ID, integration.ID, canvas.ID, "node-1", testScopeKey, &webhook.ID)

	job := NewWebhookBackfillJob(r.Registry)
	require.NoError(t, job.Run())

	// Still exactly one binding — not duplicated.
	var count int64
	require.NoError(t, database.Conn().Model(&models.WebhookSubscriptionBinding{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func Test__WebhookBackfillJob_SkipsHandlerWithoutScopeKeyer(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	// Plain DummyWebhookHandler does NOT implement ScopeKeyer.
	r.Registry.WebhookHandlers["dummy"] = impl.NewDummyWebhookHandler(impl.DummyWebhookHandlerOptions{})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "test-integration", nil)
	require.NoError(t, err)

	webhook := wMakeWebhook(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeLegacy)
	canvas := wMakeCanvas(t, r.Organization.ID)
	wMakeNode(t, canvas.ID, "node-1", &webhook.ID, &integration.ID)

	job := NewWebhookBackfillJob(r.Registry)
	require.NoError(t, job.Run())

	// No binding should have been created.
	var count int64
	require.NoError(t, database.Conn().Model(&models.WebhookSubscriptionBinding{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}
