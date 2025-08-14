package models

import (
	"encoding/json"
	"fmt"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ExecutionPending  = "pending"
	ExecutionStarted  = "started"
	ExecutionFinished = "finished"

	ExecutionResourcePending  = "pending"
	ExecutionResourceFinished = "finished"

	ResultPassed = "passed"
	ResultFailed = "failed"
)

type StageExecution struct {
	ID           uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	CanvasID     uuid.UUID
	StageID      uuid.UUID
	StageEventID uuid.UUID
	State        string
	Result       string
	CreatedAt    *time.Time
	UpdatedAt    *time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
	Outputs      datatypes.JSONType[map[string]any]
}

func (e *StageExecution) GetInputs() (map[string]any, error) {
	var inputs datatypes.JSONType[map[string]any]

	err := database.Conn().
		Table("stage_executions").
		Select("stage_events.inputs").
		Joins("inner join stage_events ON stage_executions.stage_event_id = stage_events.id").
		Where("stage_executions.id = ?", e.ID).
		Scan(&inputs).
		Error

	if err != nil {
		return nil, fmt.Errorf("error finding event: %v", err)
	}

	return inputs.Data(), nil
}

func (e *StageExecution) FindSource() (string, error) {
	var sourceName string
	err := database.Conn().
		Table("stage_executions").
		Select("stage_events.source_name").
		Joins("inner join stage_events ON stage_executions.stage_event_id = stage_events.id").
		Where("stage_executions.id = ?", e.ID).
		Scan(&sourceName).
		Error

	if err != nil {
		return "", err
	}

	return sourceName, nil
}

func (e *StageExecution) Start() error {
	now := time.Now()

	return database.Conn().Model(e).
		Update("state", ExecutionStarted).
		Update("started_at", &now).
		Update("updated_at", &now).
		Error
}

func (e *StageExecution) Finish(stage *Stage, result string) (*Event, error) {
	var event *Event
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		event, err = e.FinishInTransaction(tx, stage, result)
		return err
	})

	return event, err
}

func (e *StageExecution) FinishInTransaction(tx *gorm.DB, stage *Stage, result string) (*Event, error) {
	now := time.Now()

	//
	// Update execution state.
	//
	err := tx.Model(e).
		Clauses(clause.Returning{}).
		Update("result", result).
		Update("state", ExecutionFinished).
		Update("updated_at", &now).
		Update("finished_at", &now).
		Error

	if err != nil {
		return nil, err
	}

	//
	// Update stage event state.
	//
	err = UpdateStageEventsInTransaction(
		tx, []string{e.StageEventID.String()}, StageEventStateProcessed, "",
	)

	if err != nil {
		return nil, err
	}

	inputs, err := e.GetInputs()
	if err != nil {
		return nil, err
	}

	//
	// Create stage execution completion event
	//
	event, err := NewExecutionCompletionEvent(e, inputs, e.Outputs.Data())
	if err != nil {
		return nil, fmt.Errorf("error creating stage completion event: %v", err)
	}

	raw, err := json.Marshal(&event)
	if err != nil {
		return nil, fmt.Errorf("error marshaling event: %v", err)
	}

	createdEvent, err := CreateEventInTransaction(tx, e.StageID, stage.CanvasID, stage.Name, SourceTypeStage, event.Type, raw, []byte(`{}`))
	if err != nil {
		return nil, fmt.Errorf("error creating event: %v", err)
	}

	return createdEvent, nil
}

func (e *StageExecution) UpdateOutputs(outputs map[string]any) error {
	return database.Conn().Model(e).
		Clauses(clause.Returning{}).
		Update("outputs", datatypes.NewJSONType(outputs)).
		Update("updated_at", time.Now()).
		Error
}

type ExecutionIntegrationResource struct {
	IntegrationType     string
	IntegrationURL      string
	ParentExternalID    string
	ExecutionExternalID string
}

func (e *StageExecution) IntegrationResource(externalID string) (*ExecutionIntegrationResource, error) {
	var r ExecutionIntegrationResource
	err := database.Conn().
		Table("execution_resources").
		Joins("INNER JOIN resources ON resources.id = execution_resources.parent_resource_id").
		Joins("INNER JOIN integrations ON integrations.id = resources.integration_id").
		Where("execution_resources.execution_id = ?", e.ID).
		Where("execution_resources.external_id = ?", externalID).
		Select(`
			integrations.url as integration_url,
			integrations.type as integration_type,
			execution_resources.external_id as execution_external_id,
			resources.external_id as parent_external_id
		`).
		Scan(&r).
		Error

	if err != nil {
		return nil, err
	}

	return &r, nil
}

func FindExecutionByID(id uuid.UUID) (*StageExecution, error) {
	var execution StageExecution

	err := database.Conn().
		Where("id = ?", id).
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func FindExecutionByStageEventID(id uuid.UUID) (*StageExecution, error) {
	var execution StageExecution

	err := database.Conn().
		Where("stage_event_id = ?", id).
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func FindExecutionInState(stageID uuid.UUID, states []string) (*StageExecution, error) {
	var execution StageExecution

	err := database.Conn().
		Where("stage_id = ?", stageID).
		Where("state IN ?", states).
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func ListExecutionsInState(state string) ([]StageExecution, error) {
	var executions []StageExecution

	err := database.Conn().
		Where("state = ?", state).
		Find(&executions).
		Error

	if err != nil {
		return nil, err
	}

	return executions, nil
}

func CreateStageExecution(canvasID, stageID, stageEventID uuid.UUID) (*StageExecution, error) {
	return CreateStageExecutionInTransaction(database.Conn(), canvasID, stageID, stageEventID)
}

func CreateStageExecutionInTransaction(tx *gorm.DB, canvasID, stageID, stageEventID uuid.UUID) (*StageExecution, error) {
	now := time.Now()
	execution := StageExecution{
		CanvasID:     canvasID,
		StageID:      stageID,
		StageEventID: stageEventID,
		State:        ExecutionPending,
		CreatedAt:    &now,
		UpdatedAt:    &now,
	}

	err := tx.
		Clauses(clause.Returning{}).
		Create(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}

type ExecutionResource struct {
	ID               uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	ExecutionID      uuid.UUID
	StageID          uuid.UUID
	ParentResourceID uuid.UUID
	ExternalID       string
	Type             string
	State            string
	Result           string
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
}

func PendingExecutionResources() ([]ExecutionResource, error) {
	var resources []ExecutionResource

	err := database.Conn().
		Where("state = ?", ExecutionResourcePending).
		Find(&resources).
		Error

	if err != nil {
		return nil, err
	}

	return resources, nil
}

func (e *ExecutionResource) Finish(result string) error {
	return database.Conn().
		Model(e).
		Clauses(clause.Returning{}).
		Update("state", ExecutionResourceFinished).
		Update("result", result).
		Update("updated_at", time.Now()).
		Error
}

func (e *StageExecution) Resources() ([]ExecutionResource, error) {
	var resources []ExecutionResource

	err := database.Conn().
		Where("execution_id = ?", e.ID).
		Find(&resources).
		Error

	if err != nil {
		return nil, err
	}

	return resources, nil
}

func (e *ExecutionResource) FindIntegration() (*Integration, error) {
	var integration Integration

	err := database.Conn().
		Table("resources").
		Joins("INNER JOIN integrations ON integrations.id = resources.integration_id").
		Where("resources.id = ?", e.ParentResourceID).
		Select("integrations.*").
		First(&integration).
		Error

	if err != nil {
		return nil, err
	}

	return &integration, nil
}

func (e *ExecutionResource) FindParentResource() (*Resource, error) {
	var resource Resource

	err := database.Conn().
		Where("id = ?", e.ParentResourceID).
		First(&resource).
		Error

	if err != nil {
		return nil, err
	}

	return &resource, nil
}

func FindExecutionResource(externalID string, parentResourceID uuid.UUID) (*ExecutionResource, error) {
	var resource ExecutionResource

	err := database.Conn().
		Where("external_id = ?", externalID).
		Where("parent_resource_id = ?", parentResourceID).
		First(&resource).
		Error

	if err != nil {
		return nil, err
	}

	return &resource, nil
}

func (e *StageExecution) AddResource(externalID string, externalType string, parentResourceID uuid.UUID) (*ExecutionResource, error) {
	r := &ExecutionResource{
		ExecutionID:      e.ID,
		StageID:          e.StageID,
		ParentResourceID: parentResourceID,
		ExternalID:       externalID,
		Type:             externalType,
		State:            ExecutionResourcePending,
	}

	err := database.Conn().
		Clauses(clause.Returning{}).
		Create(r).
		Error

	if err != nil {
		return nil, err
	}

	return r, nil
}
