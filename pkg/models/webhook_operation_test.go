package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestEnqueueWebhookOperation(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	webhook := newWebhook(t, nil, models.WebhookStatePending, models.WebhookProvisioningModeLegacy)

	t.Run("creates operation on first call", func(t *testing.T) {
		op := &models.WebhookOperation{
			WebhookID:      webhook.ID,
			OperationType:  models.WebhookOperationTypeCreate,
			IdempotencyKey: "idem-create-1",
			State:          models.WebhookOperationStateQueued,
			MaxAttempts:    5,
			NextAttemptAt:  time.Now(),
		}
		require.NoError(t, models.EnqueueWebhookOperation(database.Conn(), op))
		assert.NotEqual(t, uuid.Nil, op.ID)
	})

	t.Run("is idempotent on duplicate idempotency key", func(t *testing.T) {
		op1 := &models.WebhookOperation{
			WebhookID:      webhook.ID,
			OperationType:  models.WebhookOperationTypeCreate,
			IdempotencyKey: "idem-dup",
			State:          models.WebhookOperationStateQueued,
			MaxAttempts:    5,
			NextAttemptAt:  time.Now(),
		}
		require.NoError(t, models.EnqueueWebhookOperation(database.Conn(), op1))

		// Same idempotency key with different type — should be a no-op, returning op1's row.
		op2 := &models.WebhookOperation{
			WebhookID:      webhook.ID,
			OperationType:  models.WebhookOperationTypeUpdate,
			IdempotencyKey: "idem-dup",
			State:          models.WebhookOperationStateQueued,
			MaxAttempts:    5,
			NextAttemptAt:  time.Now(),
		}
		require.NoError(t, models.EnqueueWebhookOperation(database.Conn(), op2))

		var count int64
		require.NoError(t, database.Conn().Model(&models.WebhookOperation{}).
			Where("idempotency_key = ?", "idem-dup").Count(&count).Error)
		assert.Equal(t, int64(1), count)

		// FirstOrCreate loads the existing row into op2.
		assert.Equal(t, models.WebhookOperationTypeCreate, op2.OperationType)
	})
}

func TestListQueuedOperations(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	webhook := newWebhook(t, nil, models.WebhookStatePending, models.WebhookProvisioningModeLegacy)
	past := time.Now().Add(-1 * time.Minute)
	future := time.Now().Add(10 * time.Minute)

	queued := newWebhookOp(t, webhook.ID, models.WebhookOperationTypeCreate, models.WebhookOperationStateQueued, past)
	retryable := newWebhookOp(t, webhook.ID, models.WebhookOperationTypeUpdate, models.WebhookOperationStateFailedRetryable, past)
	// In backoff — should NOT be returned.
	_ = newWebhookOp(t, webhook.ID, models.WebhookOperationTypeUpdate, models.WebhookOperationStateFailedRetryable, future)
	// Already succeeded — should NOT be returned.
	_ = newWebhookOp(t, webhook.ID, models.WebhookOperationTypeDelete, models.WebhookOperationStateSucceeded, past)

	ops, err := models.ListQueuedOperations()
	require.NoError(t, err)

	ids := webhookOpIDs(ops)
	assert.Contains(t, ids, queued.ID)
	assert.Contains(t, ids, retryable.ID)
	assert.Len(t, ids, 2)
}

func TestResetStuckRunningOperations(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	webhook := newWebhook(t, nil, models.WebhookStatePending, models.WebhookProvisioningModeLegacy)

	// Create an op in 'running' state and backdate updated_at past the lease timeout.
	stuckOp := newWebhookOp(t, webhook.ID, models.WebhookOperationTypeCreate, models.WebhookOperationStateRunning, time.Now())
	stuckTime := time.Now().Add(-11 * time.Minute)
	require.NoError(t, database.Conn().Model(stuckOp).UpdateColumn("updated_at", stuckTime).Error)

	// Create a recently-started 'running' op — must not be touched.
	recentOp := newWebhookOp(t, webhook.ID, models.WebhookOperationTypeUpdate, models.WebhookOperationStateRunning, time.Now())

	count, err := models.ResetStuckRunningOperations()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	var stuck models.WebhookOperation
	require.NoError(t, database.Conn().First(&stuck, stuckOp.ID).Error)
	assert.Equal(t, models.WebhookOperationStateQueued, stuck.State)

	var recent models.WebhookOperation
	require.NoError(t, database.Conn().First(&recent, recentOp.ID).Error)
	assert.Equal(t, models.WebhookOperationStateRunning, recent.State)
}

func webhookOpIDs(ops []models.WebhookOperation) []uuid.UUID {
	ids := make([]uuid.UUID, len(ops))
	for i, op := range ops {
		ids[i] = op.ID
	}
	return ids
}
