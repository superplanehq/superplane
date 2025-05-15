package models

import (
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
)

type EventSource struct {
	ID        uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	CanvasID  uuid.UUID
	Name      string
	Key       []byte
	CreatedAt *time.Time
	UpdatedAt *time.Time
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
