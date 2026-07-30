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

type LostRunnerTaskRecovery struct {
	ID       string
	FleetID  string
	RunnerID string
	Status   models.TaskStatus
}

type LostRunnerTaskTermination struct {
	ID       string
	FleetID  string
	RunnerID string
}

type CompleteTaskOutcome string

const (
	CompleteTaskOutcomeTerminal        CompleteTaskOutcome = "terminal"
	CompleteTaskOutcomeAlreadyTerminal CompleteTaskOutcome = "already_terminal"
	CompleteTaskOutcomeRequeued        CompleteTaskOutcome = "requeued"
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

// DispatchCandidate is a queued task whose fleet needs an active dispatch
// call (e.g. Lambda invoke), returned by ClaimDispatchCandidates along with
// the fleet fields needed to build a Dispatcher.
type DispatchCandidate struct {
	TaskID             string
	FleetID            string
	Provisioner        string
	LambdaFunctionName string
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
	OldestQueuedTaskCreatedAt(ctx context.Context, fleetID string) (*time.Time, error)
	ClaimedRunnerIDsByFleet(ctx context.Context, fleetID string) ([]string, error)
	ClaimedTaskIDsByRunners(ctx context.Context, fleetID string, runnerIDs []string) (map[string]string, error)
	ClaimTask(ctx context.Context, runnerID, fleetID string, lease time.Duration) (*models.Task, error)
	// UnclaimTask re-queues a claimed task so another runner can pick it up.
	// Returns unclaimed=true when a row was updated; false when the task was not
	// claimed by runnerID (no-op).
	UnclaimTask(ctx context.Context, taskID, runnerID string) (unclaimed bool, err error)
	MarkLostRunnerTasksTerminating(ctx context.Context, fleetID string, runnerIDs []string) ([]LostRunnerTaskTermination, error)
	FinalizeTerminatedRunnerTasks(ctx context.Context, fleetID string, runnerIDs []string) ([]LostRunnerTaskRecovery, error)
	RequestCancelTask(ctx context.Context, id string) (*models.Task, CancelOutcome, error)
	CompleteTask(ctx context.Context, req CompleteTaskRequest) (*CompleteTaskResult, error)
	ReapExpiredLeases(ctx context.Context) (requeued []ReapedLease, canceled []*models.Task, err error)

	// MarkTaskDispatched stamps dispatch_requested_at = now() after a
	// successful active dispatch call (e.g. Lambda invoke accepted).
	MarkTaskDispatched(ctx context.Context, taskID string) error
	// ClaimDispatchCandidates reserves up to limit queued tasks for the given
	// fleet provisioner whose dispatch_requested_at is unset or older than
	// staleAfter, stamping it to now() as part of the same query (so
	// concurrent broker replicas do not double-dispatch). Callers must still
	// call Dispatch; on failure the timestamp already advanced, which is the
	// intended backoff before the next sweep retries it.
	ClaimDispatchCandidates(ctx context.Context, provisioner string, staleAfter time.Duration, limit int) ([]DispatchCandidate, error)
}
