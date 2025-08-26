package models

import (
	"fmt"
	"slices"
	"strings"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	EventTypes datatypes.JSONSlice[EventType]
}

type EventType struct {
	Type           string   `json:"type"`
	FilterOperator string   `json:"filter_operator"`
	Filters        []Filter `json:"filters"`
}

// NOTE: caller must encrypt the key before calling this method.
func (s *EventSource) Create() error {
	return s.CreateInTransaction(database.Conn())
}

func (s *EventSource) CreateInTransaction(tx *gorm.DB) error {
	now := time.Now()

	s.CreatedAt = &now
	s.UpdatedAt = &now
	s.State = EventSourceStatePending

	err := tx.
		Clauses(clause.Returning{}).
		Create(&s).
		Error

	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return ErrNameAlreadyUsed
	}

	return err
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
	s.State = state
	return tx.Save(s).Error
}

func (s *EventSource) FindIntegration() (*Integration, error) {
	var integration Integration

	err := database.Conn().
		Table("resources").
		Joins("INNER JOIN integrations ON integrations.id = resources.integration_id").
		Where("resources.id = ?", s.ResourceID).
		Select("integrations.*").
		First(&integration).
		Error

	if err != nil {
		return nil, err
	}

	return &integration, nil
}

func (s *EventSource) Accept(event *Event) (bool, error) {
	//
	// If no event types are defined, accept all events.
	//
	if len(s.EventTypes) == 0 {
		return true, nil
	}

	//
	// Check if the event type is accepted before applying filters.
	//
	i := slices.IndexFunc(s.EventTypes, func(eventType EventType) bool {
		return eventType.Type == event.Type
	})

	if i == -1 {
		return false, nil
	}

	//
	// Apply the filters for the event type.
	//
	eventType := s.EventTypes[i]
	return ApplyFilters(eventType.Filters, eventType.FilterOperator, event)
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

func FindEventSourceByName(canvasID string, name string) (*EventSource, error) {
	var eventSource EventSource
	err := database.Conn().
		Where("canvas_id = ?", canvasID).
		Where("name = ?", name).
		First(&eventSource).
		Error

	if err != nil {
		return nil, err
	}

	return &eventSource, nil
}

func FindExternalEventSourceByID(canvasID string, id string) (*EventSource, error) {
	var eventSource EventSource
	err := database.Conn().
		Where("id = ?", id).
		Where("canvas_id = ?", canvasID).
		Where("scope = ?", EventSourceScopeExternal).
		First(&eventSource).
		Error

	if err != nil {
		return nil, err
	}

	return &eventSource, nil
}

func FindExternalEventSourceByName(canvasID string, name string) (*EventSource, error) {
	var eventSource EventSource
	err := database.Conn().
		Where("canvas_id = ?", canvasID).
		Where("name = ?", name).
		Where("scope = ?", EventSourceScopeExternal).
		First(&eventSource).
		Error

	if err != nil {
		return nil, err
	}

	return &eventSource, nil
}

func ListUnscopedSoftDeletedEventSources(limit int) ([]EventSource, error) {
	var sources []EventSource

	err := database.Conn().
		Unscoped().
		Where("deleted_at is not null").
		Limit(limit).
		Find(&sources).
		Error

	if err != nil {
		return nil, err
	}

	return sources, nil
}

func FindInternalEventSourceByName(canvasID string, name string) (*EventSource, error) {
	var eventSource EventSource
	err := database.Conn().
		Where("canvas_id = ?", canvasID).
		Where("name = ?", name).
		Where("scope = ?", EventSourceScopeInternal).
		First(&eventSource).
		Error

	if err != nil {
		return nil, err
	}

	return &eventSource, nil
}

func ListEventSources(canvasID string) ([]EventSource, error) {
	var sources []EventSource
	err := database.Conn().
		Where("canvas_id = ?", canvasID).
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

func (s *EventSource) Delete() error {
	deletedName := fmt.Sprintf("%s-deleted-%d", s.Name, time.Now().Unix())

	return database.Conn().Model(s).
		Where("id = ?", s.ID).
		Update("name", deletedName).
		Update("deleted_at", time.Now()).
		Error
}

func (s *EventSource) HardDeleteInTransaction(tx *gorm.DB) error {
	return tx.Unscoped().Delete(s).Error
}
