package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	FactoryWorkOrderExecutionStatusPending  = "pending"
	FactoryWorkOrderExecutionStatusRunning  = "running"
	FactoryWorkOrderExecutionStatusFinished = "finished"
)

var (
	ErrFactoryWorkOrderExecutionNotFound = errors.New("factory work order execution not found")
	ErrFactoryWorkOrderNotDispatchable   = errors.New("work order cannot be dispatched in its current state")
)

type FactoryWorkOrderExecution struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	LineID         uuid.UUID
	// LineDispatchID is the parent traversal this step run belongs to. Its
	// steps snapshot is authoritative for what StepIndex refers to —
	// see FactoryWorkOrderLineDispatch. StepName is the automation
	// (canvas) name captured when the step started.
	LineDispatchID uuid.UUID
	StepIndex      int
	StepName       string
	RunID          *uuid.UUID
	Status         string
	Result         string
	// Aggregate usage populated by runners. Both default to zero; the API
	// only surfaces non-zero values to the UI.
	TotalTokens int64
	CostCents   int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	FinishedAt  *time.Time
}

func FindWorkOrderExecutionByRunID(tx *gorm.DB, runID uuid.UUID) (*FactoryWorkOrderExecution, error) {
	var execution FactoryWorkOrderExecution
	err := tx.Where("run_id = ?", runID).First(&execution).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryWorkOrderExecutionNotFound
		}
		return nil, err
	}

	return &execution, nil
}

func (e *FactoryWorkOrderExecution) MarkRunning(tx *gorm.DB) error {
	if e.Status != FactoryWorkOrderExecutionStatusPending {
		return nil
	}

	now := time.Now()
	e.Status = FactoryWorkOrderExecutionStatusRunning
	e.UpdatedAt = now

	return tx.Model(e).Updates(map[string]any{
		"status":     FactoryWorkOrderExecutionStatusRunning,
		"updated_at": now,
	}).Error
}

func (e *FactoryWorkOrderExecution) MarkFinished(tx *gorm.DB, result string) error {
	if e.Status == FactoryWorkOrderExecutionStatusFinished {
		return nil
	}

	now := time.Now()
	e.Status = FactoryWorkOrderExecutionStatusFinished
	e.Result = result
	e.UpdatedAt = now
	e.FinishedAt = &now

	if err := tx.Model(e).Updates(map[string]any{
		"status":      FactoryWorkOrderExecutionStatusFinished,
		"result":      result,
		"updated_at":  now,
		"finished_at": &now,
	}).Error; err != nil {
		return err
	}

	return e.RecordFinished(tx, result)
}

func (e *FactoryWorkOrderExecution) RecordFinished(tx *gorm.DB, result string) error {
	f, err := FindFactory(tx, e.OrganizationID, e.FactoryID)
	if err != nil {
		return err
	}

	order, err := f.FindWorkOrder(tx, e.WorkOrderID)
	if err != nil {
		return err
	}

	// The line ref comes from the dispatch's snapshot, not the live line —
	// this event is a historical fact about the traversal, so a line
	// rename/edit after dispatch shouldn't change what it says.
	dispatch, err := FindWorkOrderLineDispatch(tx, e.LineDispatchID)
	if err != nil {
		return err
	}

	if e.RunID == nil {
		return fmt.Errorf("work order execution %s has no run", e.ID)
	}

	run, err := LockCanvasRunInTransaction(tx, *e.RunID)
	if err != nil {
		return err
	}

	data := factory.LineStepExecutionFinished{
		StepName: e.StepName,
		Order:    order.Ref(),
		Line:     dispatch.Ref(),
		App:      &factory.AppRef{ID: run.WorkflowID, Name: e.StepName},
		Run:      &factory.RunRef{ID: run.ID, State: run.State, Result: &run.Result},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	event := &FactoryWorkOrderEvent{
		ID:          uuid.New(),
		WorkOrderID: e.WorkOrderID,
		Type:        factory.EventTypeLineStepExecutionFinished,
		Data:        datatypes.JSON(jsonData),
		CreatedAt:   time.Now(),
	}

	return tx.Create(event).Error
}

// FactoryWorkOrderExecutionRecord is a step execution enriched with the
// canvas run details the API/UI need to render it. Its line/steps
// information now lives on the parent FactoryWorkOrderLineDispatch instead
// of being joined/serialized per execution.
type FactoryWorkOrderExecutionRecord struct {
	FactoryWorkOrderExecution
	CanvasID   *uuid.UUID
	CanvasName string
	RunState   string
	RunResult  string
}

// ListFactoryWorkOrderExecutionsByLineDispatchIDs bulk-loads step executions
// for the given line dispatches, keyed by line_dispatch_id.
func ListFactoryWorkOrderExecutionsByLineDispatchIDs(
	tx *gorm.DB,
	lineDispatchIDs []uuid.UUID,
) (map[uuid.UUID][]FactoryWorkOrderExecutionRecord, error) {
	result := make(map[uuid.UUID][]FactoryWorkOrderExecutionRecord, len(lineDispatchIDs))
	if len(lineDispatchIDs) == 0 {
		return result, nil
	}

	var records []FactoryWorkOrderExecutionRecord
	err := tx.
		Table("factory_work_order_executions AS e").
		Select(`
			e.*,
			c.id AS canvas_id,
			COALESCE(c.name, '') AS canvas_name,
			COALESCE(r.state, '') AS run_state,
			COALESCE(r.result, '') AS run_result
		`).
		Joins("LEFT JOIN workflow_runs r ON r.id = e.run_id").
		Joins("LEFT JOIN workflows c ON c.id = r.workflow_id").
		Where("e.line_dispatch_id IN ?", lineDispatchIDs).
		Order("e.created_at ASC").
		Order("e.id ASC").
		Scan(&records).
		Error
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		result[record.LineDispatchID] = append(result[record.LineDispatchID], record)
	}

	return result, nil
}

func factoryWorkOrderRunInput(tx *gorm.DB, order *FactoryWorkOrder) (map[string]any, error) {
	workOrder := map[string]any{
		"id":          order.ID.String(),
		"title":       order.Title,
		"description": order.Description,
		"factory_id":  order.FactoryID.String(),
	}

	if order.SourceRunID != nil {
		rootEvent, err := FindRootEventForRun(tx, *order.SourceRunID)
		if err != nil {
			return nil, err
		}

		if rootEvent != nil {
			if source := RootEventSourcePayload(rootEvent.Data.Data()); source != nil {
				workOrder["source"] = source
			}
		}
	}

	return map[string]any{
		"work_order": workOrder,
	}, nil
}

// RootEventSourcePayload peels a root event envelope down to its `.data`
// payload — used as work-order `source` for both run input and order().
func RootEventSourcePayload(eventData any) any {
	payload, ok := eventData.(map[string]any)
	if !ok {
		return eventData
	}

	source, ok := payload["data"]
	if !ok {
		return eventData
	}

	return source
}
