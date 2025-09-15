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

type EventSourceStatusInfo struct {
	EventSourceID uuid.UUID
	ReceivedCount int
	RecentEvents  []Event
}

func GetEventSourcesStatusInfo(eventSources []EventSource) (map[uuid.UUID]*EventSourceStatusInfo, error) {
	statusMap := make(map[uuid.UUID]*EventSourceStatusInfo)

	if len(eventSources) == 0 {
		return statusMap, nil
	}

	eventSourceIDs := make([]uuid.UUID, len(eventSources))
	for i, source := range eventSources {
		eventSourceIDs[i] = source.ID
		statusMap[source.ID] = &EventSourceStatusInfo{
			EventSourceID: source.ID,
			RecentEvents:  []Event{},
		}
	}

	// Get event counts for each event source
	counts, err := getEventCountsForEventSources(eventSourceIDs)
	if err != nil {
		return nil, err
	}

	// Get recent events for each event source (last 3)
	recentEvents, err := getRecentEventsForEventSources(eventSourceIDs)
	if err != nil {
		return nil, err
	}

	// Populate status map
	for sourceID, count := range counts {
		if status, exists := statusMap[sourceID]; exists {
			status.ReceivedCount = count
		}
	}

	for sourceID, events := range recentEvents {
		if status, exists := statusMap[sourceID]; exists {
			status.RecentEvents = events
		}
	}

	return statusMap, nil
}

func getEventCountsForEventSources(eventSourceIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	counts := make(map[uuid.UUID]int)

	var results []struct {
		SourceID uuid.UUID
		Count    int
	}

	err := database.Conn().
		Raw(`
			SELECT source_id, COUNT(*) as count
			FROM events 
			WHERE source_id IN ? AND source_type = ?
			GROUP BY source_id
		`, eventSourceIDs, SourceTypeEventSource).
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	for _, result := range results {
		counts[result.SourceID] = result.Count
	}

	return counts, nil
}

func getRecentEventsForEventSources(eventSourceIDs []uuid.UUID) (map[uuid.UUID][]Event, error) {
	events := make(map[uuid.UUID][]Event)

	var results []Event

	err := database.Conn().
		Raw(`
			SELECT * FROM (
				SELECT *, ROW_NUMBER() OVER (PARTITION BY source_id ORDER BY received_at DESC) as rn
				FROM events 
				WHERE source_id IN ? AND source_type = ?
			) ranked
			WHERE rn <= 3
			ORDER BY source_id, received_at DESC
		`, eventSourceIDs, SourceTypeEventSource).
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	for _, event := range results {
		if _, exists := events[event.SourceID]; !exists {
			events[event.SourceID] = []Event{}
		}
		events[event.SourceID] = append(events[event.SourceID], event)
	}

	return events, nil
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

func (s *EventSource) DeleteStageEventsInTransaction(tx *gorm.DB) error {
	// Delete events associated with stage events from this event source
	if err := tx.Unscoped().
		Where("id IN (SELECT event_id FROM stage_events WHERE source_id = ?)", s.ID).
		Delete(&Event{}).Error; err != nil {
		return fmt.Errorf("failed to delete events: %v", err)
	}

	if err := tx.Unscoped().Where("source_id = ?", s.ID).Delete(&StageEvent{}).Error; err != nil {
		return fmt.Errorf("failed to delete stage events: %v", err)
	}

	return nil
}

func (s *EventSource) DeleteConnectionsInTransaction(tx *gorm.DB) error {
	if err := tx.Unscoped().Where("source_id = ? AND source_type = ?", s.ID, SourceTypeEventSource).Delete(&Connection{}).Error; err != nil {
		return fmt.Errorf("failed to delete connections: %v", err)
	}
	return nil
}

func (s *EventSource) DeleteEventsInTransaction(tx *gorm.DB) error {
	if err := tx.Unscoped().Where("source_id = ? AND source_type = ?", s.ID, SourceTypeEventSource).Delete(&Event{}).Error; err != nil {
		return fmt.Errorf("failed to delete events: %v", err)
	}
	return nil
}
