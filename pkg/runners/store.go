package runners

import "github.com/google/uuid"

type Store interface {
	CreateFleet(name, fleetURL, authToken string, labels []string) (*RunnerFleet, error)
	ListFleets() ([]RunnerFleet, error)
	FindFleet(id uuid.UUID) (*RunnerFleet, error)
	DeleteFleet(id uuid.UUID) error

	// CreateTask persists a runner task with the given pre-generated id.
	CreateTask(id uuid.UUID, fleetID uuid.UUID, fleetTaskID string, executionID uuid.UUID) (*RunnerTask, error)
	FindTask(id uuid.UUID) (*RunnerTask, error)
	FindTaskByExecutionID(executionID uuid.UUID) (*RunnerTask, error)
}
