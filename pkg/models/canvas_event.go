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
	CanvasEventStatePending = "pending"
	CanvasEventStateRouted  = "routed"
)

type CanvasEvent struct {
	ID          uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	WorkflowID  uuid.UUID
	NodeID      string
	Channel     string
	CustomName  *string
	Data        datatypes.JSONType[any]
	ExecutionID *uuid.UUID
	State       string
	CreatedAt   *time.Time
}

func (e *CanvasEvent) TableName() string {
	return "workflow_events"
}

func FindCanvasEvents(ids []string) ([]CanvasEvent, error) {
	var events []CanvasEvent
	err := database.Conn().
		Where("id IN ?", ids).
		Find(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func FindCanvasEventsForExecutions(executionIDs []string) ([]CanvasEvent, error) {
	var events []CanvasEvent
	err := database.Conn().
		Where("execution_id IN ?", executionIDs).
		Find(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func FindCanvasEventForCanvas(canvasID uuid.UUID, id uuid.UUID) (*CanvasEvent, error) {
	var event CanvasEvent
	err := database.Conn().
		Where("workflow_id = ?", canvasID).
		Where("id = ?", id).
		First(&event).
		Error

	if err != nil {
		return nil, err
	}

	return &event, nil
}

func FindCanvasEvent(id uuid.UUID) (*CanvasEvent, error) {
	return FindCanvasEventInTransaction(database.Conn(), id)
}

func FindCanvasEventInTransaction(tx *gorm.DB, id uuid.UUID) (*CanvasEvent, error) {
	var event CanvasEvent
	err := tx.
		Where("id = ?", id).
		First(&event).
		Error

	if err != nil {
		return nil, err
	}

	return &event, nil
}

func ListCanvasEvents(canvasID uuid.UUID, nodeID string, limit int, before *time.Time) ([]CanvasEvent, error) {
	var events []CanvasEvent
	query := database.Conn().
		Where("workflow_id = ?", canvasID).
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

func CountCanvasEvents(canvasID uuid.UUID, nodeID string) (int64, error) {
	var count int64

	err := database.Conn().
		Model(&CanvasEvent{}).
		Where("workflow_id = ?", canvasID).
		Where("node_id = ?", nodeID).
		Count(&count).
		Error

	if err != nil {
		return 0, err
	}

	return count, nil
}

func ListRootCanvasEvents(canvasID uuid.UUID, limit int, before *time.Time) ([]CanvasEvent, error) {
	var events []CanvasEvent
	query := database.Conn().
		Where("workflow_id = ?", canvasID).
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

func CountRootCanvasEvents(canvasID uuid.UUID) (int64, error) {
	var count int64

	err := database.Conn().
		Model(&CanvasEvent{}).
		Where("workflow_id = ?", canvasID).
		Where("execution_id IS NULL").
		Count(&count).
		Error

	if err != nil {
		return 0, err
	}

	return count, nil
}

func ListPendingCanvasEvents() ([]CanvasEvent, error) {
	var events []CanvasEvent
	err := database.Conn().
		Joins("JOIN workflows ON workflow_events.workflow_id = workflows.id").
		Where("workflow_events.state = ?", CanvasEventStatePending).
		Where("workflows.deleted_at IS NULL").
		Find(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func LockCanvasEvent(tx *gorm.DB, id uuid.UUID) (*CanvasEvent, error) {
	var event CanvasEvent

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", id).
		Where("state = ?", CanvasEventStatePending).
		First(&event).
		Error

	if err != nil {
		return nil, err
	}

	return &event, nil
}

func (e *CanvasEvent) Routed() error {
	return e.RoutedInTransaction(database.Conn())
}

func (e *CanvasEvent) RoutedInTransaction(tx *gorm.DB) error {
	e.State = CanvasEventStateRouted
	return tx.Save(e).Error
}

// FindLastEventPerNode finds the most recent event for each node in a workflow
// using DISTINCT ON to get one event per node_id, ordered by created_at DESC
// Only returns events for nodes that have not been deleted
func FindLastEventPerNode(canvasID uuid.UUID) ([]CanvasEvent, error) {
	var events []CanvasEvent
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
		`, canvasID).
		Scan(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}
