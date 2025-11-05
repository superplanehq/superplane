package timegate

import (
	"fmt"
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
	Mode          string   `json:"mode"`
	StartTime     string   `json:"startTime"`
	EndTime       string   `json:"endTime"`
	Days          []string `json:"days"`
	StartDateTime string   `json:"startDateTime,omitempty"`
	EndDateTime   string   `json:"endDateTime,omitempty"`
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
			Required:    false,
			Description: "Start time in HH:MM format (24-hour), e.g., 09:30",
			Default:     "09:00",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "mode",
					Values: []string{TimeGateIncludeRangeMode, TimeGateExcludeRangeMode},
				},
			},
		},
		{
			Name:        "endTime",
			Label:       "End Time (HH:MM)",
			Type:        configuration.FieldTypeTime,
			Required:    false,
			Description: "End time in HH:MM format (24-hour), e.g., 17:30",
			Default:     "17:00",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "mode",
					Values: []string{TimeGateIncludeRangeMode, TimeGateExcludeRangeMode},
				},
			},
		},
		{
			Name:     "days",
			Label:    "Days of Week",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"},
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
			Name:        "startDateTime",
			Label:       "Start Date & Time",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "Start date and time for specific time window (e.g., 2024-12-31T00:00)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "mode",
					Values: []string{TimeGateIncludeSpecificMode, TimeGateExcludeSpecificMode},
				},
			},
		},
		{
			Name:        "endDateTime",
			Label:       "End Date & Time",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "End date and time for specific time window (e.g., 2025-01-01T23:59)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "mode",
					Values: []string{TimeGateIncludeSpecificMode, TimeGateExcludeSpecificMode},
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

	now := time.Now().UTC()
	nextValidTime := tg.findNextValidTime(now, spec)

	if nextValidTime.IsZero() {
		return fmt.Errorf("no valid time window found")
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
		if spec.StartDateTime == "" || spec.EndDateTime == "" {
			return fmt.Errorf("startDateTime and endDateTime are required for specific time modes")
		}

		startDateTime, err := time.Parse("2006-01-02T15:04", spec.StartDateTime)
		if err != nil {
			return fmt.Errorf("invalid startDateTime format '%s': expected YYYY-MM-DDTHH:MM", spec.StartDateTime)
		}

		endDateTime, err := time.Parse("2006-01-02T15:04", spec.EndDateTime)
		if err != nil {
			return fmt.Errorf("invalid endDateTime format '%s': expected YYYY-MM-DDTHH:MM", spec.EndDateTime)
		}

		if !startDateTime.Before(endDateTime) {
			return fmt.Errorf("start datetime (%s) must be before end datetime (%s)", spec.StartDateTime, spec.EndDateTime)
		}
	}

	if len(spec.Days) == 0 {
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

func (tg *TimeGate) findNextIncludeSpecificTime(now time.Time, spec Spec) time.Time {
	startDateTime, err := time.Parse("2006-01-02T15:04", spec.StartDateTime)
	if err != nil {
		return time.Time{}
	}

	endDateTime, err := time.Parse("2006-01-02T15:04", spec.EndDateTime)
	if err != nil {
		return time.Time{}
	}

	startDateTime = startDateTime.In(now.Location())
	endDateTime = endDateTime.In(now.Location())

	if now.After(startDateTime) && now.Before(endDateTime) {
		return now
	}

	if now.Before(startDateTime) {
		return startDateTime
	}

	return time.Time{}
}

func (tg *TimeGate) findNextExcludeSpecificEndTime(now time.Time, spec Spec) time.Time {
	startDateTime, err := time.Parse("2006-01-02T15:04", spec.StartDateTime)
	if err != nil {
		return time.Time{}
	}

	endDateTime, err := time.Parse("2006-01-02T15:04", spec.EndDateTime)
	if err != nil {
		return time.Time{}
	}

	startDateTime = startDateTime.In(now.Location())
	endDateTime = endDateTime.In(now.Location())

	if now.Before(startDateTime) || now.After(endDateTime) {
		return now
	}

	return endDateTime
}
