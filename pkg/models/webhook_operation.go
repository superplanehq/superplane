package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	WebhookOperationTypeCreate       = "create"
	WebhookOperationTypeUpdate       = "update"
	WebhookOperationTypeDelete       = "delete"
	WebhookOperationTypeRotateSecret = "rotate_secret"
	WebhookOperationTypeVerify       = "verify"

	WebhookOperationStateQueued          = "queued"
	WebhookOperationStateRunning         = "running"
	WebhookOperationStateSucceeded       = "succeeded"
	WebhookOperationStateFailedRetryable = "failed_retryable"
	WebhookOperationStateFailedTerminal  = "failed_terminal"

	// opLeaseTimeout is how long an operation may stay in 'running' before
	// it is assumed to belong to a dead worker and reset to 'queued'.
	opLeaseTimeout = 10 * time.Minute
)

type WebhookOperation struct {
	ID                uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	WebhookID         uuid.UUID
	OperationType     string
	DesiredConfig     datatypes.JSONType[any]
	DesiredConfigHash *string
	IdempotencyKey    string
	State             string
	AttemptCount      int `gorm:"default:0"`
	MaxAttempts       int `gorm:"default:5"`
	NextAttemptAt     time.Time
	LastErrorMessage  *string
	LastErrorAt       *time.Time
	CreatedAt         *time.Time
	UpdatedAt         *time.Time
}

// EnqueueWebhookOperation inserts op if no row with the same IdempotencyKey
// already exists. This makes reconciler enqueue calls safe to retry.
func EnqueueWebhookOperation(tx *gorm.DB, op *WebhookOperation) error {
	return tx.
		Where(WebhookOperation{IdempotencyKey: op.IdempotencyKey}).
		FirstOrCreate(op).
		Error
}

// ListQueuedOperations returns operations that are ready to be picked up by the ops
// provisioner: either freshly queued or in backoff (failed_retryable) with an elapsed
// next_attempt_at.
func ListQueuedOperations() ([]WebhookOperation, error) {
	var ops []WebhookOperation
	err := database.Conn().
		Where("state IN (?, ?) AND next_attempt_at <= NOW()",
			WebhookOperationStateQueued, WebhookOperationStateFailedRetryable).
		Find(&ops).
		Error
	return ops, err
}

// LockOperation acquires a FOR UPDATE SKIP LOCKED row lock on an operation that is
// ready to execute (queued or failed_retryable with elapsed backoff).
// Returns nil (no error) when another worker already claimed the row.
func LockOperation(tx *gorm.DB, id uuid.UUID) (*WebhookOperation, error) {
	var op WebhookOperation
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ? AND state IN (?, ?) AND next_attempt_at <= NOW()",
			id, WebhookOperationStateQueued, WebhookOperationStateFailedRetryable).
		First(&op).
		Error
	if err != nil {
		return nil, err
	}
	return &op, nil
}

// ResetStuckRunningOperations resets operations that have been in 'running'
// longer than opLeaseTimeout back to 'queued' so they are retried.
func ResetStuckRunningOperations() (int64, error) {
	cutoff := time.Now().Add(-opLeaseTimeout)
	result := database.Conn().
		Model(&WebhookOperation{}).
		Where("state = ? AND updated_at < ?", WebhookOperationStateRunning, cutoff).
		Updates(map[string]any{
			"state":      WebhookOperationStateQueued,
			"updated_at": time.Now(),
		})
	return result.RowsAffected, result.Error
}
