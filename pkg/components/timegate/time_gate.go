package timegate

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterComponent("time_gate", &TimeGate{})
}

const (
	TimeGateIncludeMode = "include"
	TimeGateExcludeMode = "exclude"

	// Custom modes (replaces "custom" option)
	TimeGateCustomInclude = "custom_include" // "Run only during custom time window"
	TimeGateCustomExclude = "custom_exclude" // "Don't run during custom time windows"

	// Template modes
	TimeGateTemplateWorkingHours        = "template_working_hours"
	TimeGateTemplateOutsideWorkingHours = "template_outside_working_hours"
	TimeGateTemplateWeekends            = "template_weekends"
	TimeGateTemplateNoWeekends          = "template_no_weekends"

	TimeGateItemTypeWeekly       = "weekly"
	TimeGateItemTypeSpecificDate = "specific_dates"
)

type TimeGate struct{}

type Metadata struct {
	NextValidTime *string `json:"nextValidTime"`
}

type TimeGateItem struct {
	Type      string   `json:"type"`           // "weekly" or "specific_dates"
	Days      []string `json:"days,omitempty"` // For weekly type
	Date      string   `json:"date,omitempty"` // For specific_dates type (MM-DD format for recurring dates, or YYYY-MM-DD for backward compatibility)
	StartTime string   `json:"startTime"`      // Required for both types
	EndTime   string   `json:"endTime"`        // Required for both types
}

type Spec struct {
	WhenToRun string         `mapstructure:"when_to_run"`
	Mode      string         `mapstructure:"mode"`     // "include" or "exclude"
	Items     []TimeGateItem `mapstructure:"items"`    // List of time gate items
	Timezone  string         `mapstructure:"timezone"` // Required timezone
}

func (tg *TimeGate) Name() string {
	return "time_gate"
}

func (tg *TimeGate) Label() string {
	return "Time Gate"
}

func (tg *TimeGate) Description() string {
	return "Route events based on time conditions - include or exclude specific time windows"
}

func (tg *TimeGate) Icon() string {
	return "clock"
}

func (tg *TimeGate) Color() string {
	return "blue"
}

func (tg *TimeGate) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (tg *TimeGate) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "when_to_run",
			Label:    "When to run",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label: "Run only during custom time window",
							Value: TimeGateCustomInclude,
						},
						{
							Label: "Don't run during custom time windows",
							Value: TimeGateCustomExclude,
						},
						{
							Label: "Run during working hours",
							Value: TimeGateTemplateWorkingHours,
						},
						{
							Label: "Run outside of working hours",
							Value: TimeGateTemplateOutsideWorkingHours,
						},
						{
							Label: "Run on weekends",
							Value: TimeGateTemplateWeekends,
						},
						{
							Label: "Don't run on weekends",
							Value: TimeGateTemplateNoWeekends,
						},
					},
				},
			},
		},
		{
			Name:        "items",
			Label:       "Time Windows",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "List of time windows to include or exclude",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "when_to_run",
					Values: []string{TimeGateCustomInclude, TimeGateCustomExclude, TimeGateTemplateWorkingHours, TimeGateTemplateOutsideWorkingHours, TimeGateTemplateWeekends, TimeGateTemplateNoWeekends},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Time Window",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "type",
								Label:    "Type",
								Type:     configuration.FieldTypeSelect,
								Required: true,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{
												Label: "Weekly",
												Value: TimeGateItemTypeWeekly,
											},
											{
												Label: "Specific Day",
												Value: TimeGateItemTypeSpecificDate,
											},
										},
									},
								},
							},
							{
								Name:  "days",
								Label: "Days of Week",
								Type:  configuration.FieldTypeMultiSelect,
								VisibilityConditions: []configuration.VisibilityCondition{
									{
										Field:  "type",
										Values: []string{TimeGateItemTypeWeekly},
									},
								},
								RequiredConditions: []configuration.RequiredCondition{
									{
										Field:  "type",
										Values: []string{TimeGateItemTypeWeekly},
									},
								},
								Default: []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"},
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
								Name:  "date",
								Label: "Specific Day",
								Type:  configuration.FieldTypeDate,
								VisibilityConditions: []configuration.VisibilityCondition{
									{
										Field:  "type",
										Values: []string{TimeGateItemTypeSpecificDate},
									},
								},
								RequiredConditions: []configuration.RequiredCondition{
									{
										Field:  "type",
										Values: []string{TimeGateItemTypeSpecificDate},
									},
								},
							},
							{
								Name:        "startTime",
								Label:       "Start Time (HH:MM)",
								Type:        configuration.FieldTypeTime,
								Required:    true,
								Description: "Start time in HH:MM format (24-hour), e.g., 09:30",
								Default:     "09:00",
								ValidationRules: []configuration.ValidationRule{
									{
										Type:        configuration.ValidationRuleLessThan,
										CompareWith: "endTime",
										Message:     "start time must be before end time",
									},
								},
							},
							{
								Name:        "endTime",
								Label:       "End Time (HH:MM)",
								Type:        configuration.FieldTypeTime,
								Required:    true,
								Description: "End time in HH:MM format (24-hour), e.g., 17:30",
								Default:     "17:00",
								ValidationRules: []configuration.ValidationRule{
									{
										Type:        configuration.ValidationRuleGreaterThan,
										CompareWith: "startTime",
										Message:     "end time must be after start time",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "timezone",
			Label:       "Timezone",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Timezone offset for time-based calculations (default: UTC)",
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
	}
}

func (tg *TimeGate) Setup(ctx core.SetupContext) error {
	return nil
}

func (tg *TimeGate) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (tg *TimeGate) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	// Derive mode from when_to_run if it's a custom option
	if spec.WhenToRun == TimeGateCustomInclude {
		spec.Mode = TimeGateIncludeMode
	} else if spec.WhenToRun == TimeGateCustomExclude {
		spec.Mode = TimeGateExcludeMode
	}

	// Convert template to actual mode and items if a template is selected
	if spec.WhenToRun != "" && spec.WhenToRun != TimeGateCustomInclude && spec.WhenToRun != TimeGateCustomExclude {
		spec = tg.convertTemplateToSpec(spec)
	}

	err = tg.validateSpec(spec)
	if err != nil {
		return err
	}

	var metadata Metadata
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	timezone := tg.parseTimezone(spec.Timezone)
	now := time.Now().In(timezone)
	nextValidTime := tg.findNextValidTime(now, spec)

	if nextValidTime.IsZero() {
		return fmt.Errorf("no valid time window found: check your time configuration and items")
	}

	//
	// If the configuration didn't change, don't schedule a new action.
	//
	if metadata.NextValidTime != nil {
		currentValidTime, err := time.Parse(time.RFC3339, *metadata.NextValidTime)
		if err != nil {
			return fmt.Errorf("error parsing next valid time: %v", err)
		}

		if currentValidTime.Sub(nextValidTime).Abs() < time.Second {
			return nil
		}
	}

	interval := time.Until(nextValidTime)

	if interval <= 0 {
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"timegate.finished",
			[]any{ctx.Data},
		)
	}

	//
	// Schedule the action and save the next valid time in metadata
	//
	err = ctx.Requests.ScheduleActionCall("timeReached", map[string]any{}, interval)
	if err != nil {
		return err
	}

	formatted := nextValidTime.Format(time.RFC3339)
	return ctx.Metadata.Set(Metadata{
		NextValidTime: &formatted,
	})
}

func (tg *TimeGate) validateSpec(spec Spec) error {
	validModes := map[string]bool{
		TimeGateIncludeMode:                 true,
		TimeGateExcludeMode:                 true,
		TimeGateTemplateWorkingHours:        true,
		TimeGateTemplateOutsideWorkingHours: true,
		TimeGateTemplateWeekends:            true,
		TimeGateTemplateNoWeekends:          true,
	}

	if !validModes[spec.Mode] {
		return fmt.Errorf("invalid mode '%s': must be one of include, exclude, or a template", spec.Mode)
	}

	if len(spec.Items) == 0 {
		return fmt.Errorf("at least one time window item is required")
	}

	validDays := map[string]bool{
		"monday": true, "tuesday": true, "wednesday": true, "thursday": true,
		"friday": true, "saturday": true, "sunday": true,
	}

	for i, item := range spec.Items {
		if item.Type != TimeGateItemTypeWeekly && item.Type != TimeGateItemTypeSpecificDate {
			return fmt.Errorf("item %d: invalid type '%s': must be one of weekly, specific_dates", i, item.Type)
		}

		// Validate startTime and endTime
		startTime, err := parseTimeString(item.StartTime)
		if err != nil {
			return fmt.Errorf("item %d: startTime error: %w", i, err)
		}

		endTime, err := parseTimeString(item.EndTime)
		if err != nil {
			return fmt.Errorf("item %d: endTime error: %w", i, err)
		}

		if startTime >= endTime {
			return fmt.Errorf("item %d: start time (%s) must be before end time (%s)", i, item.StartTime, item.EndTime)
		}

		if item.Type == TimeGateItemTypeWeekly {
			if len(item.Days) == 0 {
				return fmt.Errorf("item %d: at least one day must be selected for weekly type", i)
			}

			for _, day := range item.Days {
				if !validDays[day] {
					return fmt.Errorf("item %d: invalid day '%s': must be one of monday, tuesday, wednesday, thursday, friday, saturday, sunday", i, day)
				}
			}
		}

		if item.Type == TimeGateItemTypeSpecificDate {
			if item.Date == "" {
				return fmt.Errorf("item %d: date is required for specific_dates type", i)
			}

			// Validate date format (MM-DD for recurring dates, or YYYY-MM-DD for backward compatibility)
			_, errMMDD := time.Parse("01-02", item.Date)
			_, errYYYYMMDD := time.Parse("2006-01-02", item.Date)
			if errMMDD != nil && errYYYYMMDD != nil {
				return fmt.Errorf("item %d: date must be in MM-DD format (recurring date) or YYYY-MM-DD format, got: %s", i, item.Date)
			}
		}
	}

	return nil
}

func (tg *TimeGate) Actions() []core.Action {
	return []core.Action{
		{
			Name: "timeReached",
		},
		{
			Name:           "pushThrough",
			Description:    "Push Through",
			UserAccessible: true,
		},
	}
}

func (tg *TimeGate) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "timeReached":
		return tg.HandleTimeReached(ctx)
	case "pushThrough":
		return tg.HandlePushThrough(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (tg *TimeGate) HandleTimeReached(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		// already handled, for example via "pushThrough" action
		return nil
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"timegate.finished",
		[]any{map[string]any{}},
	)
}

func (tg *TimeGate) HandlePushThrough(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		// already handled, for example via "timeReached" action
		return nil
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"timegate.finished",
		[]any{map[string]any{}},
	)
}

func (tg *TimeGate) configEqual(a, b Spec) bool {
	if a.Mode != b.Mode {
		return false
	}

	if len(a.Items) != len(b.Items) {
		return false
	}

	// Compare items (order matters for now, but we could make it order-independent if needed)
	for i, itemA := range a.Items {
		if i >= len(b.Items) {
			return false
		}
		itemB := b.Items[i]

		if itemA.Type != itemB.Type {
			return false
		}

		if itemA.StartTime != itemB.StartTime || itemA.EndTime != itemB.EndTime {
			return false
		}

		if itemA.Type == TimeGateItemTypeWeekly {
			if len(itemA.Days) != len(itemB.Days) {
				return false
			}

			aDays := make(map[string]bool)
			for _, day := range itemA.Days {
				aDays[day] = true
			}

			for _, day := range itemB.Days {
				if !aDays[day] {
					return false
				}
			}
		}

		if itemA.Type == TimeGateItemTypeSpecificDate {
			if itemA.Date != itemB.Date {
				return false
			}
		}
	}

	return true
}

func (tg *TimeGate) convertTemplateToSpec(spec Spec) Spec {
	switch spec.WhenToRun {
	case TimeGateTemplateWorkingHours:
		// Include: Mon-Fri 9:00-17:00
		return Spec{
			Mode:     TimeGateIncludeMode,
			Timezone: spec.Timezone,
			Items: []TimeGateItem{
				{
					Type:      TimeGateItemTypeWeekly,
					Days:      []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
					StartTime: "09:00",
					EndTime:   "17:00",
				},
			},
		}
	case TimeGateTemplateOutsideWorkingHours:
		// Exclude: Mon-Fri 9:00-17:00
		return Spec{
			Mode:     TimeGateExcludeMode,
			Timezone: spec.Timezone,
			Items: []TimeGateItem{
				{
					Type:      TimeGateItemTypeWeekly,
					Days:      []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
					StartTime: "09:00",
					EndTime:   "17:00",
				},
			},
		}
	case TimeGateTemplateWeekends:
		// Include: Sat-Sun 00:00-23:59
		return Spec{
			Mode:     TimeGateIncludeMode,
			Timezone: spec.Timezone,
			Items: []TimeGateItem{
				{
					Type:      TimeGateItemTypeWeekly,
					Days:      []string{"saturday", "sunday"},
					StartTime: "00:00",
					EndTime:   "23:59",
				},
			},
		}
	case TimeGateTemplateNoWeekends:
		// Exclude: Sat-Sun 00:00-23:59
		return Spec{
			Mode:     TimeGateExcludeMode,
			Timezone: spec.Timezone,
			Items: []TimeGateItem{
				{
					Type:      TimeGateItemTypeWeekly,
					Days:      []string{"saturday", "sunday"},
					StartTime: "00:00",
					EndTime:   "23:59",
				},
			},
		}
	default:
		// Not a template, return as-is
		return spec
	}
}

func (tg *TimeGate) findNextValidTime(now time.Time, spec Spec) time.Time {
	if spec.Mode == TimeGateIncludeMode {
		return tg.findNextIncludeTime(now, spec)
	}
	// Exclude mode: find next time when we're outside all excluded windows
	return tg.findNextExcludeEndTime(now, spec)
}

func (tg *TimeGate) findNextIncludeTime(now time.Time, spec Spec) time.Time {
	// For include mode, find the earliest next valid time across all items
	var candidates []time.Time

	for _, item := range spec.Items {
		var candidate time.Time
		if item.Type == TimeGateItemTypeWeekly {
			candidate = tg.findNextIncludeWeeklyTime(now, item)
		} else {
			candidate = tg.findNextIncludeSpecificTime(now, item)
		}

		if !candidate.IsZero() {
			candidates = append(candidates, candidate)
		}
	}

	if len(candidates) == 0 {
		return time.Time{}
	}

	// Return the earliest candidate
	earliest := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.Before(earliest) {
			earliest = candidate
		}
	}

	return earliest
}

func (tg *TimeGate) findNextIncludeWeeklyTime(now time.Time, item TimeGateItem) time.Time {
	startTime, _ := parseTimeString(item.StartTime)
	endTime, _ := parseTimeString(item.EndTime)

	currentDay := getDayString(now.Weekday())
	isDayMatch := contains(item.Days, currentDay)
	currentTime := now.Hour()*60 + now.Minute()
	isTimeInWindow := isTimeInRange(currentTime, startTime, endTime)

	if isDayMatch && isTimeInWindow {
		return now
	}

	for i := 0; i < 8; i++ {
		checkDate := now.AddDate(0, 0, i)
		dayString := getDayString(checkDate.Weekday())

		if contains(item.Days, dayString) {
			startHour := startTime / 60
			startMinute := startTime % 60

			candidateTime := time.Date(
				checkDate.Year(), checkDate.Month(), checkDate.Day(),
				startHour, startMinute, 0, 0, now.Location(),
			)

			if i == 0 && !candidateTime.After(now) {
				continue
			}

			return candidateTime
		}
	}

	return time.Time{}
}

func (tg *TimeGate) findNextExcludeEndTime(now time.Time, spec Spec) time.Time {
	// For exclude mode, check if we're currently in any excluded window
	// If yes, find the end of the earliest ending excluded window
	// If no, we're already valid
	var excludedEndTimes []time.Time

	for _, item := range spec.Items {
		var endTime time.Time
		var isInWindow bool

		if item.Type == TimeGateItemTypeWeekly {
			endTime, isInWindow = tg.isInExcludedWeeklyWindow(now, item)
		} else {
			endTime, isInWindow = tg.isInExcludedSpecificWindow(now, item)
		}

		if isInWindow && !endTime.IsZero() {
			excludedEndTimes = append(excludedEndTimes, endTime)
		}
	}

	// If we're not in any excluded window, we're valid now
	if len(excludedEndTimes) == 0 {
		return now
	}

	// Find the earliest end time (when we can exit the excluded window)
	earliest := excludedEndTimes[0]
	for _, endTime := range excludedEndTimes[1:] {
		if endTime.Before(earliest) {
			earliest = endTime
		}
	}

	return earliest
}

func (tg *TimeGate) isInExcludedWeeklyWindow(now time.Time, item TimeGateItem) (time.Time, bool) {
	startTime, _ := parseTimeString(item.StartTime)
	endTime, _ := parseTimeString(item.EndTime)

	currentDay := getDayString(now.Weekday())
	isDayMatch := contains(item.Days, currentDay)
	currentTime := now.Hour()*60 + now.Minute()
	isTimeInWindow := isTimeInRange(currentTime, startTime, endTime)

	if !isDayMatch || !isTimeInWindow {
		return time.Time{}, false
	}

	endHour := endTime / 60
	endMinute := endTime % 60

	endOfWindow := time.Date(
		now.Year(), now.Month(), now.Day(),
		endHour, endMinute, 0, 0, now.Location(),
	)

	return endOfWindow, true
}

func (tg *TimeGate) isInExcludedSpecificWindow(now time.Time, item TimeGateItem) (time.Time, bool) {
	// Parse the date (MM-DD format for recurring dates, or YYYY-MM-DD for backward compatibility)
	var selectedMonth time.Month
	var selectedDay int

	// Try MM-DD format first (recurring date)
	if parsedDate, err := time.Parse("01-02", item.Date); err == nil {
		selectedMonth = parsedDate.Month()
		selectedDay = parsedDate.Day()
	} else if parsedDate, err := time.Parse("2006-01-02", item.Date); err == nil {
		// Fallback to YYYY-MM-DD format (backward compatibility)
		selectedMonth = parsedDate.Month()
		selectedDay = parsedDate.Day()
	} else {
		return time.Time{}, false
	}

	// Check if today matches the selected month and day (recurring date)
	if now.Month() != selectedMonth || now.Day() != selectedDay {
		return time.Time{}, false
	}

	startTime, _ := parseTimeString(item.StartTime)
	endTime, _ := parseTimeString(item.EndTime)

	// Create the start and end datetime for today (matching the selected date)
	startDateTime := time.Date(now.Year(), now.Month(), now.Day(),
		startTime/60, startTime%60, 0, 0, now.Location())
	endDateTime := time.Date(now.Year(), now.Month(), now.Day(),
		endTime/60, endTime%60, 0, 0, now.Location())

	// Check if we're within the excluded time window on this date
	if now.Before(startDateTime) || now.After(endDateTime) {
		return time.Time{}, false
	}

	// We're inside the excluded range, return end time
	return endDateTime, true
}

func (tg *TimeGate) parseTimezone(timezoneStr string) *time.Location {
	if timezoneStr == "" {
		return time.UTC
	}

	offsetHours, err := strconv.ParseFloat(timezoneStr, 64)
	if err != nil {
		return time.UTC
	}
	offsetSeconds := int(offsetHours * 3600)

	return time.FixedZone(fmt.Sprintf("GMT%+.1f", offsetHours), offsetSeconds)
}

func parseTimeString(timeStr string) (int, error) {
	if timeStr == "" {
		return 0, fmt.Errorf("time string is empty")
	}

	var hour, minute int
	var extra string
	n, err := fmt.Sscanf(timeStr, "%d:%d%s", &hour, &minute, &extra)

	// Accept n=2 (valid format) but reject n=3 (extra characters)
	if n < 2 || n > 2 {
		return 0, fmt.Errorf("invalid time format '%s': expected HH:MM (e.g., 09:30)", timeStr)
	}

	// For n=2, err might be EOF (which is expected when no extra string)
	if err != nil && n < 2 {
		return 0, fmt.Errorf("invalid time format '%s': expected HH:MM (e.g., 09:30)", timeStr)
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, fmt.Errorf("invalid time values '%s': hour must be 0-23, minute must be 0-59", timeStr)
	}
	return hour*60 + minute, nil
}

func isTimeInRange(currentTime, startTime, endTime int) bool {
	if startTime <= endTime {
		return currentTime >= startTime && currentTime <= endTime
	}
	return currentTime >= startTime || currentTime <= endTime
}

func getDayString(weekday time.Weekday) string {
	days := []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
	return days[weekday]
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (tg *TimeGate) validateDayInYear(dayStr string) error {
	_, _, err := tg.parseDayInYear(dayStr)
	return err
}

func (tg *TimeGate) parseDayInYear(dayStr string) (int, int, error) {
	if dayStr == "" {
		return 0, 0, fmt.Errorf("day string is empty")
	}

	var month, day int
	var extra string
	n, err := fmt.Sscanf(dayStr, "%d/%d%s", &month, &day, &extra)

	// Accept n=2 (valid format) but reject n=3 (extra characters)
	if n < 2 || n > 2 {
		return 0, 0, fmt.Errorf("invalid day format '%s': expected MM/DD (e.g., 12/25)", dayStr)
	}

	// For n=2, err might be EOF (which is expected when no extra string)
	if err != nil && n < 2 {
		return 0, 0, fmt.Errorf("invalid day format '%s': expected MM/DD (e.g., 12/25)", dayStr)
	}

	if month < 1 || month > 12 || day < 1 || day > 31 {
		return 0, 0, fmt.Errorf("invalid day values '%s': month must be 1-12, day must be 1-31", dayStr)
	}

	// Additional validation for days per month
	daysInMonth := []int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31} // Feb has 29 to account for leap years
	if day > daysInMonth[month] {
		return 0, 0, fmt.Errorf("invalid day '%d' for month '%d'", day, month)
	}

	return month, day, nil
}

func (tg *TimeGate) findNextIncludeSpecificTime(now time.Time, item TimeGateItem) time.Time {
	// Parse the date (MM-DD format for recurring dates, or YYYY-MM-DD for backward compatibility)
	var selectedMonth time.Month
	var selectedDay int

	// Try MM-DD format first (recurring date)
	if parsedDate, err := time.Parse("01-02", item.Date); err == nil {
		selectedMonth = parsedDate.Month()
		selectedDay = parsedDate.Day()
	} else if parsedDate, err := time.Parse("2006-01-02", item.Date); err == nil {
		// Fallback to YYYY-MM-DD format (backward compatibility)
		selectedMonth = parsedDate.Month()
		selectedDay = parsedDate.Day()
	} else {
		return time.Time{}
	}

	startTime, _ := parseTimeString(item.StartTime)
	endTime, _ := parseTimeString(item.EndTime)

	// Find the next occurrence of this date (month/day) - could be today or next year
	var targetYear int

	// Check if the date occurs this year (today or later)
	thisYearDate := time.Date(now.Year(), selectedMonth, selectedDay,
		startTime/60, startTime%60, 0, 0, now.Location())

	if thisYearDate.After(now) || (thisYearDate.Year() == now.Year() && thisYearDate.Month() == now.Month() && thisYearDate.Day() == now.Day()) {
		// The date occurs this year (today or in the future)
		targetYear = now.Year()
	} else {
		// The date has passed this year, use next year
		targetYear = now.Year() + 1
	}

	// Create the start and end datetime for the target date
	startDateTime := time.Date(targetYear, selectedMonth, selectedDay,
		startTime/60, startTime%60, 0, 0, now.Location())
	endDateTime := time.Date(targetYear, selectedMonth, selectedDay,
		endTime/60, endTime%60, 0, 0, now.Location())

	// If we're within the time window on this date, return now
	if now.After(startDateTime) && now.Before(endDateTime) {
		return now
	}

	// If we're before the start time on this date, return start time
	if now.Before(startDateTime) {
		return startDateTime
	}

	// We're after the end time on this date, so this date has passed
	// Return zero time to indicate no valid time found
	return time.Time{}
}

func (tg *TimeGate) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (tg *TimeGate) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
