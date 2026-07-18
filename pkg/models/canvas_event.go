package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
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
	Data        JSONValue
	ExecutionID *uuid.UUID
	RunID       uuid.UUID
	State       string
	CreatedAt   *time.Time
}

func (e *CanvasEvent) TableName() string {
	return "workflow_events"
}

func (e *CanvasEvent) BeforeCreate(tx *gorm.DB) error {
	if e.RunID != uuid.Nil {
		return nil
	}

	if e.ExecutionID != nil {
		var execution CanvasNodeExecution
		err := tx.
			Select("run_id").
			Where("id = ?", *e.ExecutionID).
			First(&execution).
			Error
		if err != nil {
			return err
		}

		e.RunID = execution.RunID
		return nil
	}

	run, err := CreateCanvasRunInTransaction(tx, e.WorkflowID, CanvasRunStateStarted, "")
	if err != nil {
		return err
	}

	e.RunID = run.ID
	return nil
}

func FindCanvasEvents(tx *gorm.DB, ids []string) ([]CanvasEvent, error) {
	if len(ids) == 0 {
		return []CanvasEvent{}, nil
	}

	var events []CanvasEvent
	err := tx.
		Where("id IN ?", ids).
		Find(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func FindCanvasEventsForExecutions(tx *gorm.DB, executionIDs []string) ([]CanvasEvent, error) {
	if len(executionIDs) == 0 {
		return []CanvasEvent{}, nil
	}

	var events []CanvasEvent
	err := tx.
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

func ListCanvasEventsByIDsInTransaction(tx *gorm.DB, ids []uuid.UUID) ([]CanvasEvent, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var events []CanvasEvent
	err := tx.Where("id IN ?", ids).Find(&events).Error
	if err != nil {
		return nil, err
	}

	return events, nil
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

func ListPendingCanvasEvents() ([]CanvasEvent, error) {
	var events []CanvasEvent
	query := database.Conn().
		Table("workflow_events").
		Select("workflow_events.*").
		Where("workflow_events.state = ?", CanvasEventStatePending)

	err := withActiveCanvas(query, "workflow_events.workflow_id").
		Find(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func LockCanvasEvent(tx *gorm.DB, id uuid.UUID) (*CanvasEvent, error) {
	var event CanvasEvent

	query := tx.
		Table("workflow_events").
		Select("workflow_events.*").
		Clauses(clause.Locking{
			Strength: lockingForUpdateNoKey,
			Table:    clause.Table{Name: "workflow_events"},
			Options:  "SKIP LOCKED",
		}).
		Where("workflow_events.id = ?", id).
		Where("workflow_events.state = ?", CanvasEventStatePending)

	err := withActiveCanvas(query, "workflow_events.workflow_id").
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

// FindLastEventPerNode finds the most recent event for each node in a workflow.
// Only returns events for nodes that have not been deleted.
func FindLastEventPerNode(tx *gorm.DB, canvasID uuid.UUID) ([]CanvasEvent, error) {
	var events []CanvasEvent
	err := tx.
		Raw(`
			SELECT we.*
			FROM workflow_nodes wn
			INNER JOIN LATERAL (
				SELECT *
				FROM workflow_events
				WHERE workflow_id = wn.workflow_id
				  AND node_id = wn.node_id
				ORDER BY created_at DESC
				LIMIT 1
			) we ON true
			WHERE wn.workflow_id = ?
			  AND wn.deleted_at IS NULL
		`, canvasID).
		Scan(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}
