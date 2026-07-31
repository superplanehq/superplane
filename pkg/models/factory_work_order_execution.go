package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	FactoryWorkOrderExecutionStatusPending  = "pending"
	FactoryWorkOrderExecutionStatusRunning  = "running"
	FactoryWorkOrderExecutionStatusFinished = "finished"
)

var (
	ErrFactoryWorkOrderExecutionNotFound = errors.New("factory work order execution not found")
	ErrFactoryWorkOrderLineActive        = errors.New("work order already has an active execution on this line")
	ErrFactoryWorkOrderNotOpen           = errors.New("work order is not open")
)

type FactoryWorkOrderExecution struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
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

func (FactoryWorkOrderExecution) TableName() string {
	return "factory_work_order_executions"
}

func FindFactoryWorkOrderExecutionByRunID(tx *gorm.DB, runID uuid.UUID) (*FactoryWorkOrderExecution, error) {
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

func FindActiveFactoryWorkOrderExecution(
	tx *gorm.DB,
	workOrderID, lineID uuid.UUID,
) (*FactoryWorkOrderExecution, error) {
	var execution FactoryWorkOrderExecution
	err := tx.
		Where("work_order_id = ? AND line_id = ?", workOrderID, lineID).
		Where("status IN ?", []string{
			FactoryWorkOrderExecutionStatusPending,
			FactoryWorkOrderExecutionStatusRunning,
		}).
		First(&execution).
		Error
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

	return tx.Model(e).Updates(map[string]any{
		"status":      FactoryWorkOrderExecutionStatusFinished,
		"result":      result,
		"updated_at":  now,
		"finished_at": &now,
	}).Error
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
