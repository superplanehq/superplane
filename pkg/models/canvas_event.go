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

func FindCanvasEventForCanvas(db *gorm.DB, canvasID uuid.UUID, id uuid.UUID) (*CanvasEvent, error) {
	var event CanvasEvent
	err := db.
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

// ListRootEventsForRuns returns the event that started each run, keyed by run
// id. A root event is the one with no execution behind it.
//
// A multi-input replay leaves several such events on the same run, and its
// queue items and executions are rooted on exactly one of them. Prefer that
// event so ListEventExecutions, which matches on root_event_id, can find the
// run's work.
func ListRootEventsForRuns(tx *gorm.DB, canvasID uuid.UUID, runIDs []uuid.UUID) (map[uuid.UUID]CanvasEvent, error) {
	eventsByRunID := make(map[uuid.UUID]CanvasEvent, len(runIDs))
	if len(runIDs) == 0 {
		return eventsByRunID, nil
	}

	var events []CanvasEvent
	err := tx.
		Where("workflow_id = ?", canvasID).
		Where("run_id IN ?", runIDs).
		Where("execution_id IS NULL").
		Find(&events).
		Error
	if err != nil {
		return nil, err
	}

	for _, event := range events {
		if _, exists := eventsByRunID[event.RunID]; exists {
			return listRootEventsPreferringRooted(tx, canvasID, runIDs, events)
		}

		eventsByRunID[event.RunID] = event
	}

	return eventsByRunID, nil
}

func listRootEventsPreferringRooted(
	tx *gorm.DB,
	canvasID uuid.UUID,
	runIDs []uuid.UUID,
	events []CanvasEvent,
) (map[uuid.UUID]CanvasEvent, error) {
	rootedOn, err := rootEventIDsForRuns(tx, canvasID, runIDs)
	if err != nil {
		return nil, err
	}

	eventsByRunID := make(map[uuid.UUID]CanvasEvent, len(runIDs))
	for _, event := range events {
		_, taken := eventsByRunID[event.RunID]
		if taken && !rootedOn[event.ID] {
			continue
		}

		eventsByRunID[event.RunID] = event
	}

	return eventsByRunID, nil
}

func rootEventIDsForRuns(tx *gorm.DB, canvasID uuid.UUID, runIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	var ids []uuid.UUID
	err := tx.
		Model(&CanvasNodeExecution{}).
		Where("workflow_id = ?", canvasID).
		Where("run_id IN ?", runIDs).
		Distinct().
		Pluck("root_event_id", &ids).
		Error
	if err != nil {
		return nil, err
	}

	var queueItemIDs []uuid.UUID
	err = tx.
		Model(&CanvasNodeQueueItem{}).
		Where("workflow_id = ?", canvasID).
		Where("run_id IN ?", runIDs).
		Distinct().
		Pluck("root_event_id", &queueItemIDs).
		Error
	if err != nil {
		return nil, err
	}

	rootedOn := make(map[uuid.UUID]bool, len(ids)+len(queueItemIDs))
	for _, id := range append(ids, queueItemIDs...) {
		rootedOn[id] = true
	}

	return rootedOn, nil
}

func ListCanvasEvents(db *gorm.DB, canvasID uuid.UUID, nodeID string, limit int, before *time.Time) ([]CanvasEvent, error) {
	var events []CanvasEvent
	query := db.
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

func CountCanvasEvents(db *gorm.DB, canvasID uuid.UUID, nodeID string) (int64, error) {
	var count int64

	err := db.
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
