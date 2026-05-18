package runners

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type postgresStore struct{}

func NewPostgresStore() Store {
	return &postgresStore{}
}

func (s *postgresStore) db() *gorm.DB {
	return database.Conn()
}

func (s *postgresStore) CreateFleet(name, fleetURL, authToken string, labels []string) (*RunnerFleet, error) {
	if labels == nil {
		labels = []string{}
	}
	fleet := &RunnerFleet{
		Name:      name,
		FleetURL:  fleetURL,
		AuthToken: authToken,
		Labels:    datatypes.NewJSONType(labels),
	}
	if err := s.db().Create(fleet).Error; err != nil {
		return nil, err
	}
	return fleet, nil
}

func (s *postgresStore) ListFleets() ([]RunnerFleet, error) {
	var fleets []RunnerFleet
	if err := s.db().Order("created_at ASC").Find(&fleets).Error; err != nil {
		return nil, err
	}
	return fleets, nil
}

func (s *postgresStore) FindFleet(id uuid.UUID) (*RunnerFleet, error) {
	var fleet RunnerFleet
	if err := s.db().First(&fleet, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &fleet, nil
}

func (s *postgresStore) DeleteFleet(id uuid.UUID) error {
	return s.db().Delete(&RunnerFleet{}, "id = ?", id).Error
}

func (s *postgresStore) CreateTask(id uuid.UUID, fleetID uuid.UUID, fleetTaskID string, executionID uuid.UUID) (*RunnerTask, error) {
	task := &RunnerTask{
		ID:          id,
		FleetID:     fleetID,
		FleetTaskID: fleetTaskID,
		ExecutionID: executionID,
	}
	if err := s.db().Create(task).Error; err != nil {
		return nil, err
	}
	return task, nil
}

func (s *postgresStore) FindTask(id uuid.UUID) (*RunnerTask, error) {
	var task RunnerTask
	if err := s.db().First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *postgresStore) FindTaskByExecutionID(executionID uuid.UUID) (*RunnerTask, error) {
	var task RunnerTask
	if err := s.db().Where("execution_id = ?", executionID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}
