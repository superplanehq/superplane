package schedule

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterTrigger("schedule", &Schedule{})
}

const (
	TypeMinutes = "minutes"
	TypeHourly  = "hourly"
	TypeDaily   = "daily"
	TypeWeekly  = "weekly"

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
	NextTrigger   *string `json:"nextTrigger"`
	ReferenceTime *string `json:"referenceTime"` // For minutes scheduling: time when schedule was first set up
}

type Configuration struct {
	Type     string  `json:"type"`
	Interval *int    `json:"interval"` // For minutes type: interval in minutes
	Minute   *int    `json:"minute"`   // 0-59
	Time     *string `json:"time"`     // Format: "HH:MM" UTC
	WeekDay  *string `json:"weekDay"`  // Monday, Tuesday, etc.
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

func (s *Schedule) Icon() string {
	return "alarm-clock"
}

func (s *Schedule) Color() string {
	return "yellow"
}

func (s *Schedule) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (s *Schedule) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "type",
			Label:    "Schedule Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  TypeDaily,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Every X minutes", Value: "minutes"},
						{Label: "Hourly", Value: "hourly"},
						{Label: "Daily", Value: "daily"},
						{Label: "Weekly", Value: "weekly"},
					},
				},
			},
		},
		{
			Name:    "interval",
			Label:   "Interval (minutes)",
			Type:    configuration.FieldTypeNumber,
			Default: intPtr(1),
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"minutes"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(60 * 24),
				},
			},
		},
		{
			Name:    "minute",
			Label:   "Minute of the hour",
			Type:    configuration.FieldTypeNumber,
			Default: intPtr(0),
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"hourly"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(0),
					Max: intPtr(59),
				},
			},
		},
		{
			Name:    "weekDay",
			Label:   "Day of the week",
			Type:    configuration.FieldTypeSelect,
			Default: WeekDayMonday,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"weekly"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Monday", Value: "Monday"},
						{Label: "Tuesday", Value: "Tuesday"},
						{Label: "Wednesday", Value: "Wednesday"},
						{Label: "Thursday", Value: "Thursday"},
						{Label: "Friday", Value: "Friday"},
						{Label: "Saturday", Value: "Saturday"},
						{Label: "Sunday", Value: "Sunday"},
					},
				},
			},
		},
		{
			Name:        "time",
			Label:       "Time",
			Type:        configuration.FieldTypeTime,
			Description: "Time of the day in UTC",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"daily", "weekly"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Time: &configuration.TimeTypeOptions{
					Format: "15:04",
				},
			},
		},
	}
}

func (s *Schedule) Setup(ctx core.TriggerContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var metadata Metadata
	err = mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	now := time.Now()

	if config.Type == TypeMinutes && metadata.ReferenceTime == nil {
		referenceTime := now.Format(time.RFC3339)
		metadata.ReferenceTime = &referenceTime
	}

	nextTrigger, err := getNextTrigger(config, now, metadata.ReferenceTime)
	if err != nil {
		return err
	}

	//
	// If the configuration didn't change, don't schedule a new action.
	//
	if metadata.NextTrigger != nil {
		currentTrigger, err := time.Parse(time.RFC3339, *metadata.NextTrigger)
		if err != nil {
			return fmt.Errorf("error parsing next trigger: %v", err)
		}

		if currentTrigger.Sub(*nextTrigger).Abs() < time.Second {
			return nil
		}
	}

	//
	// Always schedule the next and save the next trigger in the metadata.
	//
	err = ctx.RequestContext.ScheduleActionCall("emitEvent", map[string]any{}, time.Until(*nextTrigger))
	if err != nil {
		return err
	}

	formatted := nextTrigger.Format(time.RFC3339)
	ctx.MetadataContext.Set(Metadata{
		NextTrigger:   &formatted,
		ReferenceTime: metadata.ReferenceTime,
	})
	return nil
}

func (s *Schedule) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "emitEvent",
			UserAccessible: false,
		},
	}
}

func (s *Schedule) HandleAction(ctx core.TriggerActionContext) error {
	switch ctx.Name {
	case "emitEvent":
		return s.emitEvent(ctx)
	}

	return fmt.Errorf("action %s not supported", ctx.Name)
}

func (s *Schedule) emitEvent(ctx core.TriggerActionContext) error {
	err := ctx.EventContext.Emit(map[string]any{})
	if err != nil {
		return err
	}

	spec := Configuration{}
	err = mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	var existingMetadata Metadata
	err = mapstructure.Decode(ctx.MetadataContext.Get(), &existingMetadata)
	if err != nil {
		return fmt.Errorf("failed to parse existing metadata: %w", err)
	}

	now := time.Now()
	nextTrigger, err := getNextTrigger(spec, now, existingMetadata.ReferenceTime)
	if err != nil {
		return err
	}

	err = ctx.RequestContext.ScheduleActionCall("emitEvent", map[string]any{}, time.Until(*nextTrigger))
	if err != nil {
		return err
	}

	formatted := nextTrigger.Format(time.RFC3339)
	ctx.MetadataContext.Set(Metadata{
		NextTrigger:   &formatted,
		ReferenceTime: existingMetadata.ReferenceTime,
	})
	return nil
}

func getNextTrigger(config Configuration, now time.Time, referenceTime *string) (*time.Time, error) {
	switch config.Type {
	case TypeMinutes:
		if config.Interval == nil {
			return nil, fmt.Errorf("interval is required for minutes schedule")
		}

		return nextMinutesTrigger(*config.Interval, now, referenceTime)

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

func nextMinutesTrigger(interval int, now time.Time, referenceTime *string) (*time.Time, error) {
	if interval < 1 || interval > 1440 {
		return nil, fmt.Errorf("interval must be between 1 and 1440 minutes, got: %d", interval)
	}

	nowUTC := now.UTC()

	var reference time.Time
	if referenceTime != nil {
		var err error
		reference, err = time.Parse(time.RFC3339, *referenceTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse reference time: %w", err)
		}
		reference = reference.UTC()
	} else {
		reference = nowUTC
	}

	minutesElapsed := int(nowUTC.Sub(reference).Minutes())

	if minutesElapsed < 0 {
		minutesElapsed = 0
	}
	completedIntervals := minutesElapsed / interval

	nextTriggerMinutes := (completedIntervals + 1) * interval
	nextTrigger := reference.Add(time.Duration(nextTriggerMinutes) * time.Minute)

	if nextTrigger.Before(nowUTC) || nextTrigger.Equal(nowUTC) {
		nextTrigger = nextTrigger.Add(time.Duration(interval) * time.Minute)
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
