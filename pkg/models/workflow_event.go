package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	WorkflowEventStateRouting    = "routing"
	WorkflowEventStateProcessing = "processing"
	WorkflowEventStateCompleted  = "completed"
	WorkflowEventStateFailed     = "failed"
)

type WorkflowEvent struct {
	ID            uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	WorkflowID    uuid.UUID
	ParentEventID *uuid.UUID
	BlueprintName *string
	Data          datatypes.JSONType[map[string]any]
	State         string
	CreatedAt     *time.Time
	UpdatedAt     *time.Time
}

func (w *WorkflowEvent) Route() error {
	now := time.Now()
	w.State = WorkflowEventStateRouting
	w.UpdatedAt = &now
	return database.Conn().Save(w).Error
}

func (w *WorkflowEvent) Complete() error {
	now := time.Now()
	w.State = WorkflowEventStateCompleted
	w.UpdatedAt = &now
	return database.Conn().Save(w).Error
}

func (w *WorkflowEvent) Fail() error {
	now := time.Now()
	w.State = WorkflowEventStateFailed
	w.UpdatedAt = &now
	return database.Conn().Save(w).Error
}

func (w *WorkflowEvent) Processing() error {
	return w.ProcessingInTransaction(database.Conn())
}

func (w *WorkflowEvent) ProcessingInTransaction(tx *gorm.DB) error {
	now := time.Now()
	w.State = WorkflowEventStateProcessing
	w.UpdatedAt = &now
	return tx.Save(w).Error
}

func FindWorkflowEvent(id string) (*WorkflowEvent, error) {
	var event WorkflowEvent
	if err := database.Conn().First(&event, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func ListEventsToRoute() ([]WorkflowEvent, error) {
	var events []WorkflowEvent
	err := database.Conn().
		Where("state = ?", WorkflowEventStateRouting).
		Find(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}
