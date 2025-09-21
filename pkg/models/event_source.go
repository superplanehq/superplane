package models

import (
	"fmt"
	"slices"
	"strconv"
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

	ScheduleTypeHourly = "hourly"
	ScheduleTypeDaily  = "daily"
	ScheduleTypeWeekly = "weekly"

	WeekDayMonday    = "monday"
	WeekDayTuesday   = "tuesday"
	WeekDayWednesday = "wednesday"
	WeekDayThursday  = "thursday"
	WeekDayFriday    = "friday"
	WeekDaySaturday  = "saturday"
	WeekDaySunday    = "sunday"
)

type EventSource struct {
	ID              uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	CanvasID        uuid.UUID
	ResourceID      *uuid.UUID
	Name            string
	Description     string
	Key             []byte
	State           string
	Scope           string
	Schedule        *datatypes.JSONType[Schedule] `gorm:"column:schedule"`
	LastTriggeredAt *time.Time                    `gorm:"column:last_triggered_at"`
	NextTriggerAt   *time.Time                    `gorm:"column:next_trigger_at"`
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	EventTypes datatypes.JSONSlice[EventType]
}

type EventType struct {
	Type           string   `json:"type"`
	FilterOperator string   `json:"filter_operator"`
	Filters        []Filter `json:"filters"`
}

type Schedule struct {
	Type   string          `json:"type"`
	Hourly *HourlySchedule `json:"hourly,omitempty"`
	Daily  *DailySchedule  `json:"daily,omitempty"`
	Weekly *WeeklySchedule `json:"weekly,omitempty"`
}

func (s *Schedule) CalculateNextTrigger(now time.Time) (*time.Time, error) {
	switch s.Type {
	case ScheduleTypeHourly:
		if s.Hourly == nil {
			return nil, fmt.Errorf("hourly schedule configuration is missing")
		}
		return s.calculateNextHourlyTrigger(s.Hourly.Minute, now)
	case ScheduleTypeDaily:
		if s.Daily == nil {
			return nil, fmt.Errorf("daily schedule configuration is missing")
		}
		return s.calculateNextDailyTrigger(s.Daily.Time, now)
	case ScheduleTypeWeekly:
		if s.Weekly == nil {
			return nil, fmt.Errorf("weekly schedule configuration is missing")
		}
		return s.calculateNextWeeklyTrigger(s.Weekly.WeekDay, s.Weekly.Time, now)
	default:
		return nil, fmt.Errorf("unsupported schedule type: %s", s.Type)
	}
}

func (s *Schedule) calculateNextHourlyTrigger(minute int, now time.Time) (*time.Time, error) {
	if minute < 0 || minute > 59 {
		return nil, fmt.Errorf("minute must be between 0 and 59, got: %d", minute)
	}

	nowUTC := now.UTC()
	nextTrigger := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), nowUTC.Hour(), minute, 0, 0, time.UTC)

	if nextTrigger.Before(nowUTC) || nextTrigger.Equal(nowUTC) {
		nextTrigger = nextTrigger.Add(time.Hour)
	}

	return &nextTrigger, nil
}

func (s *Schedule) calculateNextDailyTrigger(timeValue string, now time.Time) (*time.Time, error) {
	hour, minute, err := parseTime(timeValue)
	if err != nil {
		return nil, fmt.Errorf("invalid time format: %v", err)
	}

	nowUTC := now.UTC()
	nextTrigger := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), hour, minute, 0, 0, time.UTC)

	if nextTrigger.Before(nowUTC) || nextTrigger.Equal(nowUTC) {
		nextTrigger = nextTrigger.AddDate(0, 0, 1)
	}

	return &nextTrigger, nil
}

func (s *Schedule) calculateNextWeeklyTrigger(weekDay string, timeValue string, now time.Time) (*time.Time, error) {
	hour, minute, err := parseTime(timeValue)
	if err != nil {
		return nil, fmt.Errorf("invalid time format: %v", err)
	}

	targetWeekday, err := parseWeekday(weekDay)
	if err != nil {
		return nil, fmt.Errorf("invalid weekday: %v", err)
	}

	nowUTC := now.UTC()
	currentWeekday := nowUTC.Weekday()

	daysUntilTarget := int(targetWeekday - currentWeekday)
	if daysUntilTarget < 0 {
		daysUntilTarget += 7
	}

	nextTrigger := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), hour, minute, 0, 0, time.UTC)
	nextTrigger = nextTrigger.AddDate(0, 0, daysUntilTarget)

	if daysUntilTarget == 0 && (nextTrigger.Before(nowUTC) || nextTrigger.Equal(nowUTC)) {
		nextTrigger = nextTrigger.AddDate(0, 0, 7)
	}

	return &nextTrigger, nil
}


func parseTime(timeValue string) (hour int, minute int, err error) {
	parts := strings.Split(timeValue, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("time must be in HH:MM format, got: %s", timeValue)
	}

	hour, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid hour: %s", parts[0])
	}

	minute, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minute: %s", parts[1])
	}

	if hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("hour must be between 0 and 23, got: %d", hour)
	}

	if minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("minute must be between 0 and 59, got: %d", minute)
	}

	return hour, minute, nil
}

func parseWeekday(weekDay string) (time.Weekday, error) {
	switch strings.ToLower(weekDay) {
	case WeekDayMonday:
		return time.Monday, nil
	case WeekDayTuesday:
		return time.Tuesday, nil
	case WeekDayWednesday:
		return time.Wednesday, nil
	case WeekDayThursday:
		return time.Thursday, nil
	case WeekDayFriday:
		return time.Friday, nil
	case WeekDaySaturday:
		return time.Saturday, nil
	case WeekDaySunday:
		return time.Sunday, nil
	default:
		return time.Sunday, fmt.Errorf("invalid weekday: %s", weekDay)
	}
}

func WeekdayToString(weekday time.Weekday) string {
	switch weekday {
	case time.Monday:
		return WeekDayMonday
	case time.Tuesday:
		return WeekDayTuesday
	case time.Wednesday:
		return WeekDayWednesday
	case time.Thursday:
		return WeekDayThursday
	case time.Friday:
		return WeekDayFriday
	case time.Saturday:
		return WeekDaySaturday
	case time.Sunday:
		return WeekDaySunday
	default:
		return WeekDayMonday
	}
}

type HourlySchedule struct {
	Minute int `json:"minute"` // 0-59, minute of the hour to trigger
}

type DailySchedule struct {
	Time string `json:"time"` // Format: "HH:MM" in UTC (24-hour format)
}

type WeeklySchedule struct {
	WeekDay string `json:"week_day"` // Monday, Tuesday, etc.
	Time    string `json:"time"`     // Format: "HH:MM" in UTC (24-hour format)
}

func (s *EventSource) UpdateNextTrigger(nextTrigger time.Time) error {
	return s.UpdateNextTriggerInTransaction(database.Conn(), nextTrigger)
}

func (s *EventSource) UpdateNextTriggerInTransaction(tx *gorm.DB, nextTrigger time.Time) error {
	now := time.Now()
	s.NextTriggerAt = &nextTrigger
	s.LastTriggeredAt = &now
	s.UpdatedAt = &now

	return tx.Save(s).Error
}

func ListDueScheduledEventSources() ([]EventSource, error) {
	var eventSources []EventSource
	now := time.Now()

	err := database.Conn().
		Where("next_trigger_at <= ?", now).
		Find(&eventSources).
		Error

	if err != nil {
		return nil, err
	}

	return eventSources, nil
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
