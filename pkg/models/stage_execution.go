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

	//
	// TODO: not so sure about this column
	// TODO: maybe we can use a special execution tag for this?
	// The ID of the "thing" that is running.
	// For now, since we only have workflow/task runs,
	// this is always a Semaphore workflow ID, but we might want to support other types of executions in the future,
	// so keeping the name generic for now, and also not using uuid.UUID for this column, since we can't guarantee
	// that all IDs will be UUIDs.
	//
	ReferenceID string

	// TODO: I'm storing the extra tags pushed from the execution here,
	// but I'm not sure if that is the best place for it
	Tags datatypes.JSON
}

func (e *StageExecution) GetEventData() (map[string]any, error) {
	var data struct {
		Raw datatypes.JSON
	}

	err := database.Conn().
		Table("stage_executions").
		Select("events.raw").
		Joins("inner join stage_events ON stage_executions.stage_event_id = stage_events.id").
		Joins("inner join events ON stage_events.event_id = events.id").
		Where("stage_executions.id = ?", e.ID).
		Scan(&data).
		Error

	if err != nil {
		return nil, fmt.Errorf("error finding event: %v", err)
	}

	var m map[string]any
	err = json.Unmarshal(data.Raw, &m)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling data: %v", err)
	}

	return m, nil
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

func (e *StageExecution) Start(referenceID string) error {
	now := time.Now()

	return database.Conn().Model(e).
		Update("reference_id", referenceID).
		Update("state", StageExecutionStarted).
		Update("started_at", &now).
		Update("updated_at", &now).
		Error
}

func (e *StageExecution) FinishInTransaction(tx *gorm.DB, result string) error {
	now := time.Now()

	return tx.Model(e).
		Clauses(clause.Returning{}).
		Update("result", result).
		Update("state", StageExecutionFinished).
		Update("updated_at", &now).
		Update("finished_at", &now).
		Error
}

func (e *StageExecution) AddTags(tags []byte) error {
	return database.Conn().Model(e).
		Clauses(clause.Returning{}).
		Update("tags", tags).
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

func CreateStageExecution(stageID, stageEventID uuid.UUID) (*StageExecution, error) {
	return CreateStageExecutionInTransaction(database.Conn(), stageID, stageEventID)
}

func CreateStageExecutionInTransaction(tx *gorm.DB, stageID, stageEventID uuid.UUID) (*StageExecution, error) {
	now := time.Now()
	execution := StageExecution{
		StageID:      stageID,
		StageEventID: stageEventID,
		State:        StageExecutionPending,
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
