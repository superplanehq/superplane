package store

import (
	"context"
	"time"

	"github.com/superplane/runner/shared/models"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
)

// CancelOutcome is the result of RequestCancelTask.
type CancelOutcome string

const (
	CancelOutcomeNotFound        CancelOutcome = "not_found"
	CancelOutcomeAlreadyTerminal CancelOutcome = "already_terminal"
	CancelOutcomeCanceledQueued  CancelOutcome = "canceled"
	CancelOutcomeCancelRequested CancelOutcome = "cancel_requested"
)

// ReapedLease identifies a task whose expired lease was requeued. Only ID and
// FleetID are populated — not a full models.Task.
type ReapedLease struct {
	ID      string
	FleetID string
}

// Store persists fleets and the task queue.
type Store interface {
	CreateFleet(ctx context.Context, f *brokermodels.Fleet) error
	DeleteFleet(ctx context.Context, id string) error
	ListFleets(ctx context.Context) ([]brokermodels.Fleet, error)
	GetFleet(ctx context.Context, id string) (*brokermodels.Fleet, error)

	CreateTask(ctx context.Context, t *models.Task) error
	GetTask(ctx context.Context, id string) (*models.Task, error)
	ListActiveTasks(ctx context.Context) ([]*models.Task, error)
	CountTasksByFleet(ctx context.Context, fleetID string) (queued, claimed int, err error)
	ClaimTask(ctx context.Context, runnerID, fleetID string, lease time.Duration) (*models.Task, error)
	// UnclaimTask re-queues a claimed task so another runner can pick it up.
	// It is a no-op (returns nil) if the task is not in the claimed state or
	// is claimed by a different runner.
	UnclaimTask(ctx context.Context, taskID, runnerID string) error
	RequestCancelTask(ctx context.Context, id string) (*models.Task, CancelOutcome, error)
	CompleteTask(ctx context.Context, id, runnerID string, exitCode int, resultJSON, errMsg string, canceled bool) (*models.Task, error)
	ReapExpiredLeases(ctx context.Context) (requeued []ReapedLease, canceled []*models.Task, err error)
}
