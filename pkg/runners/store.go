package runners

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/runners/models"
)

type Store interface {
	CreateFleet(name, authToken string) (*models.RunnerFleet, error)
	ListFleets() ([]models.RunnerFleet, error)
	FindFleet(id uuid.UUID) (*models.RunnerFleet, error)
	FindFleetByAuthToken(token string) (*models.RunnerFleet, error)
	DeleteFleet(id uuid.UUID) error

	FindTask(id uuid.UUID) (*models.RunnerTask, error)
	FindTaskByExecutionID(executionID uuid.UUID) (*models.RunnerTask, error)

	EnqueueJob(fleetID, executionID uuid.UUID, spec models.JobSpec) (*models.RunnerTask, error)
	ClaimNextQueuedJob(fleetID uuid.UUID) (*models.RunnerTask, error)
	CompleteJob(taskID uuid.UUID, req models.FleetCompleteRequest) (*models.RunnerTask, error)
}
