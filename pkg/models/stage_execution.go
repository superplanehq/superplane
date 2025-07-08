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
	StageExecutionPending  = "pending"
	StageExecutionStarted  = "started"
	StageExecutionFinished = "finished"

	StageExecutionResultPassed = "passed"
	StageExecutionResultFailed = "failed"
)

type StageExecution struct {
	ID           uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	StageID      uuid.UUID
	StageEventID uuid.UUID
	State        string
	Result       string
	CreatedAt    *time.Time
	UpdatedAt    *time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
	Outputs      datatypes.JSONType[map[string]any]
	Message      string

	//
	// TODO: not so sure about this column
	// TODO: maybe we can use a special execution tag for this?
	// The ID of the "thing" that is running.
	// For now, since we only have workflow/task runs,
	// this is always a Semaphore workflow ID, but we want to support other types of executions in the future,
	// so keeping the name generic for now, and also not using uuid.UUID for this column, since we can't guarantee
	// that all IDs will be UUIDs.
	//
	ReferenceID string
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
		Update("state", StageExecutionStarted).
		Update("started_at", &now).
		Update("updated_at", &now).
		Error
}

func (e *StageExecution) StartWithReferenceID(referenceID string) error {
	now := time.Now()

	return database.Conn().Model(e).
		Update("reference_id", referenceID).
		Update("state", StageExecutionStarted).
		Update("started_at", &now).
		Update("updated_at", &now).
		Error
}

func (e *StageExecution) Finish(stage *Stage, result string) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		return e.FinishInTransaction(tx, stage, result)
	})
}

func (e *StageExecution) FinishInTransaction(tx *gorm.DB, stage *Stage, result string) error {
	now := time.Now()

	//
	// Update execution state.
	//
	err := tx.Model(e).
		Clauses(clause.Returning{}).
		Update("result", result).
		Update("state", StageExecutionFinished).
		Update("updated_at", &now).
		Update("finished_at", &now).
		Error

	if err != nil {
		return err
	}

	//
	// Update stage event state.
	//
	err = UpdateStageEventsInTransaction(
		tx, []string{e.StageEventID.String()}, StageEventStateProcessed, "",
	)

	if err != nil {
		return err
	}

	//
	// Create stage execution completion event
	//
	event, err := NewStageExecutionCompletion(e, e.Outputs.Data())
	if err != nil {
		return fmt.Errorf("error creating stage completion event: %v", err)
	}

	raw, err := json.Marshal(&event)
	if err != nil {
		return fmt.Errorf("error marshaling event: %v", err)
	}

	// Generate message for stage completion event
	var message string
	if result == StageExecutionResultPassed {
		message = fmt.Sprintf("Stage %s completed successfully", stage.Name)
	} else {
		message = fmt.Sprintf("Stage %s failed", stage.Name)
	}

	_, err = CreateEventInTransaction(tx, e.StageID, stage.Name, SourceTypeStage, raw, []byte(`{}`), message)
	if err != nil {
		return fmt.Errorf("error creating event: %v", err)
	}

	return nil
}

func (e *StageExecution) UpdateOutputs(outputs map[string]any) error {
	return database.Conn().Model(e).
		Clauses(clause.Returning{}).
		Update("outputs", datatypes.NewJSONType(outputs)).
		Update("updated_at", time.Now()).
		Error
}

func FindExecutionByReference(referenceId string) (*StageExecution, error) {
	var execution StageExecution

	err := database.Conn().
		Where("reference_id = ?", referenceId).
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
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

func ListStageExecutionsInState(state string) ([]StageExecution, error) {
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

func CreateStageExecution(stageID, stageEventID uuid.UUID, message string) (*StageExecution, error) {
	return CreateStageExecutionInTransaction(database.Conn(), stageID, stageEventID, message)
}

func CreateStageExecutionInTransaction(tx *gorm.DB, stageID, stageEventID uuid.UUID, message string) (*StageExecution, error) {
	now := time.Now()
	execution := StageExecution{
		StageID:      stageID,
		StageEventID: stageEventID,
		State:        StageExecutionPending,
		CreatedAt:    &now,
		UpdatedAt:    &now,
		Message:      message,
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
