package models

import (
	"encoding/json"
	"errors"
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
	ErrFactoryWorkOrderExecutionActive   = errors.New("work order already has an active execution")
	ErrFactoryWorkOrderNotOpen           = errors.New("work order is not open")
)

type FactoryWorkOrderExecution struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	LineID         uuid.UUID
	StepIndex      int
	StepName       string
	RunID          uuid.UUID
	Status         string
	Result         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	FinishedAt     *time.Time
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

	line, err := f.FindLine(tx, e.LineID)
	if err != nil {
		return err
	}

	order, err := f.FindWorkOrder(tx, e.WorkOrderID)
	if err != nil {
		return err
	}

	run, err := LockCanvasRunInTransaction(tx, e.RunID)
	if err != nil {
		return err
	}

	data := factory.LineStepExecutionFinished{
		StepName: e.StepName,
		Order:    order.Ref(),
		Line:     &factory.LineRef{ID: line.ID, Name: line.Name},
		App:      &factory.AppRef{ID: run.WorkflowID},
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

type FactoryWorkOrderExecutionRecord struct {
	FactoryWorkOrderExecution
	LineName   string
	CanvasID   uuid.UUID
	CanvasName string
	RunState   string
	RunResult  string
}

func ListFactoryWorkOrderExecutionsByWorkOrderIDs(
	tx *gorm.DB,
	workOrderIDs []uuid.UUID,
) (map[uuid.UUID][]FactoryWorkOrderExecutionRecord, error) {
	result := make(map[uuid.UUID][]FactoryWorkOrderExecutionRecord, len(workOrderIDs))
	if len(workOrderIDs) == 0 {
		return result, nil
	}

	var records []FactoryWorkOrderExecutionRecord
	err := tx.
		Table("factory_work_order_executions AS e").
		Select(`
			e.*,
			l.name AS line_name,
			c.id AS canvas_id,
			c.name AS canvas_name,
			r.state AS run_state,
			r.result AS run_result
		`).
		Joins("JOIN factory_lines l ON l.id = e.line_id").
		Joins("JOIN workflow_runs r ON r.id = e.run_id").
		Joins("JOIN workflows c ON c.id = r.workflow_id").
		Where("e.work_order_id IN ?", workOrderIDs).
		Order("e.created_at ASC").
		Order("e.id ASC").
		Scan(&records).
		Error
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		result[record.WorkOrderID] = append(result[record.WorkOrderID], record)
	}

	return result, nil
}

func factoryWorkOrderRunInput(order *FactoryWorkOrder) map[string]any {
	return map[string]any{
		"work_order": map[string]any{
			"id":          order.ID.String(),
			"title":       order.Title,
			"description": order.Description,
			"factory_id":  order.FactoryID.String(),
		},
	}
}
