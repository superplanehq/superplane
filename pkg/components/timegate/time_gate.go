package timegate

import (
	"fmt"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterComponent("time_gate", &TimeGate{})
}

const (
	TimeGateIncludeRangeMode    = "include_range"
	TimeGateExcludeRangeMode    = "exclude_range"
	TimeGateIncludeSpecificMode = "include_specific"
	TimeGateExcludeSpecificMode = "exclude_specific"
)

type TimeGate struct{}

type Metadata struct {
	NextValidTime *string `json:"nextValidTime"`
}

type Spec struct {
	Mode            string   `json:"mode"`
	StartTime       string   `json:"startTime"`
	EndTime         string   `json:"endTime"`
	Days            []string `json:"days"`
	StartDayInYear  string   `json:"startDayInYear,omitempty"`
	EndDayInYear    string   `json:"endDayInYear,omitempty"`
	Timezone        string   `json:"timezone,omitempty"`
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

func (tg *TimeGate) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (tg *TimeGate) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "mode",
			Label:    "Mode",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label: "Include Range",
							Value: TimeGateIncludeRangeMode,
						},
						{
							Label: "Exclude Range",
							Value: TimeGateExcludeRangeMode,
						},
						{
							Label: "Include Specific Times",
							Value: TimeGateIncludeSpecificMode,
						},
						{
							Label: "Exclude Specific Times",
							Value: TimeGateExcludeSpecificMode,
						},
					},
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
		{
			Name:  "days",
			Label: "Days of Week",
			Type:  configuration.FieldTypeMultiSelect,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "mode",
					Values: []string{TimeGateIncludeRangeMode, TimeGateExcludeRangeMode},
				},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{
					Field:  "mode",
					Values: []string{TimeGateIncludeRangeMode, TimeGateExcludeRangeMode},
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
			Name:        "startDayInYear",
			Label:       "Start Day (MM/DD)",
			Type:        configuration.FieldTypeDayInYear,
			Required:    false,
			Description: "Start day in MM/DD format (e.g., 12/25 for Christmas)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "mode",
					Values: []string{TimeGateIncludeSpecificMode, TimeGateExcludeSpecificMode},
				},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{
					Field:  "mode",
					Values: []string{TimeGateIncludeSpecificMode, TimeGateExcludeSpecificMode},
				},
			},
			ValidationRules: []configuration.ValidationRule{
				{
					Type:        configuration.ValidationRuleLessThan,
					CompareWith: "endDayInYear",
					Message:     "start day must be before end day",
				},
			},
		},
		{
			Name:        "endDayInYear",
			Label:       "End Day (MM/DD)",
			Type:        configuration.FieldTypeDayInYear,
			Required:    false,
			Description: "End day in MM/DD format (e.g., 01/01 for New Year)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "mode",
					Values: []string{TimeGateIncludeSpecificMode, TimeGateExcludeSpecificMode},
				},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{
					Field:  "mode",
					Values: []string{TimeGateIncludeSpecificMode, TimeGateExcludeSpecificMode},
				},
			},
			ValidationRules: []configuration.ValidationRule{
				{
					Type:        configuration.ValidationRuleGreaterThan,
					CompareWith: "startDayInYear",
					Message:     "end day must be after start day",
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

func (tg *TimeGate) Setup(ctx components.SetupContext) error {
	return nil
}

func (tg *TimeGate) Execute(ctx components.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	err = tg.validateSpec(spec)
	if err != nil {
		return err
	}

	var metadata Metadata
	err = mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	timezone := tg.parseTimezone(spec.Timezone)
	now := time.Now().In(timezone)
	nextValidTime := tg.findNextValidTime(now, spec)

	if nextValidTime.IsZero() {
		switch spec.Mode {
		case TimeGateIncludeSpecificMode:
			return fmt.Errorf("no valid time window found: the specified day range (%s to %s) has already passed for this year", spec.StartDayInYear, spec.EndDayInYear)
		case TimeGateExcludeSpecificMode:
			return fmt.Errorf("no valid time window found: the specified day range (%s to %s) has already passed for this year", spec.StartDayInYear, spec.EndDayInYear)
		default:
			return fmt.Errorf("no valid time window found: check your time configuration and selected days")
		}
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
		return ctx.ExecutionStateContext.Pass(map[string][]any{
			components.DefaultOutputChannel.Name: {ctx.Data},
		})
	}

	//
	// Schedule the action and save the next valid time in metadata
	//
	err = ctx.RequestContext.ScheduleActionCall("timeReached", map[string]any{}, interval)
	if err != nil {
		return err
	}

	formatted := nextValidTime.Format(time.RFC3339)
	ctx.MetadataContext.Set(Metadata{
		NextValidTime: &formatted,
	})

	return nil
}

func (tg *TimeGate) validateSpec(spec Spec) error {
	validModes := map[string]bool{
		TimeGateIncludeRangeMode:    true,
		TimeGateExcludeRangeMode:    true,
		TimeGateIncludeSpecificMode: true,
		TimeGateExcludeSpecificMode: true,
	}

	if !validModes[spec.Mode] {
		return fmt.Errorf("invalid mode '%s': must be one of include_range, exclude_range, include_specific, exclude_specific", spec.Mode)
	}

	if spec.Mode == TimeGateIncludeRangeMode || spec.Mode == TimeGateExcludeRangeMode {
		startTime, err := parseTimeString(spec.StartTime)
		if err != nil {
			return fmt.Errorf("startTime error: %w", err)
		}

		endTime, err := parseTimeString(spec.EndTime)
		if err != nil {
			return fmt.Errorf("endTime error: %w", err)
		}

		if startTime >= endTime {
			return fmt.Errorf("start time (%s) must be before end time (%s)", spec.StartTime, spec.EndTime)
		}
	}

	if spec.Mode == TimeGateIncludeSpecificMode || spec.Mode == TimeGateExcludeSpecificMode {
		if spec.StartDayInYear == "" || spec.EndDayInYear == "" {
			return fmt.Errorf("startDayInYear and endDayInYear are required for specific time modes")
		}

		err := tg.validateDayInYear(spec.StartDayInYear)
		if err != nil {
			return fmt.Errorf("startDayInYear error: %w", err)
		}

		err = tg.validateDayInYear(spec.EndDayInYear)
		if err != nil {
			return fmt.Errorf("endDayInYear error: %w", err)
		}

		startMonth, startDay, _ := tg.parseDayInYear(spec.StartDayInYear)
		endMonth, endDay, _ := tg.parseDayInYear(spec.EndDayInYear)

		// For cross-year ranges (e.g., 12/25 to 01/05), we allow this
		// Same day is allowed (e.g., 07/04 to 07/04 for Independence Day)
		if startMonth == endMonth && startDay > endDay {
			return fmt.Errorf("start day (%s) must be before or same as end day (%s) when in the same month", spec.StartDayInYear, spec.EndDayInYear)
		}
	}

	if (spec.Mode == TimeGateIncludeRangeMode || spec.Mode == TimeGateExcludeRangeMode) && len(spec.Days) == 0 {
		return fmt.Errorf("at least one day must be selected")
	}

	validDays := map[string]bool{
		"monday": true, "tuesday": true, "wednesday": true, "thursday": true,
		"friday": true, "saturday": true, "sunday": true,
	}
	for _, day := range spec.Days {
		if !validDays[day] {
			return fmt.Errorf("invalid day '%s': must be one of monday, tuesday, wednesday, thursday, friday, saturday, sunday", day)
		}
	}

	return nil
}

func (tg *TimeGate) Actions() []components.Action {
	return []components.Action{
		{
			Name: "timeReached",
		},
	}
}

func (tg *TimeGate) HandleAction(ctx components.ActionContext) error {
	switch ctx.Name {
	case "timeReached":
		return ctx.ExecutionStateContext.Pass(map[string][]any{
			components.DefaultOutputChannel.Name: {map[string]any{}},
		})

	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (tg *TimeGate) configEqual(a, b Spec) bool {

	if a.Mode != b.Mode {
		return false
	}

	if a.StartTime != b.StartTime || a.EndTime != b.EndTime {
		return false
	}

	if len(a.Days) != len(b.Days) {
		return false
	}

	aDays := make(map[string]bool)
	for _, day := range a.Days {
		aDays[day] = true
	}

	for _, day := range b.Days {
		if !aDays[day] {
			return false
		}
	}

	return true
}

func (tg *TimeGate) findNextValidTime(now time.Time, spec Spec) time.Time {
	switch spec.Mode {
	case TimeGateIncludeRangeMode:
		return tg.findNextIncludeTime(now, spec)
	case TimeGateExcludeRangeMode:
		return tg.findNextExcludeEndTime(now, spec)
	case TimeGateIncludeSpecificMode:
		return tg.findNextIncludeSpecificTime(now, spec)
	case TimeGateExcludeSpecificMode:
		return tg.findNextExcludeSpecificEndTime(now, spec)
	default:
		return time.Time{}
	}
}

func (tg *TimeGate) findNextIncludeTime(now time.Time, spec Spec) time.Time {
	startTime, _ := parseTimeString(spec.StartTime)
	endTime, _ := parseTimeString(spec.EndTime)

	currentDay := getDayString(now.Weekday())
	isDayMatch := contains(spec.Days, currentDay)
	currentTime := now.Hour()*60 + now.Minute()
	isTimeInWindow := isTimeInRange(currentTime, startTime, endTime)

	if isDayMatch && isTimeInWindow {
		return now
	}

	for i := 0; i < 8; i++ {
		checkDate := now.AddDate(0, 0, i)
		dayString := getDayString(checkDate.Weekday())

		if contains(spec.Days, dayString) {
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
	startTime, _ := parseTimeString(spec.StartTime)
	endTime, _ := parseTimeString(spec.EndTime)

	currentDay := getDayString(now.Weekday())
	isDayMatch := contains(spec.Days, currentDay)
	currentTime := now.Hour()*60 + now.Minute()
	isTimeInWindow := isTimeInRange(currentTime, startTime, endTime)

	if !isDayMatch || !isTimeInWindow {
		return now
	}

	endHour := endTime / 60
	endMinute := endTime % 60

	endOfWindow := time.Date(
		now.Year(), now.Month(), now.Day(),
		endHour, endMinute, 0, 0, now.Location(),
	)

	if endOfWindow.After(now) {
		return endOfWindow
	}

	return now
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

func (tg *TimeGate) findNextIncludeSpecificTime(now time.Time, spec Spec) time.Time {
	startMonth, startDay, err := tg.parseDayInYear(spec.StartDayInYear)
	if err != nil {
		return time.Time{}
	}

	endMonth, endDay, err := tg.parseDayInYear(spec.EndDayInYear)
	if err != nil {
		return time.Time{}
	}

	startTime, _ := parseTimeString(spec.StartTime)
	endTime, _ := parseTimeString(spec.EndTime)

	currentYear := now.Year()

	// Create the start and end datetime for this year
	startDateTime := time.Date(currentYear, time.Month(startMonth), startDay,
		startTime/60, startTime%60, 0, 0, now.Location())
	endDateTime := time.Date(currentYear, time.Month(endMonth), endDay,
		endTime/60, endTime%60, 0, 0, now.Location())

	// Handle cross-year ranges (e.g., Dec 25 to Jan 5)
	if startMonth > endMonth {
		// If we're before the start date, use this year's dates
		if now.Before(startDateTime) {
			return startDateTime
		}
		// If we're after start date but before new year, we're in the range
		if now.After(startDateTime) && now.Month() >= time.Month(startMonth) {
			return now
		}
		// If we're in the new year and before end date, we're in the range
		endDateTime = time.Date(currentYear+1, time.Month(endMonth), endDay,
			endTime/60, endTime%60, 0, 0, now.Location())
		if now.Before(endDateTime) && now.Month() <= time.Month(endMonth) {
			return now
		}
		// Check next year's start date
		nextStartDateTime := time.Date(currentYear+1, time.Month(startMonth), startDay,
			startTime/60, startTime%60, 0, 0, now.Location())
		if now.Before(nextStartDateTime) {
			return nextStartDateTime
		}
		return time.Time{}
	}

	// Normal same-year range
	if now.After(startDateTime) && now.Before(endDateTime) {
		return now
	}

	if now.Before(startDateTime) {
		return startDateTime
	}

	// Check next year
	nextStartDateTime := time.Date(currentYear+1, time.Month(startMonth), startDay,
		startTime/60, startTime%60, 0, 0, now.Location())
	return nextStartDateTime
}

func (tg *TimeGate) findNextExcludeSpecificEndTime(now time.Time, spec Spec) time.Time {
	startMonth, startDay, err := tg.parseDayInYear(spec.StartDayInYear)
	if err != nil {
		return time.Time{}
	}

	endMonth, endDay, err := tg.parseDayInYear(spec.EndDayInYear)
	if err != nil {
		return time.Time{}
	}

	startTime, _ := parseTimeString(spec.StartTime)
	endTime, _ := parseTimeString(spec.EndTime)

	currentYear := now.Year()

	// Create the start and end datetime for this year
	startDateTime := time.Date(currentYear, time.Month(startMonth), startDay,
		startTime/60, startTime%60, 0, 0, now.Location())
	endDateTime := time.Date(currentYear, time.Month(endMonth), endDay,
		endTime/60, endTime%60, 0, 0, now.Location())

	// Handle cross-year ranges (e.g., Dec 25 to Jan 5)
	if startMonth > endMonth {
		// If we're before the start date, we're outside the excluded range
		if now.Before(startDateTime) {
			return now
		}
		// If we're after start date but before new year, we're in the excluded range
		if now.After(startDateTime) && now.Month() >= time.Month(startMonth) {
			// Need to wait until next year's end date
			nextEndDateTime := time.Date(currentYear+1, time.Month(endMonth), endDay,
				endTime/60, endTime%60, 0, 0, now.Location())
			return nextEndDateTime
		}
		// If we're in the new year and before end date, we're in the excluded range
		endDateTime = time.Date(currentYear+1, time.Month(endMonth), endDay,
			endTime/60, endTime%60, 0, 0, now.Location())
		if now.Before(endDateTime) && now.Month() <= time.Month(endMonth) {
			return endDateTime
		}
		// We're after the end date, so we're outside the excluded range
		return now
	}

	// Normal same-year range
	if now.Before(startDateTime) || now.After(endDateTime) {
		return now
	}

	// We're inside the excluded range, return end time
	return endDateTime
}
