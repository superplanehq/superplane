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
	// TODO: no need to specify it here
	// we should get this from the resource and not from here
	ProjectID string `json:"project_id"`
	TaskID    string `json:"task_id"`

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
