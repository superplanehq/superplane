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

func FindWorkflowEvents(ids []string) ([]WorkflowEvent, error) {
	var events []WorkflowEvent
	err := database.Conn().
		Where("id IN ?", ids).
		Find(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func FindWorkflowEventsForExecutions(executionIDs []string) ([]WorkflowEvent, error) {
	var events []WorkflowEvent
	err := database.Conn().
		Where("execution_id IN ?", executionIDs).
		Find(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func FindWorkflowEventForWorkflow(workflowID uuid.UUID, id uuid.UUID) (*WorkflowEvent, error) {
	var event WorkflowEvent
	err := database.Conn().
		Where("workflow_id = ?", workflowID).
		Where("id = ?", id).
		First(&event).
		Error

	if err != nil {
		return nil, err
	}

	return &event, nil
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

func ListWorkflowEvents(workflowID uuid.UUID, nodeID string, limit int, before *time.Time) ([]WorkflowEvent, error) {
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

func CountWorkflowEvents(workflowID uuid.UUID, nodeID string) (int64, error) {
	var count int64

	err := database.Conn().
		Model(&WorkflowEvent{}).
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID).
		Count(&count).
		Error

	if err != nil {
		return 0, err
	}

	return count, nil
}

func ListRootWorkflowEvents(workflowID uuid.UUID, limit int, before *time.Time) ([]WorkflowEvent, error) {
	var events []WorkflowEvent
	query := database.Conn().
		Where("workflow_id = ?", workflowID).
		Where("execution_id IS NULL")

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

func CountRootWorkflowEvents(workflowID uuid.UUID) (int64, error) {
	var count int64

	err := database.Conn().
		Model(&WorkflowEvent{}).
		Where("workflow_id = ?", workflowID).
		Where("execution_id IS NULL").
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
		Joins("JOIN workflows ON workflow_events.workflow_id = workflows.id").
		Where("workflow_events.state = ?", WorkflowEventStatePending).
		Where("workflows.deleted_at IS NULL").
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

// FindLastEventPerNode finds the most recent event for each node in a workflow
// using DISTINCT ON to get one event per node_id, ordered by created_at DESC
// Only returns events for nodes that have not been deleted
func FindLastEventPerNode(workflowID uuid.UUID) ([]WorkflowEvent, error) {
	var events []WorkflowEvent
	err := database.Conn().
		Raw(`
			SELECT DISTINCT ON (we.node_id) we.*
			FROM workflow_events we
			INNER JOIN workflow_nodes wn
				ON we.workflow_id = wn.workflow_id
				AND we.node_id = wn.node_id
			WHERE we.workflow_id = ?
			AND wn.deleted_at IS NULL
			ORDER BY we.node_id, we.created_at DESC
		`, workflowID).
		Scan(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}
