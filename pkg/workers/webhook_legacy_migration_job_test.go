package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
)

func Test__WebhookLegacyMigrationJob_MigratesReadyWebhookToOpsMode(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.WebhookHandlers["dummy"] = newScopedHandler(testScopeKey, impl.DummyWebhookHandlerOptions{})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "test-integration", nil)
	require.NoError(t, err)

	webhook := wMakeWebhook(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeLegacy)

	job := NewWebhookLegacyMigrationJob(r.Registry)
	require.NoError(t, job.Run())

	updated, err := models.FindWebhook(webhook.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WebhookProvisioningModeOps, updated.ProvisioningMode)
	assert.Equal(t, models.WebhookStateReady, updated.State) // state preserved
	require.NotNil(t, updated.ScopeKey)
	assert.Equal(t, testScopeKey, *updated.ScopeKey)

	// Ready webhooks do not need re-provisioning; no op should be created.
	ops, err := models.ListQueuedOperations()
	require.NoError(t, err)
	assert.Empty(t, ops)
}

func Test__WebhookLegacyMigrationJob_MigratesPendingWebhookAndEnqueuesOp(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.WebhookHandlers["dummy"] = newScopedHandler(testScopeKey, impl.DummyWebhookHandlerOptions{})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "test-integration", nil)
	require.NoError(t, err)

	webhook := wMakeWebhook(t, &integration.ID, models.WebhookStatePending, models.WebhookProvisioningModeLegacy)

	job := NewWebhookLegacyMigrationJob(r.Registry)
	require.NoError(t, job.Run())

	updated, err := models.FindWebhook(webhook.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WebhookProvisioningModeOps, updated.ProvisioningMode)

	// A 'create' op must be queued so the ops provisioner picks it up.
	ops, err := models.ListQueuedOperations()
	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, models.WebhookOperationTypeCreate, ops[0].OperationType)
	assert.Equal(t, webhook.ID, ops[0].WebhookID)
	// next_attempt_at must be in the past or now so it is eligible immediately.
	assert.False(t, ops[0].NextAttemptAt.After(time.Now().Add(time.Second)))
}

func Test__WebhookLegacyMigrationJob_SkipsHandlerWithoutScopeKeyer(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	// Plain DummyWebhookHandler does NOT implement ScopeKeyer.
	r.Registry.WebhookHandlers["dummy"] = impl.NewDummyWebhookHandler(impl.DummyWebhookHandlerOptions{})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "test-integration", nil)
	require.NoError(t, err)

	webhook := wMakeWebhook(t, &integration.ID, models.WebhookStatePending, models.WebhookProvisioningModeLegacy)

	job := NewWebhookLegacyMigrationJob(r.Registry)
	require.NoError(t, job.Run())

	// Webhook must remain in legacy mode — handler has not adopted ScopeKeyer.
	unchanged, err := models.FindWebhook(webhook.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WebhookProvisioningModeLegacy, unchanged.ProvisioningMode)
	assert.Nil(t, unchanged.ScopeKey)
}
