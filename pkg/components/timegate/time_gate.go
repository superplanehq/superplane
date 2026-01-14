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

type TimeGate struct{}

type Metadata struct {
	NextValidTime *string          `json:"nextValidTime"`
	PushedThrough *PushThroughInfo `json:"pushedThrough,omitempty"`
	CancelledBy   *CancelInfo      `json:"cancelledBy,omitempty"`
}

type PushThroughInfo struct {
	At     string `json:"at"`     // RFC3339 timestamp
	UserID string `json:"userId"` // User ID
	Email  string `json:"email"`  // User email
	Name   string `json:"name"`   // User name
}

type CancelInfo struct {
	At     string `json:"at"`     // RFC3339 timestamp
	UserID string `json:"userId"` // User ID
	Email  string `json:"email"`  // User email
	Name   string `json:"name"`   // User name
}

type TimeGateItem struct {
	Days      []string `json:"days"`      // Days of week (monday, tuesday, etc.)
	StartTime string   `json:"startTime"` // Required
	EndTime   string   `json:"endTime"`   // Required
}

type ExcludeDate struct {
	Date      string `json:"date"`      // MM-DD format
	StartTime string `json:"startTime"` // HH:MM format (currently not used in UI, always defaults to 00:00)
	EndTime   string `json:"endTime"`   // HH:MM format (currently not used in UI, always defaults to 23:59)
}

type Spec struct {
	Items        []TimeGateItem `mapstructure:"items"`         // List of time gate items (all allow mode)
	ExcludeDates []ExcludeDate  `mapstructure:"exclude_dates"` // List of dates to exclude (entire day)
	Timezone     string         `mapstructure:"timezone"`      // Required timezone
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
			Name:        "items",
			Label:       "Time Window",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "Items will wait until the next valid time window is reached",
			Default:     `[{"days":["monday","tuesday","wednesday","thursday","friday"],"startTime":"00:00","endTime":"23:59"}]`,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Time Window",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "days",
								Label:    "Days of Week",
								Type:     configuration.FieldTypeMultiSelect,
								Required: true,
								Default:  []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
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
								Name:        "startTime",
								Label:       "Start Time (HH:MM)",
								Type:        configuration.FieldTypeTime,
								Required:    true,
								Description: "Start time in HH:MM format (24-hour), e.g., 09:30",
								Default:     "00:00",
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
								Default:     "23:59",
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
			Name:        "exclude_dates",
			Label:       "Exclude Dates",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Override the rules above for specific dates like holidays.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "date",
								Label:    "Date",
								Type:     configuration.FieldTypeDate,
								Required: true,
								TypeOptions: &configuration.TypeOptions{
									Date: &configuration.DateTypeOptions{
										Format: "01-02", // MM-DD format for recurring dates
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
	if len(spec.Items) == 0 {
		return fmt.Errorf("at least one time window item is required")
	}

	validDays := map[string]bool{
		"monday": true, "tuesday": true, "wednesday": true, "thursday": true,
		"friday": true, "saturday": true, "sunday": true,
	}

	for i, item := range spec.Items {
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

		if len(item.Days) == 0 {
			return fmt.Errorf("item %d: at least one day must be selected", i)
		}

		for _, day := range item.Days {
			if !validDays[day] {
				return fmt.Errorf("item %d: invalid day '%s': must be one of monday, tuesday, wednesday, thursday, friday, saturday, sunday", i, day)
			}
		}
	}

	// Validate exclude dates
	for i, excludeDate := range spec.ExcludeDates {
		// Validate date format (MM-DD)
		_, err := time.Parse("01-02", excludeDate.Date)
		if err != nil {
			return fmt.Errorf("exclude date %d: invalid date format '%s': must be in MM-DD format (e.g., 12-31)", i, excludeDate.Date)
		}

		// Validate times if provided
		if excludeDate.StartTime != "" {
			startTime, err := parseTimeString(excludeDate.StartTime)
			if err != nil {
				return fmt.Errorf("exclude date %d: startTime error: %w", i, err)
			}

			endTime := startTime + 1 // Default endTime if not provided
			if excludeDate.EndTime != "" {
				endTime, err = parseTimeString(excludeDate.EndTime)
				if err != nil {
					return fmt.Errorf("exclude date %d: endTime error: %w", i, err)
				}
			}

			if startTime >= endTime {
				return fmt.Errorf("exclude date %d: start time (%s) must be before end time (%s)", i, excludeDate.StartTime, excludeDate.EndTime)
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

	// Store push through information in metadata
	user := ctx.Auth.AuthenticatedUser()
	pushThroughInfo := PushThroughInfo{
		At: time.Now().Format(time.RFC3339),
	}
	if user != nil {
		pushThroughInfo.UserID = user.ID
		pushThroughInfo.Email = user.Email
		pushThroughInfo.Name = user.Name
	}

	// Get existing metadata
	existingMetadata := Metadata{}
	existingMetaData := ctx.Metadata.Get()
	if existingMetaData != nil {
		if err := mapstructure.Decode(existingMetaData, &existingMetadata); err != nil {
			// If decode fails, try to read as map
			if metaMap, ok := existingMetaData.(map[string]any); ok {
				if nextValidTime, ok := metaMap["nextValidTime"].(string); ok {
					existingMetadata.NextValidTime = &nextValidTime
				}
			}
		}
	}

	// Set updated metadata with push through info
	existingMetadata.PushedThrough = &pushThroughInfo
	err := ctx.Metadata.Set(existingMetadata)
	if err != nil {
		return fmt.Errorf("failed to store push through metadata: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"timegate.finished",
		[]any{map[string]any{}},
	)
}

func (tg *TimeGate) configEqual(a, b Spec) bool {
	if len(a.Items) != len(b.Items) {
		return false
	}

	// Compare items (order matters for now, but we could make it order-independent if needed)
	for i, itemA := range a.Items {
		if i >= len(b.Items) {
			return false
		}
		itemB := b.Items[i]

		if itemA.StartTime != itemB.StartTime || itemA.EndTime != itemB.EndTime {
			return false
		}

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

	// Compare exclude dates
	if len(a.ExcludeDates) != len(b.ExcludeDates) {
		return false
	}

	aExcludes := make(map[string]ExcludeDate)
	for _, excludeDate := range a.ExcludeDates {
		key := fmt.Sprintf("%s:%s:%s", excludeDate.Date, excludeDate.StartTime, excludeDate.EndTime)
		aExcludes[key] = excludeDate
	}

	for _, excludeDate := range b.ExcludeDates {
		key := fmt.Sprintf("%s:%s:%s", excludeDate.Date, excludeDate.StartTime, excludeDate.EndTime)
		if _, exists := aExcludes[key]; !exists {
			return false
		}
	}

	return true
}

func (tg *TimeGate) findNextValidTime(now time.Time, spec Spec) time.Time {
	// Always use include mode for items, but check exclude dates
	return tg.findNextIncludeTimeWithExcludes(now, spec)
}

func (tg *TimeGate) findNextIncludeTimeWithExcludes(now time.Time, spec Spec) time.Time {
	// Find the earliest next valid time across all items, skipping exclude dates
	var candidates []time.Time

	for _, item := range spec.Items {
		candidate := tg.findNextIncludeWeeklyTimeWithExcludes(now, item, spec.ExcludeDates)
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

func (tg *TimeGate) findNextIncludeWeeklyTimeWithExcludes(now time.Time, item TimeGateItem, excludeDates []ExcludeDate) time.Time {
	startTime, _ := parseTimeString(item.StartTime)
	endTime, _ := parseTimeString(item.EndTime)

	// Check if current time is valid (within window and not excluded)
	currentDay := getDayString(now.Weekday())
	isDayMatch := contains(item.Days, currentDay)
	currentTime := now.Hour()*60 + now.Minute()
	isTimeInWindow := isTimeInRange(currentTime, startTime, endTime)
	isExcluded := tg.isDateExcluded(now, excludeDates)

	if isDayMatch && isTimeInWindow && !isExcluded {
		return now
	}

	// Search up to 30 days ahead to find next valid time
	for i := 0; i < 30; i++ {
		checkDate := now.AddDate(0, 0, i)
		dayString := getDayString(checkDate.Weekday())

		if contains(item.Days, dayString) {
			startHour := startTime / 60
			startMinute := startTime % 60

			candidateTime := time.Date(
				checkDate.Year(), checkDate.Month(), checkDate.Day(),
				startHour, startMinute, 0, 0, now.Location(),
			)

			// Check if the candidate time (with window start time) is excluded
			if tg.isDateExcluded(candidateTime, excludeDates) {
				continue
			}

			if i == 0 && !candidateTime.After(now) {
				continue
			}

			return candidateTime
		}
	}

	return time.Time{}
}

// isDateExcluded checks if a date and time matches any of the exclude dates
func (tg *TimeGate) isDateExcluded(date time.Time, excludeDates []ExcludeDate) bool {
	dateStr := fmt.Sprintf("%02d-%02d", int(date.Month()), date.Day())
	currentTime := date.Hour()*60 + date.Minute()

	for _, excludeDate := range excludeDates {
		if dateStr != excludeDate.Date {
			continue
		}

		// If no startTime is set, it's all day (exclude the entire day)
		if excludeDate.StartTime == "" {
			return true
		}

		// Check if current time is within the excluded time range
		startTime, _ := parseTimeString(excludeDate.StartTime)
		endTime := startTime + 1
		if excludeDate.EndTime != "" {
			endTime, _ = parseTimeString(excludeDate.EndTime)
		}

		if isTimeInRange(currentTime, startTime, endTime) {
			return true
		}
	}
	return false
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

func (tg *TimeGate) Cancel(ctx core.ExecutionContext) error {
	// Store cancellation information in metadata
	user := ctx.Auth.AuthenticatedUser()
	if user != nil {
		cancelInfo := CancelInfo{
			At:     time.Now().Format(time.RFC3339),
			UserID: user.ID,
			Email:  user.Email,
			Name:   user.Name,
		}

		// Get existing metadata
		existingMetadata := Metadata{}
		existingMetaData := ctx.Metadata.Get()
		if existingMetaData != nil {
			if err := mapstructure.Decode(existingMetaData, &existingMetadata); err != nil {
				// If decode fails, try to read as map
				if metaMap, ok := existingMetaData.(map[string]any); ok {
					if nextValidTime, ok := metaMap["nextValidTime"].(string); ok {
						existingMetadata.NextValidTime = &nextValidTime
					}
				}
			}
		}

		// Set updated metadata with cancellation info
		existingMetadata.CancelledBy = &cancelInfo
		err := ctx.Metadata.Set(existingMetadata)
		if err != nil {
			return fmt.Errorf("failed to store cancellation metadata: %w", err)
		}
	}

	return nil
}

func (tg *TimeGate) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
