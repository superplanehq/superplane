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
	Type            string   `json:"type"`
	MinutesInterval *int     `json:"minutesInterval"` // 1-59 minutes between triggers
	HoursInterval   *int     `json:"hoursInterval"`   // 1-23 hours between triggers
	DaysInterval    *int     `json:"daysInterval"`    // 1-31 days between triggers
	WeeksInterval   *int     `json:"weeksInterval"`   // 1-52 weeks between triggers
	MonthsInterval  *int     `json:"monthsInterval"`  // 1-24 months between triggers
	Minute          *int     `json:"minute"`          // 0-59 for hours, days, weeks, months
	Hour            *int     `json:"hour"`            // 0-23 for days, weeks, months
	WeekDays        []string `json:"weekDays"`        // For weeks scheduling (multiple days)
	DayOfMonth      *int     `json:"dayOfMonth"`      // 1-31 for months scheduling
	CronExpression  *string  `json:"cronExpression"`  // For cron scheduling
	Timezone        *string  `json:"timezone"`        // Timezone offset (e.g., "0", "-5", "5.5")
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
			Type:        configuration.FieldTypeTimezone,
			Required:    true,
			Default:     "current",
			Description: "Timezone offset for scheduling calculations (default: your current timezone)",
			TypeOptions: &configuration.TypeOptions{
				Timezone: &configuration.TimezoneTypeOptions{},
			},
		},
		{
			Name:     "type",
			Label:    "Frequency",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  TypeMinutes,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Minutes", Value: "minutes"},
						{Label: "Hours", Value: "hours"},
						{Label: "Days", Value: "days"},
						{Label: "Weeks", Value: "weeks"},
						{Label: "Months", Value: "months"},
						{Label: "Cron (Custom)", Value: "cron"},
					},
				},
			},
		},
		{
			Name:        "minutesInterval",
			Label:       "Minutes between triggers",
			Type:        configuration.FieldTypeNumber,
			Default:     intPtr(1),
			Description: "Number of minutes between triggers (1-59)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"minutes"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "type", Values: []string{"minutes"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(59),
				},
			},
		},
		{
			Name:        "hoursInterval",
			Label:       "Hours between triggers",
			Type:        configuration.FieldTypeNumber,
			Default:     intPtr(1),
			Description: "Number of hours between triggers (1-23)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"hours"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "type", Values: []string{"hours"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(23),
				},
			},
		},
		{
			Name:        "daysInterval",
			Label:       "Days between triggers",
			Type:        configuration.FieldTypeNumber,
			Default:     intPtr(1),
			Description: "Number of days between triggers (1-31)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"days"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "type", Values: []string{"days"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(31),
				},
			},
		},
		{
			Name:        "weeksInterval",
			Label:       "Weeks between triggers",
			Type:        configuration.FieldTypeNumber,
			Default:     intPtr(1),
			Description: "Number of weeks between triggers (1-52)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"weeks"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "type", Values: []string{"weeks"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(52),
				},
			},
		},
		{
			Name:        "monthsInterval",
			Label:       "Months between triggers",
			Type:        configuration.FieldTypeNumber,
			Default:     intPtr(1),
			Description: "Number of months between triggers (1-24)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"months"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "type", Values: []string{"months"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(24),
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
			Name:        "cronExpression",
			Label:       "Cron Expression",
			Type:        configuration.FieldTypeCron,
			Description: "Cron expression in 5-field (e.g., '30 14 * * MON-FRI') or 6-field (e.g., '0 30 14 * * MON-FRI') format. Valid wildcards: * , - /",
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

	switch config.Type {
	case TypeMinutes:
		if config.MinutesInterval == nil {
			return nil, fmt.Errorf("minutesInterval is required for minutes schedule")
		}
		return nextMinutesTrigger(*config.MinutesInterval, nowInTZ, referenceTime)

	case TypeHours:
		if config.HoursInterval == nil {
			return nil, fmt.Errorf("hoursInterval is required for hours schedule")
		}
		minute := 0
		if config.Minute != nil {
			minute = *config.Minute
		}
		return nextHoursTrigger(*config.HoursInterval, minute, nowInTZ)

	case TypeDays:
		if config.DaysInterval == nil {
			return nil, fmt.Errorf("daysInterval is required for days schedule")
		}
		hour := 0
		if config.Hour != nil {
			hour = *config.Hour
		}
		minute := 0
		if config.Minute != nil {
			minute = *config.Minute
		}
		return nextDaysTrigger(*config.DaysInterval, hour, minute, nowInTZ)

	case TypeWeeks:
		if config.WeeksInterval == nil {
			return nil, fmt.Errorf("weeksInterval is required for weeks schedule")
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
		return nextWeeksTrigger(*config.WeeksInterval, config.WeekDays, hour, minute, nowInTZ)

	case TypeMonths:
		if config.MonthsInterval == nil {
			return nil, fmt.Errorf("monthsInterval is required for months schedule")
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
		return nextMonthsTrigger(*config.MonthsInterval, *config.DayOfMonth, hour, minute, nowInTZ)

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

	nowInTZ := now

	var reference time.Time
	if referenceTime != nil {
		var err error
		reference, err = time.Parse(time.RFC3339, *referenceTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse reference time: %w", err)
		}
		reference = reference.In(nowInTZ.Location())
	} else {
		reference = nowInTZ
	}

	minutesElapsed := int(nowInTZ.Sub(reference).Minutes())

	if minutesElapsed < 0 {
		minutesElapsed = 0
	}
	completedIntervals := minutesElapsed / interval

	nextTriggerMinutes := (completedIntervals + 1) * interval
	nextTrigger := reference.Add(time.Duration(nextTriggerMinutes) * time.Minute)

	if nextTrigger.Before(nowInTZ) || nextTrigger.Equal(nowInTZ) {
		nextTrigger = nextTrigger.Add(time.Duration(interval) * time.Minute)
	}

	utcResult := nextTrigger.UTC()
	return &utcResult, nil
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

func nextHoursTrigger(interval int, minute int, now time.Time) (*time.Time, error) {
	if interval < 1 || interval > 23 {
		return nil, fmt.Errorf("interval must be between 1 and 23 hours, got: %d", interval)
	}
	if minute < 0 || minute > 59 {
		return nil, fmt.Errorf("minute must be between 0 and 59, got: %d", minute)
	}

	nowInTZ := now

	// Start with the occurrence of the specified minute in the current hour
	nextTrigger := time.Date(nowInTZ.Year(), nowInTZ.Month(), nowInTZ.Day(), nowInTZ.Hour(), minute, 0, 0, nowInTZ.Location())
	nextTrigger = nextTrigger.Add(time.Duration(interval) * time.Hour)

	utcResult := nextTrigger.UTC()
	return &utcResult, nil
}

func nextDaysTrigger(interval int, hour int, minute int, now time.Time) (*time.Time, error) {
	if interval < 1 || interval > 31 {
		return nil, fmt.Errorf("interval must be between 1 and 31 days, got: %d", interval)
	}
	if hour < 0 || hour > 23 {
		return nil, fmt.Errorf("hour must be between 0 and 23, got: %d", hour)
	}
	if minute < 0 || minute > 59 {
		return nil, fmt.Errorf("minute must be between 0 and 59, got: %d", minute)
	}

	nowInTZ := now

	nextTrigger := nowInTZ.AddDate(0, 0, interval)
	nextTrigger = time.Date(nextTrigger.Year(), nextTrigger.Month(), nextTrigger.Day(), hour, minute, 0, 0, nextTrigger.Location())

	utcResult := nextTrigger.UTC()
	return &utcResult, nil
}

func nextWeeksTrigger(interval int, weekDays []string, hour int, minute int, now time.Time) (*time.Time, error) {
	if interval < 1 || interval > 52 {
		return nil, fmt.Errorf("interval must be between 1 and 52 weeks, got: %d", interval)
	}
	if hour < 0 || hour > 23 {
		return nil, fmt.Errorf("hour must be between 0 and 23, got: %d", hour)
	}
	if minute < 0 || minute > 59 {
		return nil, fmt.Errorf("minute must be between 0 and 59, got: %d", minute)
	}

	nowInTZ := now

	// Find next occurrence of any of the specified weekdays
	validWeekdays := make(map[time.Weekday]bool)
	for _, dayStr := range weekDays {
		weekday, err := parseWeekday(dayStr)
		if err != nil {
			return nil, err
		}
		validWeekdays[weekday] = true
	}

	nextIntervalStart := nowInTZ.AddDate(0, 0, interval*7)

	// start the search on Sunday of the next week
	nextIntervalStart.Add(-time.Duration(nextIntervalStart.Weekday()) * time.Hour)
	for i := 0; i < 7; i++ {
		checkDate := nextIntervalStart.AddDate(0, 0, i)
		if validWeekdays[checkDate.Weekday()] {
			candidateTime := time.Date(checkDate.Year(), checkDate.Month(), checkDate.Day(), hour, minute, 0, 0, checkDate.Location())
			utcResult := candidateTime.UTC()
			return &utcResult, nil
		}
	}

	return nil, fmt.Errorf("no valid weekday found")
}

func nextMonthsTrigger(interval int, dayOfMonth int, hour int, minute int, now time.Time) (*time.Time, error) {
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

	nowInTZ := now
	nextTriggerMonths := interval
	nextTrigger := nowInTZ.AddDate(0, nextTriggerMonths, 0)
	nextTrigger = time.Date(nextTrigger.Year(), nextTrigger.Month(), dayOfMonth, hour, minute, 0, 0, nextTrigger.Location())

	utcResult := nextTrigger.UTC()
	return &utcResult, nil
}

func nextCronTrigger(cronExpression string, now time.Time) (*time.Time, error) {
	// Count fields to determine format
	fields := strings.Fields(cronExpression)

	var parser cron.Parser

	switch len(fields) {
	case 5:
		// Standard 5-field cron: minute, hour, day of month, month, day of week
		parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	case 6:
		// Extended 6-field cron: second, minute, hour, day of month, month, day of week
		parser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	default:
		return nil, fmt.Errorf("cron expression must have either 5 fields (minute hour day month dayofweek) or 6 fields (second minute hour day month dayofweek), got %d fields", len(fields))
	}

	schedule, err := parser.Parse(cronExpression)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	nextTime := schedule.Next(now)
	// Convert result back to UTC for consistent API
	utcResult := nextTime.UTC()
	return &utcResult, nil
}

func intPtr(v int) *int {
	return &v
}
