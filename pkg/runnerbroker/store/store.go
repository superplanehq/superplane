package store

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/runnerbroker/models"
	brokermodels "github.com/superplanehq/superplane/pkg/runnerbroker/storemodels"
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

type CompleteTaskOutcome string

const (
	CompleteTaskOutcomeTerminal CompleteTaskOutcome = "terminal"
	CompleteTaskOutcomeRequeued CompleteTaskOutcome = "requeued"
)

type CompleteTaskRequest struct {
	ID           string
	RunnerID     string
	ExitCode     int
	ResultJSON   string
	ErrorMessage string
	Canceled     bool
	FailureKind  string
}

type CompleteTaskResult struct {
	Task    *models.Task
	Outcome CompleteTaskOutcome
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
	ClaimedRunnerIDsByFleet(ctx context.Context, fleetID string) ([]string, error)
	ClaimTask(ctx context.Context, runnerID, fleetID string, lease time.Duration) (*models.Task, error)
	// UnclaimTask re-queues a claimed task so another runner can pick it up.
	// Returns unclaimed=true when a row was updated; false when the task was not
	// claimed by runnerID (no-op).
	UnclaimTask(ctx context.Context, taskID, runnerID string) (unclaimed bool, err error)
	RequestCancelTask(ctx context.Context, id string) (*models.Task, CancelOutcome, error)
	CompleteTask(ctx context.Context, req CompleteTaskRequest) (*CompleteTaskResult, error)
	ReapExpiredLeases(ctx context.Context) (requeued []ReapedLease, canceled []*models.Task, err error)
}
