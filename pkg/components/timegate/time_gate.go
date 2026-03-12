package timegate

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterComponent("timeGate", &TimeGate{})
}

type TimeGate struct{}

type Metadata struct {
	NextValidTime   *string    `json:"nextValidTime"`
	PushedThroughBy *core.User `json:"pushedThroughBy,omitempty"`
	PushedThroughAt *string    `json:"pushedThroughAt,omitempty"`
}

type Spec struct {
	Days         []string `json:"days"`
	TimeRange    string   `json:"timeRange"`
	Timezone     string   `json:"timezone,omitempty"`
	ExcludeDates []string `json:"excludeDates,omitempty"`
}

func (tg *TimeGate) Name() string {
	return "timeGate"
}

func (tg *TimeGate) Label() string {
	return "Time Gate"
}

func (tg *TimeGate) Description() string {
	return "Route events based on active days and time windows, with optional excluded dates"
}

func (tg *TimeGate) Documentation() string {
	return `The Time Gate component delays event processing until the next valid day and time window, with optional excluded dates.

## Use Cases

- **Business hours**: Only process events during business hours
- **Scheduled releases**: Delay deployments until off-peak hours
- **Holiday handling**: Exclude specific dates from processing
- **Time-based routing**: Route events based on time of day or specific dates

## Configuration

- **Active Days**: Days of the week when the gate can open
- **Active Time**: Start and end times in HH:MM-HH:MM format (24-hour)
- **Timezone**: Timezone offset for time calculations (default: current)
- **Exclude Dates**: Specific MM/DD dates that override the rules above

## Behavior

- Events wait until the next valid time window is reached
- Exclude dates override the day/time rules
- Can be manually pushed through using the "Push Through" action
- Automatically schedules execution when the time window is reached`
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
			Name:        "days",
			Label:       "Active Days",
			Type:        configuration.FieldTypeDaysOfWeek,
			Required:    true,
			Description: "Select the days of the week when the gate can open",
			Default:     []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"},
		},
		{
			Name:        "timeRange",
			Label:       "Active Time",
			Type:        configuration.FieldTypeTimeRange,
			Required:    true,
			Description: "Time range in HH:MM-HH:MM format (24-hour), e.g., 09:00-17:30",
			Default:     "00:00-23:59",
		},
		{
			Name:        "timezone",
			Label:       "Timezone",
			Type:        configuration.FieldTypeTimezone,
			Required:    true,
			Description: "Timezone offset for time-based calculations (default: current)",
			Default:     "current",
		},
		{
			Name:        "excludeDates",
			Label:       "Exclude Dates (MM/DD)",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional list of specific dates (MM/DD) to exclude, such as holidays",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Date",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeDayInYear,
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
	startMinutes, endMinutes, err := parseTimeRangeString(spec.TimeRange)
	if err != nil {
		return err
	}

	nextValidTime := tg.findNextValidTime(now, spec, startMinutes, endMinutes)

	if nextValidTime.IsZero() {
		return fmt.Errorf("no valid time window found: check your time gate configuration")
	}

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
	if len(spec.Days) == 0 {
		return fmt.Errorf("at least one day must be selected")
	}

	validDays := []string{
		"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday",
	}
	for _, day := range spec.Days {
		if !slices.Contains(validDays, day) {
			return fmt.Errorf("invalid day '%s': must be one of monday, tuesday, wednesday, thursday, friday, saturday, sunday", day)
		}
	}

	startMinutes, endMinutes, err := parseTimeRangeString(spec.TimeRange)
	if err != nil {
		return err
	}

	if startMinutes >= endMinutes {
		return fmt.Errorf("start time must be before end time")
	}

	if len(spec.ExcludeDates) > 0 {
		seen := map[string]bool{}
		for _, dateStr := range spec.ExcludeDates {
			month, day, err := tg.parseDayInYear(dateStr)
			if err != nil {
				return fmt.Errorf("excludeDates error: %w", err)
			}
			key := formatDayKey(month, day)
			if seen[key] {
				return fmt.Errorf("excludeDates contains duplicate date '%s'", key)
			}
			seen[key] = true
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
		return nil
	}

	var metadata Metadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	pushedThroughAt := time.Now().Format(time.RFC3339)
	if ctx.Auth != nil {
		metadata.PushedThroughBy = ctx.Auth.AuthenticatedUser()
	}
	metadata.PushedThroughAt = &pushedThroughAt

	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"timegate.finished",
		[]any{map[string]any{}},
	)
}

func (tg *TimeGate) findNextValidTime(now time.Time, spec Spec, startMinutes int, endMinutes int) time.Time {
	excludedDates := tg.buildExcludedDateSet(spec.ExcludeDates)

	for i := 0; i <= 366; i++ {
		checkDate := now.AddDate(0, 0, i)
		if !tg.isActiveDay(checkDate, spec.Days) {
			continue
		}

		if tg.isExcludedDate(checkDate, excludedDates) {
			continue
		}

		if i == 0 {
			currentMinutes := now.Hour()*60 + now.Minute()
			if currentMinutes < startMinutes {
				return time.Date(
					checkDate.Year(), checkDate.Month(), checkDate.Day(),
					startMinutes/60, startMinutes%60, 0, 0, now.Location(),
				)
			}

			if currentMinutes >= startMinutes && currentMinutes <= endMinutes {
				return now
			}

			continue
		}

		return time.Date(
			checkDate.Year(), checkDate.Month(), checkDate.Day(),
			startMinutes/60, startMinutes%60, 0, 0, now.Location(),
		)
	}

	return time.Time{}
}

func (tg *TimeGate) buildExcludedDateSet(excludeDates []string) map[string]struct{} {
	if len(excludeDates) == 0 {
		return nil
	}

	excluded := map[string]struct{}{}
	for _, dateStr := range excludeDates {
		month, day, err := tg.parseDayInYear(dateStr)
		if err != nil {
			continue
		}
		excluded[formatDayKey(month, day)] = struct{}{}
	}

	return excluded
}

func (tg *TimeGate) isActiveDay(date time.Time, days []string) bool {
	dayString := getDayString(date.Weekday())
	return slices.Contains(days, dayString)
}

func (tg *TimeGate) isExcludedDate(date time.Time, excludedDates map[string]struct{}) bool {
	if len(excludedDates) == 0 {
		return false
	}

	_, ok := excludedDates[formatDayKey(int(date.Month()), date.Day())]
	return ok
}

func (tg *TimeGate) parseTimezone(timezoneStr string) *time.Location {
	if timezoneStr == "" || timezoneStr == "current" {
		return time.Local
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

	if n < 2 || n > 2 {
		return 0, fmt.Errorf("invalid time format '%s': expected HH:MM (e.g., 09:30)", timeStr)
	}

	if err != nil && n < 2 {
		return 0, fmt.Errorf("invalid time format '%s': expected HH:MM (e.g., 09:30)", timeStr)
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, fmt.Errorf("invalid time values '%s': hour must be 0-23, minute must be 0-59", timeStr)
	}
	return hour*60 + minute, nil
}

func parseTimeRangeString(timeRangeStr string) (int, int, error) {
	if timeRangeStr == "" {
		return 0, 0, fmt.Errorf("timeRange error: time range cannot be empty")
	}

	parts := strings.SplitN(timeRangeStr, "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("timeRange error: must be in HH:MM-HH:MM format")
	}

	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	startMinutes, err := parseTimeString(startStr)
	if err != nil {
		return 0, 0, fmt.Errorf("timeRange error: %w", err)
	}

	endMinutes, err := parseTimeString(endStr)
	if err != nil {
		return 0, 0, fmt.Errorf("timeRange error: %w", err)
	}

	return startMinutes, endMinutes, nil
}

func getDayString(weekday time.Weekday) string {
	days := []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
	return days[weekday]
}

func formatDayKey(month int, day int) string {
	return fmt.Sprintf("%02d/%02d", month, day)
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

	if n < 2 || n > 2 {
		return 0, 0, fmt.Errorf("invalid day format '%s': expected MM/DD (e.g., 12/25)", dayStr)
	}

	if err != nil && n < 2 {
		return 0, 0, fmt.Errorf("invalid day format '%s': expected MM/DD (e.g., 12/25)", dayStr)
	}

	if month < 1 || month > 12 || day < 1 || day > 31 {
		return 0, 0, fmt.Errorf("invalid day values '%s': month must be 1-12, day must be 1-31", dayStr)
	}

	daysInMonth := []int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if day > daysInMonth[month] {
		return 0, 0, fmt.Errorf("invalid day '%d' for month '%d'", day, month)
	}

	return month, day, nil
}

func (tg *TimeGate) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (tg *TimeGate) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (tg *TimeGate) Cleanup(ctx core.SetupContext) error {
	return nil
}
