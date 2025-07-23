package models

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
)

type StageExecutor struct {
	ID         uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	StageID    uuid.UUID
	ResourceID uuid.UUID
	Type       string
	Spec       datatypes.JSONType[ExecutorSpec]
}

type ExecutorSpec struct {
	Semaphore *SemaphoreExecutorSpec `json:"semaphore,omitempty"`
	HTTP      *HTTPExecutorSpec      `json:"http,omitempty"`
}

type SemaphoreExecutorSpec struct {
	// TODO: not exactly sure we should store this here
	// or if we should have the resource referenced by this executor
	// be a task instead of a project.
	TaskId *string `json:"task_id,omitempty"`

	Branch       string            `json:"branch"`
	PipelineFile string            `json:"pipeline_file"`
	Parameters   map[string]string `json:"parameters"`
}

type HTTPExecutorSpec struct {
	URL            string              `json:"url"`
	Payload        map[string]string   `json:"payload"`
	Headers        map[string]string   `json:"headers"`
	ResponsePolicy *HTTPResponsePolicy `json:"success_policy"`
}

type HTTPResponsePolicy struct {
	StatusCodes []uint32 `json:"status_codes"`
}

func (e *StageExecutor) GetResource() (*Resource, error) {
	var resource Resource

	err := database.Conn().
		Where("id = ?", e.ResourceID).
		First(&resource).
		Error

	if err != nil {
		return nil, err
	}

	return &resource, nil
}

func (e *StageExecutor) GetIntegrationResource() (*IntegrationResource, error) {
	var r IntegrationResource

	err := database.Conn().
		Table("resources").
		Joins("INNER JOIN integrations ON integrations.id = resources.integration_id").
		Select("resources.name as name, integrations.name as integration_name, integrations.domain_type as domain_type").
		Where("resources.id = ?", e.ResourceID).
		First(&r).
		Error

	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (e *StageExecutor) FindIntegration() (*Integration, error) {
	var integration Integration

	err := database.Conn().
		Table("stage_executors").
		Joins("INNER JOIN resources ON resources.id = stage_executors.resource_id").
		Joins("INNER JOIN integrations ON integrations.id = resources.integration_id").
		Where("stage_executors.id = ?", e.ID).
		Select("integrations.*").
		First(&integration).
		Error

	if err != nil {
		return nil, err
	}

	return &integration, nil
}
