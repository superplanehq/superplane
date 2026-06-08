package workers

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
)

func Test__WebhookOpsProvisioner_SuccessfulCreate(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.WebhookHandlers["dummy"] = impl.NewDummyWebhookHandler(impl.DummyWebhookHandlerOptions{
		SetupFunc: func(ctx core.WebhookHandlerContext) (any, error) {
			return map[string]any{"provider_id": "gh-123"}, nil
		},
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "test-integration", nil)
	require.NoError(t, err)

	webhook := wMakeWebhookWithScope(t, &integration.ID, models.WebhookStatePending, models.WebhookProvisioningModeOps, testScopeKey)
	op := wMakeOp(t, webhook.ID, models.WebhookOperationTypeCreate, models.WebhookOperationStateQueued, time.Now())

	provisioner := NewWebhookOpsProvisioner("https://example.com", r.Encryptor, r.Registry)
	require.NoError(t, provisioner.lockAndProcess(*op))

	var updatedOp models.WebhookOperation
	require.NoError(t, database.Conn().First(&updatedOp, op.ID).Error)
	assert.Equal(t, models.WebhookOperationStateSucceeded, updatedOp.State)

	updatedWebhook, err := models.FindWebhook(webhook.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WebhookStateReady, updatedWebhook.State)
}

func Test__WebhookOpsProvisioner_SuccessfulDelete(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.WebhookHandlers["dummy"] = impl.NewDummyWebhookHandler(impl.DummyWebhookHandlerOptions{
		CleanupFunc: func(ctx core.WebhookHandlerContext) error { return nil },
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "test-integration", nil)
	require.NoError(t, err)

	webhook := wMakeWebhookWithScope(t, &integration.ID, models.WebhookStateDeletingPending, models.WebhookProvisioningModeOps, testScopeKey)
	op := wMakeOp(t, webhook.ID, models.WebhookOperationTypeDelete, models.WebhookOperationStateQueued, time.Now())

	provisioner := NewWebhookOpsProvisioner("https://example.com", r.Encryptor, r.Registry)
	require.NoError(t, provisioner.lockAndProcess(*op))

	var updatedOp models.WebhookOperation
	require.NoError(t, database.Conn().First(&updatedOp, op.ID).Error)
	assert.Equal(t, models.WebhookOperationStateSucceeded, updatedOp.State)

	// Webhook must be soft-deleted after a successful delete op.
	var count int64
	require.NoError(t, database.Conn().Model(&models.Webhook{}).Where("id = ?", webhook.ID).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func Test__WebhookOpsProvisioner_FailureWithRetry(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.WebhookHandlers["dummy"] = impl.NewDummyWebhookHandler(impl.DummyWebhookHandlerOptions{
		SetupFunc: func(ctx core.WebhookHandlerContext) (any, error) {
			return nil, errors.New("provider unavailable")
		},
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "test-integration", nil)
	require.NoError(t, err)

	webhook := wMakeWebhookWithScope(t, &integration.ID, models.WebhookStatePending, models.WebhookProvisioningModeOps, testScopeKey)
	op := wMakeOp(t, webhook.ID, models.WebhookOperationTypeCreate, models.WebhookOperationStateQueued, time.Now())

	provisioner := NewWebhookOpsProvisioner("https://example.com", r.Encryptor, r.Registry)
	require.NoError(t, provisioner.lockAndProcess(*op))

	var updatedOp models.WebhookOperation
	require.NoError(t, database.Conn().First(&updatedOp, op.ID).Error)
	assert.Equal(t, models.WebhookOperationStateFailedRetryable, updatedOp.State)
	assert.Equal(t, 1, updatedOp.AttemptCount)
	assert.NotNil(t, updatedOp.LastErrorMessage)
	// next_attempt_at must be in the future (backoff applied).
	assert.True(t, updatedOp.NextAttemptAt.After(time.Now()))
}

func Test__WebhookOpsProvisioner_TerminalFailure(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.WebhookHandlers["dummy"] = impl.NewDummyWebhookHandler(impl.DummyWebhookHandlerOptions{
		SetupFunc: func(ctx core.WebhookHandlerContext) (any, error) {
			return nil, errors.New("unrecoverable error")
		},
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", "test-integration", nil)
	require.NoError(t, err)

	webhook := wMakeWebhookWithScope(t, &integration.ID, models.WebhookStatePending, models.WebhookProvisioningModeOps, testScopeKey)

	// Start with AttemptCount already at MaxAttempts - 1 so the next failure is terminal.
	op := wMakeOp(t, webhook.ID, models.WebhookOperationTypeCreate, models.WebhookOperationStateQueued, time.Now())
	require.NoError(t, database.Conn().Model(op).UpdateColumn("attempt_count", op.MaxAttempts-1).Error)
	op.AttemptCount = op.MaxAttempts - 1

	provisioner := NewWebhookOpsProvisioner("https://example.com", r.Encryptor, r.Registry)
	require.NoError(t, provisioner.lockAndProcess(*op))

	var updatedOp models.WebhookOperation
	require.NoError(t, database.Conn().First(&updatedOp, op.ID).Error)
	assert.Equal(t, models.WebhookOperationStateFailedTerminal, updatedOp.State)
	assert.Equal(t, op.MaxAttempts, updatedOp.AttemptCount)
}
