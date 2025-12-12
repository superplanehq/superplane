package schedule

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/robfig/cron/v3"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/triggers"
)

func init() {
	registry.RegisterTrigger("schedule", &Schedule{})
}

const (
	TypeMinutes = "minutes"
	TypeHours   = "hours"
	TypeDays    = "days"
	TypeWeeks   = "weeks"
	TypeMonths  = "months"
	TypeCron    = "cron"

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
	Type           string   `json:"type"`
	Interval       *int     `json:"interval"`       // Minutes (1-59), hours (1-23), days (1-31), weeks (1-52), months (1-24)
	Minute         *int     `json:"minute"`         // 0-59 for hours, days, weeks, months
	Hour           *int     `json:"hour"`           // 0-23 for days, weeks, months
	WeekDays       []string `json:"weekDays"`       // For weeks scheduling (multiple days)
	DayOfMonth     *int     `json:"dayOfMonth"`     // 1-31 for months scheduling
	CronExpression *string  `json:"cronExpression"` // For cron scheduling
	Timezone       *string  `json:"timezone"`       // Timezone offset (e.g., "0", "-5", "5.5")
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

func (s *Schedule) HandleWebhook(ctx triggers.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (s *Schedule) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "timezone",
			Label:       "Timezone",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "0",
			Description: "Timezone offset for scheduling calculations (default: UTC)",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "GMT-12 (Baker Island)", Value: "-12"},
						{Label: "GMT-11 (American Samoa)", Value: "-11"},
						{Label: "GMT-10 (Hawaii)", Value: "-10"},
						{Label: "GMT-9 (Alaska)", Value: "-9"},
						{Label: "GMT-8 (Los Angeles, Vancouver)", Value: "-8"},
						{Label: "GMT-7 (Denver, Phoenix)", Value: "-7"},
						{Label: "GMT-6 (Chicago, Mexico City)", Value: "-6"},
						{Label: "GMT-5 (New York, Toronto)", Value: "-5"},
						{Label: "GMT-4 (Santiago, Atlantic)", Value: "-4"},
						{Label: "GMT-3 (São Paulo, Buenos Aires)", Value: "-3"},
						{Label: "GMT-2 (South Georgia)", Value: "-2"},
						{Label: "GMT-1 (Azores)", Value: "-1"},
						{Label: "GMT+0 (London, Dublin, UTC)", Value: "0"},
						{Label: "GMT+1 (Paris, Berlin, Rome)", Value: "1"},
						{Label: "GMT+2 (Cairo, Helsinki, Athens)", Value: "2"},
						{Label: "GMT+3 (Moscow, Istanbul, Riyadh)", Value: "3"},
						{Label: "GMT+4 (Dubai, Baku)", Value: "4"},
						{Label: "GMT+5 (Karachi, Tashkent)", Value: "5"},
						{Label: "GMT+5:30 (Mumbai, Delhi)", Value: "5.5"},
						{Label: "GMT+6 (Dhaka, Almaty)", Value: "6"},
						{Label: "GMT+7 (Bangkok, Jakarta)", Value: "7"},
						{Label: "GMT+8 (Beijing, Singapore, Perth)", Value: "8"},
						{Label: "GMT+9 (Tokyo, Seoul)", Value: "9"},
						{Label: "GMT+9:30 (Adelaide)", Value: "9.5"},
						{Label: "GMT+10 (Sydney, Melbourne)", Value: "10"},
						{Label: "GMT+11 (Solomon Islands)", Value: "11"},
						{Label: "GMT+12 (Auckland, Fiji)", Value: "12"},
						{Label: "GMT+13 (Tonga, Samoa)", Value: "13"},
						{Label: "GMT+14 (Kiribati)", Value: "14"},
					},
				},
			},
		},
		{
			Name:     "type",
			Label:    "Schedule Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  TypeMinutes,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Every X minutes", Value: "minutes"},
						{Label: "Every X hours", Value: "hours"},
						{Label: "Every X days", Value: "days"},
						{Label: "Every X weeks", Value: "weeks"},
						{Label: "Every X months", Value: "months"},
						{Label: "Cron (Custom)", Value: "cron"},
					},
				},
			},
		},
		{
			Name:        "interval",
			Label:       "Interval",
			Type:        configuration.FieldTypeNumber,
			Default:     intPtr(1),
			Description: "Minutes (1-59), Hours (1-23), Days (1-31), Weeks (1-52), Months (1-24)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"minutes", "hours", "days", "weeks", "months"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "type", Values: []string{"minutes", "hours", "days", "weeks", "months"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(59), // Will be dynamically validated based on type
				},
			},
		},
		{
			Name:        "minute",
			Label:       "Trigger at minute",
			Type:        configuration.FieldTypeNumber,
			Default:     intPtr(0),
			Description: "Minute of the hour (0-59)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"hours", "days", "weeks", "months"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(0),
					Max: intPtr(59),
				},
			},
		},
		{
			Name:        "hour",
			Label:       "Trigger at hour",
			Type:        configuration.FieldTypeNumber,
			Default:     intPtr(0),
			Description: "Hour of the day (0-23)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"days", "weeks", "months"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(0),
					Max: intPtr(23),
				},
			},
		},
		{
			Name:        "weekDays",
			Label:       "Trigger on days of the week",
			Type:        configuration.FieldTypeMultiSelect,
			Default:     []string{WeekDayMonday},
			Description: "Select which days of the week to trigger",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"weeks"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "type", Values: []string{"weeks"}},
			},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Monday", Value: "monday"},
						{Label: "Tuesday", Value: "tuesday"},
						{Label: "Wednesday", Value: "wednesday"},
						{Label: "Thursday", Value: "thursday"},
						{Label: "Friday", Value: "friday"},
						{Label: "Saturday", Value: "saturday"},
						{Label: "Sunday", Value: "sunday"},
					},
				},
			},
		},
		{
			Name:        "dayOfMonth",
			Label:       "Trigger on day of the month",
			Type:        configuration.FieldTypeNumber,
			Default:     intPtr(1),
			Description: "Day of the month (1-31)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"months"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "type", Values: []string{"months"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(31),
				},
			},
		},
		{
			Name:        "cronExpression",
			Label:       "Cron Expression",
			Type:        configuration.FieldTypeCron,
			Description: "Cron expression (e.g., '0 30 14 * * MON-FRI' for 2:30 PM weekdays). Valid wildcards: * , - /",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"cron"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "type", Values: []string{"cron"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Cron: &configuration.CronTypeOptions{},
			},
		},
	}
}

func (s *Schedule) Setup(ctx triggers.TriggerContext) error {
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
	timezone := parseTimezone(config.Timezone)
	nowInTZ := now.In(timezone)

	// Validate interval ranges based on type
	if config.Interval != nil {
		err := validateIntervalForType(config.Type, *config.Interval)
		if err != nil {
			return nil, err
		}
	}

	switch config.Type {
	case TypeMinutes:
		if config.Interval == nil {
			return nil, fmt.Errorf("interval is required for minutes schedule")
		}
		return nextMinutesTrigger(*config.Interval, nowInTZ, referenceTime)

	case TypeHours:
		if config.Interval == nil {
			return nil, fmt.Errorf("interval is required for hours schedule")
		}
		minute := 0
		if config.Minute != nil {
			minute = *config.Minute
		}
		return nextHoursTrigger(*config.Interval, minute, nowInTZ, referenceTime)

	case TypeDays:
		if config.Interval == nil {
			return nil, fmt.Errorf("interval is required for days schedule")
		}
		hour := 0
		if config.Hour != nil {
			hour = *config.Hour
		}
		minute := 0
		if config.Minute != nil {
			minute = *config.Minute
		}
		return nextDaysTrigger(*config.Interval, hour, minute, nowInTZ, referenceTime)

	case TypeWeeks:
		if config.Interval == nil {
			return nil, fmt.Errorf("interval is required for weeks schedule")
		}
		if config.WeekDays == nil || len(config.WeekDays) == 0 {
			return nil, fmt.Errorf("weekDays is required for weeks schedule")
		}
		hour := 0
		if config.Hour != nil {
			hour = *config.Hour
		}
		minute := 0
		if config.Minute != nil {
			minute = *config.Minute
		}
		return nextWeeksTrigger(*config.Interval, config.WeekDays, hour, minute, nowInTZ, referenceTime)

	case TypeMonths:
		if config.Interval == nil {
			return nil, fmt.Errorf("interval is required for months schedule")
		}
		if config.DayOfMonth == nil {
			return nil, fmt.Errorf("dayOfMonth is required for months schedule")
		}
		hour := 0
		if config.Hour != nil {
			hour = *config.Hour
		}
		minute := 0
		if config.Minute != nil {
			minute = *config.Minute
		}
		return nextMonthsTrigger(*config.Interval, *config.DayOfMonth, hour, minute, nowInTZ, referenceTime)

	case TypeCron:
		if config.CronExpression == nil {
			return nil, fmt.Errorf("cronExpression is required for cron schedule")
		}
		return nextCronTrigger(*config.CronExpression, nowInTZ)

	default:
		return nil, fmt.Errorf("unsupported schedule type: %s", config.Type)
	}
}

func nextMinutesTrigger(interval int, now time.Time, referenceTime *string) (*time.Time, error) {
	if interval < 1 || interval > 59 {
		return nil, fmt.Errorf("interval must be between 1 and 59 minutes, got: %d", interval)
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

func validateIntervalForType(scheduleType string, interval int) error {
	switch scheduleType {
	case TypeMinutes:
		if interval < 1 || interval > 59 {
			return fmt.Errorf("minutes interval must be between 1 and 59, got: %d", interval)
		}
	case TypeHours:
		if interval < 1 || interval > 23 {
			return fmt.Errorf("hours interval must be between 1 and 23, got: %d", interval)
		}
	case TypeDays:
		if interval < 1 || interval > 31 {
			return fmt.Errorf("days interval must be between 1 and 31, got: %d", interval)
		}
	case TypeWeeks:
		if interval < 1 || interval > 52 {
			return fmt.Errorf("weeks interval must be between 1 and 52, got: %d", interval)
		}
	case TypeMonths:
		if interval < 1 || interval > 24 {
			return fmt.Errorf("months interval must be between 1 and 24, got: %d", interval)
		}
	}
	return nil
}

func parseTimezone(timezoneStr *string) *time.Location {
	if timezoneStr == nil || *timezoneStr == "" {
		return time.UTC
	}

	offsetHours, err := strconv.ParseFloat(*timezoneStr, 64)
	if err != nil {
		return time.UTC
	}
	offsetSeconds := int(offsetHours * 3600)

	return time.FixedZone(fmt.Sprintf("GMT%+.1f", offsetHours), offsetSeconds)
}

func nextHoursTrigger(interval int, minute int, now time.Time, referenceTime *string) (*time.Time, error) {
	if interval < 1 || interval > 23 {
		return nil, fmt.Errorf("interval must be between 1 and 23 hours, got: %d", interval)
	}
	if minute < 0 || minute > 59 {
		return nil, fmt.Errorf("minute must be between 0 and 59, got: %d", minute)
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

	hoursElapsed := int(nowUTC.Sub(reference).Hours())
	if hoursElapsed < 0 {
		hoursElapsed = 0
	}
	completedIntervals := hoursElapsed / interval

	nextTriggerHours := (completedIntervals + 1) * interval
	nextTrigger := reference.Add(time.Duration(nextTriggerHours) * time.Hour)
	nextTrigger = time.Date(nextTrigger.Year(), nextTrigger.Month(), nextTrigger.Day(), nextTrigger.Hour(), minute, 0, 0, nextTrigger.Location())

	if nextTrigger.Before(nowUTC) || nextTrigger.Equal(nowUTC) {
		nextTrigger = nextTrigger.Add(time.Duration(interval) * time.Hour)
	}

	return &nextTrigger, nil
}

func nextDaysTrigger(interval int, hour int, minute int, now time.Time, referenceTime *string) (*time.Time, error) {
	if interval < 1 || interval > 31 {
		return nil, fmt.Errorf("interval must be between 1 and 31 days, got: %d", interval)
	}
	if hour < 0 || hour > 23 {
		return nil, fmt.Errorf("hour must be between 0 and 23, got: %d", hour)
	}
	if minute < 0 || minute > 59 {
		return nil, fmt.Errorf("minute must be between 0 and 59, got: %d", minute)
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

	daysElapsed := int(nowUTC.Sub(reference).Hours() / 24)
	if daysElapsed < 0 {
		daysElapsed = 0
	}
	completedIntervals := daysElapsed / interval

	nextTriggerDays := (completedIntervals + 1) * interval
	nextTrigger := reference.AddDate(0, 0, nextTriggerDays)
	nextTrigger = time.Date(nextTrigger.Year(), nextTrigger.Month(), nextTrigger.Day(), hour, minute, 0, 0, nextTrigger.Location())

	if nextTrigger.Before(nowUTC) || nextTrigger.Equal(nowUTC) {
		nextTrigger = nextTrigger.AddDate(0, 0, interval)
	}

	return &nextTrigger, nil
}

func nextWeeksTrigger(interval int, weekDays []string, hour int, minute int, now time.Time, referenceTime *string) (*time.Time, error) {
	if interval < 1 || interval > 52 {
		return nil, fmt.Errorf("interval must be between 1 and 52 weeks, got: %d", interval)
	}
	if hour < 0 || hour > 23 {
		return nil, fmt.Errorf("hour must be between 0 and 23, got: %d", hour)
	}
	if minute < 0 || minute > 59 {
		return nil, fmt.Errorf("minute must be between 0 and 59, got: %d", minute)
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

	// Find next occurrence of any of the specified weekdays
	validWeekdays := make(map[time.Weekday]bool)
	for _, dayStr := range weekDays {
		weekday, err := parseWeekday(dayStr)
		if err != nil {
			return nil, err
		}
		validWeekdays[weekday] = true
	}

	weeksElapsed := int(nowUTC.Sub(reference).Hours() / (24 * 7))
	if weeksElapsed < 0 {
		weeksElapsed = 0
	}
	completedIntervals := weeksElapsed / interval

	// Check if current week has valid days after the reference interval
	currentWeekStart := reference.AddDate(0, 0, completedIntervals*interval*7)
	for i := 0; i < 14; i++ { // Check current and next week
		checkDate := currentWeekStart.AddDate(0, 0, i)
		if validWeekdays[checkDate.Weekday()] {
			candidateTime := time.Date(checkDate.Year(), checkDate.Month(), checkDate.Day(), hour, minute, 0, 0, checkDate.Location())
			if candidateTime.After(nowUTC) {
				return &candidateTime, nil
			}
		}
	}

	// If no valid day found in current interval, check next interval
	nextIntervalStart := reference.AddDate(0, 0, (completedIntervals+1)*interval*7)
	for i := 0; i < 7; i++ {
		checkDate := nextIntervalStart.AddDate(0, 0, i)
		if validWeekdays[checkDate.Weekday()] {
			candidateTime := time.Date(checkDate.Year(), checkDate.Month(), checkDate.Day(), hour, minute, 0, 0, checkDate.Location())
			return &candidateTime, nil
		}
	}

	return nil, fmt.Errorf("no valid weekday found")
}

func nextMonthsTrigger(interval int, dayOfMonth int, hour int, minute int, now time.Time, referenceTime *string) (*time.Time, error) {
	if interval < 1 || interval > 24 {
		return nil, fmt.Errorf("interval must be between 1 and 24 months, got: %d", interval)
	}
	if dayOfMonth < 1 || dayOfMonth > 31 {
		return nil, fmt.Errorf("dayOfMonth must be between 1 and 31, got: %d", dayOfMonth)
	}
	if hour < 0 || hour > 23 {
		return nil, fmt.Errorf("hour must be between 0 and 23, got: %d", hour)
	}
	if minute < 0 || minute > 59 {
		return nil, fmt.Errorf("minute must be between 0 and 59, got: %d", minute)
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

	monthsElapsed := (nowUTC.Year()-reference.Year())*12 + int(nowUTC.Month()-reference.Month())
	if monthsElapsed < 0 {
		monthsElapsed = 0
	}
	completedIntervals := monthsElapsed / interval

	nextTriggerMonths := (completedIntervals + 1) * interval
	nextTrigger := reference.AddDate(0, nextTriggerMonths, 0)
	nextTrigger = time.Date(nextTrigger.Year(), nextTrigger.Month(), dayOfMonth, hour, minute, 0, 0, nextTrigger.Location())

	if nextTrigger.Before(nowUTC) || nextTrigger.Equal(nowUTC) || nextTrigger.Day() != dayOfMonth {
		nextTrigger = reference.AddDate(0, (completedIntervals+1)*interval, 0)
		nextTrigger = time.Date(nextTrigger.Year(), nextTrigger.Month(), dayOfMonth, hour, minute, 0, 0, nextTrigger.Location())
	}

	return &nextTrigger, nil
}

func nextCronTrigger(cronExpression string, now time.Time) (*time.Time, error) {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(cronExpression)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	nextTime := schedule.Next(now)
	return &nextTime, nil
}

func intPtr(v int) *int {
	return &v
}
