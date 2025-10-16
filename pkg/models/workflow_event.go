package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	WorkflowEventStatePending = "pending"
	WorkflowEventStateRouted  = "routed"
)

type WorkflowEvent struct {
	ID          uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	WorkflowID  uuid.UUID
	NodeID      string
	Channel     string
	Data        datatypes.JSONType[any]
	ExecutionID *uuid.UUID
	State       string
	CreatedAt   *time.Time
}

func FindWorkflowEvent(id uuid.UUID) (*WorkflowEvent, error) {
	return FindWorkflowEventInTransaction(database.Conn(), id)
}

func FindWorkflowEventInTransaction(tx *gorm.DB, id uuid.UUID) (*WorkflowEvent, error) {
	var event WorkflowEvent
	err := tx.
		Where("id = ?", id).
		First(&event).
		Error

	if err != nil {
		return nil, err
	}

	return &event, nil
}

func ListWorkflowEventsForNode(workflowID uuid.UUID, nodeID string, limit int, before *time.Time) ([]WorkflowEvent, error) {
	var events []WorkflowEvent
	query := database.Conn().
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID)

	if limit > 0 {
		query = query.Limit(limit)
	}

	if before != nil {
		query = query.Where("created_at < ?", before)
	}

	err := query.Order("created_at DESC").Find(&events).Error
	if err != nil {
		return nil, err
	}

	return events, nil
}

func CountWorkflowEventsForNode(workflowID uuid.UUID, nodeID string) (int64, error) {
	var count int64

	err := database.Conn().
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID).
		Count(&count).
		Error

	if err != nil {
		return 0, err
	}

	return count, nil
}

func ListPendingWorkflowEvents() ([]WorkflowEvent, error) {
	var events []WorkflowEvent
	err := database.Conn().
		Where("state = ?", WorkflowEventStatePending).
		Find(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func LockWorkflowEvent(tx *gorm.DB, id uuid.UUID) (*WorkflowEvent, error) {
	var event WorkflowEvent

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", id).
		Where("state = ?", WorkflowEventStatePending).
		First(&event).
		Error

	if err != nil {
		return nil, err
	}

	return &event, nil
}

func (e *WorkflowEvent) Routed() error {
	return e.RoutedInTransaction(database.Conn())
}

func (e *WorkflowEvent) RoutedInTransaction(tx *gorm.DB) error {
	e.State = WorkflowEventStateRouted
	return tx.Save(e).Error
}
