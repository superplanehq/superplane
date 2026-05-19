package runners

import "github.com/google/uuid"

type Store interface {
	CreateFleet(name, mode, fleetURL, authToken string, labels []string) (*RunnerFleet, error)
	ListFleets() ([]RunnerFleet, error)
	FindFleet(id uuid.UUID) (*RunnerFleet, error)
	FindFleetByAuthToken(token string) (*RunnerFleet, error)
	DeleteFleet(id uuid.UUID) error

	CreateTask(id uuid.UUID, fleetID uuid.UUID, fleetTaskID string, executionID uuid.UUID) (*RunnerTask, error)
	FindTask(id uuid.UUID) (*RunnerTask, error)
	FindTaskByExecutionID(executionID uuid.UUID) (*RunnerTask, error)

	EnqueueJob(fleetID, executionID uuid.UUID, spec JobSpec) (*RunnerTask, error)
	ClaimNextQueuedJob(fleetID uuid.UUID) (*RunnerTask, error)
	CompleteJob(taskID uuid.UUID, req FleetCompleteRequest) (*RunnerTask, error)
}
