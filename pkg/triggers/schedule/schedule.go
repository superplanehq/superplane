package schedule

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/triggers"
)

const (
	TypeHourly = "hourly"
	TypeDaily  = "daily"
	TypeWeekly = "weekly"

	WeekDayMonday    = "monday"
	WeekDayTuesday   = "tuesday"
	WeekDayWednesday = "wednesday"
	WeekDayThursday  = "thursday"
	WeekDayFriday    = "friday"
	WeekDaySaturday  = "saturday"
	WeekDaySunday    = "sunday"
)

type Schedule struct{}

type Metadata struct {
	NextTrigger *time.Time `mapstructure:"next_trigger" json:"next_trigger"`
}

type Configuration struct {
	Type    string  `json:"type"`
	Minute  *int    `json:"minute"`   // 0-59
	Time    *string `json:"time"`     // Format: "HH:MM" UTC
	WeekDay *string `json:"week_day"` // Monday, Tuesday, etc.
}

func (s *Schedule) Name() string {
	return "schedule"
}

func (s *Schedule) Label() string {
	return "Schedule"
}

func (s *Schedule) Description() string {
	return "Start a new execution chain on a schedule"
}

func (s *Schedule) OutputChannels() []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (s *Schedule) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:  "type",
			Label: "Schedule Type",
			Type:  components.FieldTypeSelect,
			Options: []components.FieldOption{
				{Label: "Hourly", Value: "hourly"},
				{Label: "Daily", Value: "daily"},
				{Label: "Weekly", Value: "weekly"},
			},
			Required: true,
		},
		{
			Name:  "minute",
			Label: "Minute of the hour",
			Type:  components.FieldTypeNumber,
			Min:   intPtr(0),
			Max:   intPtr(59),
			VisibilityConditions: []components.VisibilityCondition{
				{Field: "type", Values: []string{"hourly"}},
			},
		},
		{
			Name:        "time",
			Label:       "Time",
			Type:        components.FieldTypeString,
			Description: "Time of the day, in HH:MM format",
			VisibilityConditions: []components.VisibilityCondition{
				{Field: "type", Values: []string{"daily", "weekly"}},
			},
		},
		{
			Name:  "week_day",
			Label: "Day of the week",
			Type:  components.FieldTypeSelect,
			VisibilityConditions: []components.VisibilityCondition{
				{Field: "type", Values: []string{"weekly"}},
			},
			Options: []components.FieldOption{
				{Label: "Monday", Value: "Monday"},
				{Label: "Tuesday", Value: "Tuesday"},
				{Label: "Wednesday", Value: "Wednesday"},
				{Label: "Thursday", Value: "Thursday"},
				{Label: "Friday", Value: "Friday"},
				{Label: "Saturday", Value: "Saturday"},
				{Label: "Sunday", Value: "Sunday"},
			},
		},
	}
}

func (s *Schedule) Start(ctx triggers.TriggerContext) error {
	log.Printf("Configuration before: %v", ctx.Configuration)

	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	log.Printf("Configuration after: %v", config)

	var metadata Metadata
	err = mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	//
	// If nextTrigger timestamp is before the current time, emit an event.
	//
	if metadata.NextTrigger != nil && metadata.NextTrigger.Before(time.Now()) {
		err = ctx.EventContext.Emit(map[string]any{})
		if err != nil {
			return err
		}
	}

	nextTrigger, err := getNextTrigger(config, time.Now())
	if err != nil {
		return err
	}

	//
	// Always schedule the next and save the next trigger in the metadata.
	//
	err = ctx.RequestContext.ScheduleActionCall("emitEvent", map[string]any{}, time.Until(*nextTrigger))
	if err != nil {
		return err
	}

	metadata.NextTrigger = nextTrigger
	ctx.MetadataContext.Set(metadata)
	return nil
}

func (s *Schedule) Actions() []components.Action {
	return []components.Action{
		{
			Name:           "emitEvent",
			UserAccessible: false,
		},
	}
}

func (s *Schedule) HandleAction(ctx triggers.TriggerActionContext) error {
	switch ctx.Name {
	case "emitEvent":
		return s.emitEvent(ctx)
	}

	return fmt.Errorf("action %s not supported", ctx.Name)
}

func (s *Schedule) emitEvent(ctx triggers.TriggerActionContext) error {
	err := ctx.EventContext.Emit(map[string]any{})
	if err != nil {
		return err
	}

	spec := Configuration{}
	err = mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	now := time.Now()
	nextTrigger, err := getNextTrigger(spec, now)
	if err != nil {
		return err
	}

	err = ctx.RequestContext.ScheduleActionCall("emitEvent", map[string]any{}, time.Until(*nextTrigger))
	if err != nil {
		return err
	}

	var metadata Metadata
	err = mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	metadata.NextTrigger = nextTrigger
	ctx.MetadataContext.Set(metadata)
	return nil
}

func getNextTrigger(config Configuration, now time.Time) (*time.Time, error) {
	log.Printf("Configuration: %v", config)

	switch config.Type {
	case TypeHourly:
		if config.Minute == nil {
			return nil, fmt.Errorf("minute is required for hourly schedule")
		}

		return nextHourlyTrigger(*config.Minute, now)

	case TypeDaily:
		if config.Time == nil {
			return nil, fmt.Errorf("time is required for daily schedule")
		}

		return nextDailyTrigger(*config.Time, now)

	case TypeWeekly:
		if config.Time == nil {
			return nil, fmt.Errorf("time is required for weekly schedule")
		}

		if config.WeekDay == nil {
			return nil, fmt.Errorf("week_day is required for weekly schedule")
		}

		return nextWeeklyTrigger(*config.WeekDay, *config.Time, now)

	default:
		return nil, fmt.Errorf("unsupported schedule type: %s", config.Type)
	}
}

func nextHourlyTrigger(minute int, now time.Time) (*time.Time, error) {
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

func nextDailyTrigger(timeValue string, now time.Time) (*time.Time, error) {
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

func nextWeeklyTrigger(weekDay string, timeValue string, now time.Time) (*time.Time, error) {
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
	nextTrigger := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), hour, minute, 0, 0, time.UTC)

	//
	// If target and current week days are the same,
	// we need to check if the next trigger is in the past
	//
	if targetWeekday == currentWeekday {
		if nextTrigger.Before(nowUTC) || nextTrigger.Equal(nowUTC) {
			nextTrigger = nextTrigger.AddDate(0, 0, 7)
		}
		return &nextTrigger, nil
	}

	//
	// Otherwise, we need to calculate the number of days until the target week day
	//
	daysUntilTarget := int(targetWeekday - currentWeekday)
	if daysUntilTarget < 0 {
		daysUntilTarget += 7
	}

	nextTrigger = nextTrigger.AddDate(0, 0, daysUntilTarget)
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

func intPtr(v int) *int {
	return &v
}
