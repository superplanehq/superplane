package models

import (
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

const (
	EventSourceStatePending = "pending"
	EventSourceStateReady   = "ready"
)

type EventSource struct {
	ID         uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	CanvasID   uuid.UUID
	ResourceID *uuid.UUID
	Name       string
	Key        []byte
	State      string
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
}

func (s *EventSource) UpdateKey(key []byte) error {
	now := time.Now()
	s.Key = key
	s.UpdatedAt = &now
	return database.Conn().Save(s).Error
}

func (s *EventSource) UpdateState(state string) error {
	return s.UpdateStateInTransaction(database.Conn(), state)
}

func (s *EventSource) UpdateStateInTransaction(tx *gorm.DB, state string) error {
	now := time.Now()
	s.State = state
	s.UpdatedAt = &now
	return tx.Save(s).Error
}

func FindEventSource(id uuid.UUID) (*EventSource, error) {
	var eventSource EventSource
	err := database.Conn().
		Where("id = ?", id).
		First(&eventSource).
		Error

	if err != nil {
		return nil, err
	}

	return &eventSource, nil
}

func FindEventSourceByResourceID(resourceID uuid.UUID) (*EventSource, error) {
	var eventSource EventSource
	err := database.Conn().
		Where("resource_id = ?", resourceID).
		First(&eventSource).
		Error

	if err != nil {
		return nil, err
	}

	return &eventSource, nil
}

func (c *Canvas) ListEventSources() ([]EventSource, error) {
	var sources []EventSource
	err := database.Conn().
		Where("canvas_id = ?", c.ID).
		Find(&sources).
		Error

	if err != nil {
		return nil, err
	}

	return sources, nil
}

func ListPendingEventSources() ([]EventSource, error) {
	eventSources := []EventSource{}

	err := database.Conn().
		Where("state = ?", EventSourceStatePending).
		Find(&eventSources).
		Error

	if err != nil {
		return nil, err
	}

	return eventSources, nil
}

func FindExecutorFromSource(sourceID uuid.UUID) (*StageExecutor, error) {
	var executor StageExecutor

	err := database.Conn().
		Table("event_sources").
		Select("stage_executors.*").
		Joins("INNER JOIN resources ON resources.id = event_sources.resource_id").
		Joins("INNER JOIN stage_executors ON stage_executors.resource_id = resources.id").
		Where("event_sources.id = ?", sourceID).
		First(&executor).
		Error

	if err != nil {
		return nil, err
	}

	return &executor, nil
}
