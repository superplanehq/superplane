package models

import (
	"context"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

const (
	EventSourceStatePending  = "pending"
	EventSourceStateReady    = "ready"
	EventSourceScopeExternal = "external"
	EventSourceScopeInternal = "internal"
)

type EventSource struct {
	ID          uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	CanvasID    uuid.UUID
	ResourceID  *uuid.UUID
	Name        string
	Description string
	Key         []byte
	State       string
	Scope       string
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

func (s *EventSource) UpdateKey(key []byte) error {
	now := time.Now()
	s.Key = key
	s.UpdatedAt = &now
	return database.Conn().Save(s).Error
}

func (s *EventSource) GetDecryptedKey(ctx context.Context, encryptor crypto.Encryptor) ([]byte, error) {
	return s.GetDecryptedKeyInTransaction(ctx, database.Conn(), encryptor)
}

func (s *EventSource) GetDecryptedKeyInTransaction(ctx context.Context, tx *gorm.DB, encryptor crypto.Encryptor) ([]byte, error) {
	if s.ResourceID == nil {
		return encryptor.Decrypt(ctx, s.Key, []byte(s.Name))
	}

	resource, err := FindResourceByIDInTransaction(tx, *s.ResourceID)
	if err != nil {
		return nil, err
	}

	return encryptor.Decrypt(ctx, s.Key, []byte(resource.Id()))
}

func (s *EventSource) UpdateState(state string) error {
	return s.UpdateStateInTransaction(database.Conn(), state)
}

func (s *EventSource) UpdateStateInTransaction(tx *gorm.DB, state string) error {
	s.State = state
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

func FindEventSourceByName(name string) (*EventSource, error) {
	return FindEventSourceByNameInTransaction(database.Conn(), name)
}

func FindEventSourceByNameInTransaction(tx *gorm.DB, name string) (*EventSource, error) {
	var eventSource EventSource
	err := tx.
		Where("name = ?", name).
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
		Where("scope = ?", EventSourceScopeExternal).
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
