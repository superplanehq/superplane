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

func LockExpiredRoutedRootCanvasEventsInTransaction(tx *gorm.DB, referenceTime time.Time, limit int) ([]CanvasEvent, error) {
	var events []CanvasEvent

	query := expiredRoutedRootCanvasEventsQuery(tx, referenceTime).
		Scopes(
			lockCanvasEventsForUpdate,
			withoutRootEventQueueItems,
			withoutActiveRootEventExecutions,
			withoutPendingRootEventRequests,
			oldestCanvasEventsFirst,
		)

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&events).Error
	if err != nil {
		return nil, err
	}

	return events, nil
}

func expiredRoutedRootCanvasEventsQuery(tx *gorm.DB, referenceTime time.Time) *gorm.DB {
	query := tx.
		Table("workflow_events").
		Select("workflow_events.*").
		Where("organizations.usage_retention_window_days IS NOT NULL").
		Where("organizations.usage_retention_window_days > 0").
		Where("workflow_events.execution_id IS NULL").
		Where("workflow_events.state = ?", CanvasEventStateRouted).
		Where("workflow_events.created_at + (organizations.usage_retention_window_days * INTERVAL '1 day') < ?", referenceTime.UTC())

	return withActiveCanvas(query, "workflow_events.workflow_id")
}

func lockCanvasEventsForUpdate(tx *gorm.DB) *gorm.DB {
	return tx.Clauses(clause.Locking{
		Strength: "UPDATE",
		Table:    clause.Table{Name: "workflow_events"},
		Options:  "SKIP LOCKED",
	})
}

func withoutRootEventQueueItems(tx *gorm.DB) *gorm.DB {
	return tx.Where(`
		NOT EXISTS (
			SELECT 1
			FROM workflow_node_queue_items
			WHERE workflow_node_queue_items.root_event_id = workflow_events.id
		)
	`)
}

func withoutActiveRootEventExecutions(tx *gorm.DB) *gorm.DB {
	return tx.Where(`
		NOT EXISTS (
			SELECT 1
			FROM workflow_node_executions
			WHERE workflow_node_executions.root_event_id = workflow_events.id
			AND workflow_node_executions.state IN ?
		)
	`, []string{CanvasNodeExecutionStatePending, CanvasNodeExecutionStateStarted})
}

func withoutPendingRootEventRequests(tx *gorm.DB) *gorm.DB {
	return tx.Where(`
		NOT EXISTS (
			SELECT 1
			FROM workflow_node_requests
			JOIN workflow_node_executions ON workflow_node_requests.execution_id = workflow_node_executions.id
			WHERE workflow_node_executions.root_event_id = workflow_events.id
			AND workflow_node_requests.state = ?
		)
	`, NodeExecutionRequestStatePending)
}

func oldestCanvasEventsFirst(tx *gorm.DB) *gorm.DB {
	return tx.Order("workflow_events.created_at ASC")
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
			Strength: "UPDATE",
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

func DeleteRootCanvasEventChainsInTransaction(tx *gorm.DB, rootEventIDs []uuid.UUID) error {
	if len(rootEventIDs) == 0 {
		return nil
	}

	var executionIDs []uuid.UUID
	err := tx.
		Model(&CanvasNodeExecution{}).
		Where("root_event_id IN ?", rootEventIDs).
		Pluck("id", &executionIDs).
		Error
	if err != nil {
		return err
	}

	if len(executionIDs) > 0 {
		if err := tx.Where("execution_id IN ?", executionIDs).Delete(&CanvasNodeRequest{}).Error; err != nil {
			return err
		}

		if err := tx.Where("execution_id IN ?", executionIDs).Delete(&CanvasNodeExecutionKV{}).Error; err != nil {
			return err
		}

		if err := tx.Where("execution_id IN ?", executionIDs).Delete(&CanvasEvent{}).Error; err != nil {
			return err
		}

		if err := tx.Where("root_event_id IN ?", rootEventIDs).Delete(&CanvasNodeExecution{}).Error; err != nil {
			return err
		}
	}

	return tx.Where("id IN ?", rootEventIDs).Delete(&CanvasEvent{}).Error
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
