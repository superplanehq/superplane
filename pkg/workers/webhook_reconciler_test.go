package workers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
)

const testScopeKey = "repo:owner/repo"

func Test__WebhookReconciler_CreatesRegistrationAndOp(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.WebhookHandlers["dummy"] = newScopedHandler(testScopeKey, impl.DummyWebhookHandlerOptions{
		CompareConfigFunc: func(a, b any) (bool, error) { return false, nil },
		MergeFunc:         func(current, requested any) (any, bool, error) { return current, false, nil },
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "test-integration", nil)
	require.NoError(t, err)

	canvas := wMakeCanvas(t, r.Organization.ID)
	wMakeBinding(t, r.Organization.ID, integration.ID, canvas.ID, "node-1", testScopeKey, nil)

	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureWebhookReconciler))

	reconciler := NewWebhookReconciler(r.Registry, r.Encryptor, "https://example.com")
	group := models.BindingGroup{AppInstallationID: integration.ID, ScopeKey: testScopeKey}
	require.NoError(t, reconciler.reconcileGroup(group))

	// A new ops-mode webhook registration should have been created.
	webhook, err := models.FindWebhookByScope(database.Conn(), integration.ID, testScopeKey)
	require.NoError(t, err)
	assert.Equal(t, models.WebhookProvisioningModeOps, webhook.ProvisioningMode)
	assert.Equal(t, models.WebhookStatePending, webhook.State)

	// A 'create' operation should be queued.
	ops, err := models.ListQueuedOperations()
	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, models.WebhookOperationTypeCreate, ops[0].OperationType)
	assert.Equal(t, webhook.ID, ops[0].WebhookID)
}

func Test__WebhookReconciler_EnqueuesUpdateOpOnDrift(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	// CompareConfig returns false → configs differ → drift detected.
	r.Registry.WebhookHandlers["dummy"] = newScopedHandler(testScopeKey, impl.DummyWebhookHandlerOptions{
		CompareConfigFunc: func(a, b any) (bool, error) { return false, nil },
		MergeFunc:         func(current, requested any) (any, bool, error) { return requested, true, nil },
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "test-integration", nil)
	require.NoError(t, err)

	canvas := wMakeCanvas(t, r.Organization.ID)
	wMakeBinding(t, r.Organization.ID, integration.ID, canvas.ID, "node-1", testScopeKey, nil)

	existing := wMakeWebhookWithScope(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeOps, testScopeKey)

	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureWebhookReconciler))

	reconciler := NewWebhookReconciler(r.Registry, r.Encryptor, "https://example.com")
	group := models.BindingGroup{AppInstallationID: integration.ID, ScopeKey: testScopeKey}
	require.NoError(t, reconciler.reconcileGroup(group))

	// An 'update' operation should be queued for the existing registration.
	ops, err := models.ListQueuedOperations()
	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, models.WebhookOperationTypeUpdate, ops[0].OperationType)
	assert.Equal(t, existing.ID, ops[0].WebhookID)
}

func Test__WebhookReconciler_NoOpWhenConfigUnchanged(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	// CompareConfig returns true → configs match → no drift.
	r.Registry.WebhookHandlers["dummy"] = newScopedHandler(testScopeKey, impl.DummyWebhookHandlerOptions{
		CompareConfigFunc: func(a, b any) (bool, error) { return true, nil },
		MergeFunc:         func(current, requested any) (any, bool, error) { return current, false, nil },
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "test-integration", nil)
	require.NoError(t, err)

	canvas := wMakeCanvas(t, r.Organization.ID)
	wMakeBinding(t, r.Organization.ID, integration.ID, canvas.ID, "node-1", testScopeKey, nil)
	wMakeWebhookWithScope(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeOps, testScopeKey)

	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureWebhookReconciler))

	reconciler := NewWebhookReconciler(r.Registry, r.Encryptor, "https://example.com")
	group := models.BindingGroup{AppInstallationID: integration.ID, ScopeKey: testScopeKey}
	require.NoError(t, reconciler.reconcileGroup(group))

	// No operations should have been enqueued.
	ops, err := models.ListQueuedOperations()
	require.NoError(t, err)
	assert.Empty(t, ops)
}

func Test__WebhookReconciler_ShadowModeWhenFeatureDisabled(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.WebhookHandlers["dummy"] = newScopedHandler(testScopeKey, impl.DummyWebhookHandlerOptions{
		CompareConfigFunc: func(a, b any) (bool, error) { return false, nil },
		MergeFunc:         func(current, requested any) (any, bool, error) { return requested, true, nil },
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "test-integration", nil)
	require.NoError(t, err)

	canvas := wMakeCanvas(t, r.Organization.ID)
	wMakeBinding(t, r.Organization.ID, integration.ID, canvas.ID, "node-1", testScopeKey, nil)

	// Feature NOT enabled — reconciler must log only and make no DB writes.

	reconciler := NewWebhookReconciler(r.Registry, r.Encryptor, "https://example.com")
	group := models.BindingGroup{AppInstallationID: integration.ID, ScopeKey: testScopeKey}
	require.NoError(t, reconciler.reconcileGroup(group))

	// No registration should have been created.
	var count int64
	require.NoError(t, database.Conn().Model(&models.Webhook{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)

	// No operations should have been enqueued.
	ops, err := models.ListQueuedOperations()
	require.NoError(t, err)
	assert.Empty(t, ops)
}
