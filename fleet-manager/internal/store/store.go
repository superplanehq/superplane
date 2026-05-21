package store

import (
	"context"
	"time"

	"github.com/superplane/runner/shared/models"
)

// Store persists tasks and supports claiming with leases.
type Store interface {
	CreateTask(ctx context.Context, t *models.Task) error
	// GetTask returns a task by id, or nil if not found.
	GetTask(ctx context.Context, id string) (*models.Task, error)
	// ClaimTask assigns the next queued task to runnerID, or returns nil if none.
	ClaimTask(ctx context.Context, runnerID string, lease time.Duration) (*models.Task, error)
	// RequestCancelTask requests stop: queued tasks become canceled immediately; claimed tasks set cancel_requested.
	RequestCancelTask(ctx context.Context, id string) (*models.Task, CancelOutcome, error)
	// CompleteTask records terminal state; runnerID must match the claim. When canceled is true, status is always canceled.
	CompleteTask(ctx context.Context, id, runnerID string, exitCode int, resultJSON, errMsg string, canceled bool) (*models.Task, error)
	// ReapExpiredLeases returns expired claimed rows to queued, or to canceled when cancel_requested; canceled lists tasks that need webhooks.
	ReapExpiredLeases(ctx context.Context) (requeued int64, canceled []*models.Task, err error)
}
