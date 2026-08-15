package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CanvasNodeExecutionConsumedEvent is a link row written every time a
// node execution consumes a queue item.
//
// This lives in the engine, not in components, so that we can answer
// "which event(s) fed this execution?" for any execution -- including
// ones that consume more than one event over their lifetime (e.g. a
// merge waiting on multiple sources) -- without any component-specific
// knowledge.
//
// EventID is nil once retention deletes the event: the row survives as a
// tombstone (ON DELETE SET NULL), so an erased input stays distinguishable
// from a source that never fed the execution.
type CanvasNodeExecutionConsumedEvent struct {
	ID          uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	ExecutionID uuid.UUID
	EventID     *uuid.UUID
	CreatedAt   *time.Time
}

func (e *CanvasNodeExecutionConsumedEvent) TableName() string {
	return "workflow_node_execution_consumed_events"
}

// A redelivery of an event the execution already consumed is a no-op: the link
// row exists, and a second one would replay as a duplicate input.
func CreateConsumedEvent(tx *gorm.DB, executionID, eventID uuid.UUID) error {
	now := time.Now()
	return tx.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "execution_id"}, {Name: "event_id"}},
			DoNothing: true,
		}).
		Create(&CanvasNodeExecutionConsumedEvent{
			ExecutionID: executionID,
			EventID:     &eventID,
			CreatedAt:   &now,
		}).Error
}

func ListConsumedEventsForExecution(tx *gorm.DB, executionID uuid.UUID) ([]CanvasNodeExecutionConsumedEvent, error) {
	var events []CanvasNodeExecutionConsumedEvent
	err := tx.
		Where("execution_id = ?", executionID).
		Order("created_at ASC").
		Find(&events).
		Error
	if err != nil {
		return nil, err
	}

	return events, nil
}
