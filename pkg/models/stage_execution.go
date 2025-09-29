package models

import (
	"encoding/json"
	"errors"
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

	ResultPassed    = "passed"
	ResultFailed    = "failed"
	ResultCancelled = "cancelled"

	ResultReasonError          = "error"
	ResultReasonTimeout        = "timeout"
	ResultReasonUser           = "user"
	ResultReasonMissingOutputs = "missing-outputs"
)

var (
	ErrExecutionAlreadyCancelled  = errors.New("execution already cancelled")
	ErrExecutionCannotBeCancelled = errors.New("execution cannot be cancelled")
)

type StageExecution struct {
	ID            uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	CanvasID      uuid.UUID
	StageID       uuid.UUID
	StageEventID  uuid.UUID
	State         string
	Result        string
	ResultReason  string
	ResultMessage string
	CreatedAt     *time.Time
	UpdatedAt     *time.Time
	StartedAt     *time.Time
	FinishedAt    *time.Time
	Outputs       datatypes.JSONType[map[string]any]
	CancelledAt   *time.Time
	CancelledBy   *uuid.UUID

	StageEvent *StageEvent `gorm:"foreignKey:StageEventID;references:ID"`
}

func (e *StageExecution) IsTimedOut(now time.Time, timeout time.Duration) bool {
	if e.StartedAt == nil {
		return false
	}

	runningDuration := now.Sub(*e.StartedAt)
	return runningDuration > timeout
}

func (e *StageExecution) Cancel(userID uuid.UUID) error {
	if e.State == ExecutionFinished {
		return ErrExecutionCannotBeCancelled
	}

	if e.CancelledAt != nil {
		return ErrExecutionAlreadyCancelled
	}

	now := time.Now()
	return database.Conn().Model(e).
		Clauses(clause.Returning{}).
		Update("updated_at", &now).
		Update("cancelled_at", &now).
		Update("cancelled_by", &userID).
		Error
}

func (e *StageExecution) GetInputs() (map[string]any, error) {
	return e.GetInputsInTransaction(database.Conn())
}

func (e *StageExecution) GetInputsInTransaction(tx *gorm.DB) (map[string]any, error) {
	var inputs datatypes.JSONType[map[string]any]

	err := tx.
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
	return e.StartInTransaction(database.Conn())
}

func (e *StageExecution) StartInTransaction(tx *gorm.DB) error {
	now := time.Now()

	return tx.Model(e).
		Update("state", ExecutionStarted).
		Update("started_at", &now).
		Update("updated_at", &now).
		Error
}

func (e *StageExecution) Finish(stage *Stage, result, reason, message string) (*Event, error) {
	var event *Event
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		event, err = e.FinishInTransaction(tx, stage, result, reason, message)
		return err
	})

	return event, err
}

func (e *StageExecution) FinishInTransaction(tx *gorm.DB, stage *Stage, result, reason, message string) (*Event, error) {
	now := time.Now()

	//
	// Update execution state.
	//
	err := tx.Model(e).
		Clauses(clause.Returning{}).
		Update("result", result).
		Update("result_reason", reason).
		Update("result_message", message).
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
	return e.UpdateOutputsInTransaction(database.Conn(), outputs)
}

func (e *StageExecution) UpdateOutputsInTransaction(tx *gorm.DB, outputs map[string]any) error {
	return tx.Model(e).
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

func (e *StageExecution) Finished(resources []*ExecutionResource) bool {
	for _, r := range resources {
		if !r.Finished() {
			return false
		}
	}

	return true
}

func (e *StageExecution) GetResult(stage *Stage, resources []*ExecutionResource) (string, string, string) {
	for _, r := range resources {
		if !r.Successful() {
			return ResultFailed, ResultReasonError, fmt.Sprintf("%s failed: %s", r.Type(), r.Id())
		}
	}

	missingOutputs := stage.MissingRequiredOutputs(e.Outputs.Data())
	if len(missingOutputs) > 0 {
		return ResultFailed, ResultReasonMissingOutputs, fmt.Sprintf("missing outputs: %v", missingOutputs)
	}

	return ResultPassed, "", ""
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

func FindExecutionByID(id, stageID uuid.UUID) (*StageExecution, error) {
	var execution StageExecution

	err := database.Conn().
		Where("id = ?", id).
		Where("stage_id = ?", stageID).
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func FindUnscopedExecutionByID(id uuid.UUID) (*StageExecution, error) {
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

func ListExecutionsInState(state string, limit int) ([]StageExecution, error) {
	var executions []StageExecution

	query := database.Conn().
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("state = ?", state)

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&executions).Error
	if err != nil {
		return nil, err
	}

	return executions, nil
}

func LockExecution(tx *gorm.DB, id uuid.UUID) (*StageExecution, error) {
	var execution StageExecution

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", id).
		Where("state = ?", ExecutionPending).
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
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
	ResourceType     string `gorm:"column:type"`
	State            string
	Result           string
	LastPolledAt     *time.Time
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
}

func (r *ExecutionResource) Finished() bool {
	return r.State == ExecutionResourceFinished
}

func (r *ExecutionResource) Successful() bool {
	return r.Result == ResultPassed
}

func (r *ExecutionResource) Id() string {
	return r.ExternalID
}

func (r *ExecutionResource) Type() string {
	return r.ResourceType
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

func (r *ExecutionResource) Finish(result string) error {
	return database.Conn().
		Model(r).
		Clauses(clause.Returning{}).
		Update("state", ExecutionResourceFinished).
		Update("result", result).
		Update("updated_at", time.Now()).
		Error
}

func (r *ExecutionResource) UpdatePollingMetadata() error {
	return database.Conn().
		Model(r).
		Clauses(clause.Returning{}).
		Update("last_polled_at", time.Now()).
		Update("updated_at", time.Now()).
		Error
}

func (r *ExecutionResource) ShouldPoll(pollDelay time.Duration) bool {
	if r.LastPolledAt == nil {
		return true
	}

	return time.Since(*r.LastPolledAt) >= pollDelay
}

func (e *StageExecution) Resources() ([]*ExecutionResource, error) {
	var resources []*ExecutionResource

	err := database.Conn().
		Where("execution_id = ?", e.ID).
		Find(&resources).
		Error

	if err != nil {
		return nil, err
	}

	return resources, nil
}

func (r *ExecutionResource) FindIntegration() (*Integration, error) {
	var integration Integration

	err := database.Conn().
		Table("resources").
		Joins("INNER JOIN integrations ON integrations.id = resources.integration_id").
		Where("resources.id = ?", r.ParentResourceID).
		Select("integrations.*").
		First(&integration).
		Error

	if err != nil {
		return nil, err
	}

	return &integration, nil
}

func (r *ExecutionResource) FindParentResource() (*Resource, error) {
	var resource Resource

	err := database.Conn().
		Where("id = ?", r.ParentResourceID).
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
	return e.AddResourceInTransaction(database.Conn(), externalID, externalType, parentResourceID)
}

func (e *StageExecution) AddResourceInTransaction(tx *gorm.DB, externalID string, externalType string, parentResourceID uuid.UUID) (*ExecutionResource, error) {
	r := &ExecutionResource{
		ExecutionID:      e.ID,
		StageID:          e.StageID,
		ParentResourceID: parentResourceID,
		ExternalID:       externalID,
		ResourceType:     externalType,
		State:            ExecutionResourcePending,
	}

	err := tx.
		Clauses(clause.Returning{}).
		Create(r).
		Error

	if err != nil {
		return nil, err
	}

	return r, nil
}

func DeleteStageExecutionsBySourceInTransaction(tx *gorm.DB, sourceID uuid.UUID, sourceType string) error {
	if err := tx.Unscoped().
		Where("execution_id IN (SELECT id FROM stage_executions WHERE stage_event_id IN (SELECT id FROM stage_events WHERE source_id = ? AND source_type = ?))", sourceID, sourceType).
		Delete(&ExecutionResource{}).Error; err != nil {
		return fmt.Errorf("failed to delete execution resources for source stage executions: %v", err)
	}

	if err := tx.Unscoped().
		Where("stage_event_id IN (SELECT id FROM stage_events WHERE source_id = ? AND source_type = ?)", sourceID, sourceType).
		Delete(&StageExecution{}).Error; err != nil {
		return fmt.Errorf("failed to delete stage executions for source stage events: %v", err)
	}

	return nil
}

func DeleteExecutionResourcesByParentResourceInTransaction(tx *gorm.DB, parentResourceID uuid.UUID) error {
	if err := tx.Unscoped().
		Where("parent_resource_id = ?", parentResourceID).
		Delete(&ExecutionResource{}).Error; err != nil {
		return fmt.Errorf("failed to delete execution resources for parent resource: %v", err)
	}
	return nil
}
