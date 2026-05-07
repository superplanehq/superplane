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
	// CompleteTask records terminal state; runnerID must match the claim.
	CompleteTask(ctx context.Context, id, runnerID string, exitCode int, output, errMsg string) (*models.Task, error)
	ReapExpiredLeases(ctx context.Context) (int64, error)
}
