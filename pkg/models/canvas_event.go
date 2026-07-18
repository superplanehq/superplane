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

	run, err := CreateCanvasRunInTransaction(tx, e.WorkflowID, e.NodeID, CanvasRunStateStarted, "")
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

func ListRootCanvasEvents(canvasID uuid.UUID, limit int, before *time.Time) ([]CanvasEvent, error) {
	return ListRootCanvasEventsInTransaction(database.Conn(), canvasID, limit, before)
}

func ListRootCanvasEventsInTransaction(tx *gorm.DB, canvasID uuid.UUID, limit int, before *time.Time) ([]CanvasEvent, error) {
	var events []CanvasEvent
	query := tx.
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
	return CountRootCanvasEventsInTransaction(database.Conn(), canvasID)
}

func CountRootCanvasEventsInTransaction(tx *gorm.DB, canvasID uuid.UUID) (int64, error) {
	var count int64

	err := tx.
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
	return tx.
		Table("workflow_events").
		Select("workflow_events.*").
		Joins("JOIN workflows ON workflow_events.workflow_id = workflows.id").
		Joins("JOIN organizations ON workflows.organization_id = organizations.id").
		Where("organizations.usage_retention_window_days IS NOT NULL").
		Where("organizations.usage_retention_window_days > 0").
		Where("workflow_events.execution_id IS NULL").
		Where("workflow_events.state = ?", CanvasEventStateRouted).
		Where("workflow_events.created_at + (organizations.usage_retention_window_days * INTERVAL '1 day') < ?", referenceTime.UTC())
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
	`, []string{CanvasNodeExecutionStatePending, CanvasNodeExecutionStateStarted, CanvasNodeExecutionStateCancelling})
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

	var runIDs []uuid.UUID
	err = tx.
		Model(&CanvasEvent{}).
		Where("id IN ?", rootEventIDs).
		Distinct("run_id").
		Pluck("run_id", &runIDs).
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

	if err := tx.Where("root_event_id IN ?", rootEventIDs).Delete(&CanvasNodeQueueItem{}).Error; err != nil {
		return err
	}

	if err := tx.Where("id IN ?", rootEventIDs).Delete(&CanvasEvent{}).Error; err != nil {
		return err
	}

	if len(runIDs) > 0 {
		return tx.Where("id IN ?", runIDs).Delete(&CanvasRun{}).Error
	}

	return nil
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
