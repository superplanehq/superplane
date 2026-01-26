package timegate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestTimeGate_Name(t *testing.T) {
	tg := &TimeGate{}
	assert.Equal(t, "timeGate", tg.Name())
}

func TestTimeGate_Label(t *testing.T) {
	tg := &TimeGate{}
	assert.Equal(t, "Time Gate", tg.Label())
}

func TestTimeGate_Description(t *testing.T) {
	tg := &TimeGate{}
	expected := "Route events based on active days and time windows, with optional excluded dates"
	assert.Equal(t, expected, tg.Description())
}

func TestTimeGate_Icon(t *testing.T) {
	tg := &TimeGate{}
	assert.Equal(t, "clock", tg.Icon())
}

func TestTimeGate_Color(t *testing.T) {
	tg := &TimeGate{}
	assert.Equal(t, "blue", tg.Color())
}

func TestTimeGate_OutputChannels(t *testing.T) {
	tg := &TimeGate{}
	channels := tg.OutputChannels(nil)
	assert.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func TestTimeGate_Configuration(t *testing.T) {
	tg := &TimeGate{}
	config := tg.Configuration()
	assert.Len(t, config, 4)

	assert.Equal(t, "days", config[0].Name)
	assert.Equal(t, "Active Days", config[0].Label)
	assert.True(t, config[0].Required)

	assert.Equal(t, "timeRange", config[1].Name)
	assert.Equal(t, "Active Time", config[1].Label)
	assert.True(t, config[1].Required)
	assert.Equal(t, "00:00-23:59", config[1].Default)

	assert.Equal(t, "timezone", config[2].Name)
	assert.Equal(t, "Timezone", config[2].Label)
	assert.True(t, config[2].Required)
	assert.Equal(t, "current", config[2].Default)

	assert.Equal(t, "excludeDates", config[3].Name)
	assert.Equal(t, "Exclude Dates (MM/DD)", config[3].Label)
}

func TestTimeGate_Actions(t *testing.T) {
	tg := &TimeGate{}
	actions := tg.Actions()

	if len(actions) < 2 {
		t.Fatalf("expected at least 2 actions, got %d: %+v", len(actions), actions)
	}

	var hasTimeReached, hasPushThrough bool
	for _, a := range actions {
		if a.Name == "timeReached" {
			hasTimeReached = true
		}
		if a.Name == "pushThrough" {
			hasPushThrough = true
		}
	}

	assert.True(t, hasTimeReached)
	assert.True(t, hasPushThrough)
}

func TestTimeGate_HandleAction_PushThrough_Finishes(t *testing.T) {
	tg := &TimeGate{}

	stateCtx := &contexts.ExecutionStateContext{}
	ctx := core.ActionContext{
		Name:           "pushThrough",
		ExecutionState: stateCtx,
		Parameters:     map[string]any{},
		Metadata:       &contexts.MetadataContext{},
		Auth: &contexts.AuthContext{
			User: &core.User{
				ID:    "123",
				Name:  "Test User",
				Email: "test@example.com",
			},
		},
	}

	err := tg.HandleAction(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)
}

func TestTimeGate_ValidateSpec_DuplicateExcludeDates(t *testing.T) {
	tg := &TimeGate{}
	now := time.Now().UTC()
	monthDay := formatDayKey(int(now.Month()), now.Day())

	spec := Spec{
		Days:         []string{"monday"},
		TimeRange:    "09:00-17:00",
		Timezone:     "0",
		ExcludeDates: []string{monthDay, monthDay},
	}

	err := tg.validateSpec(spec)
	assert.Error(t, err)
}

func TestTimeGate_FindNextValidTime_WithinWindow(t *testing.T) {
	tg := &TimeGate{}
	base := time.Now().UTC()
	now := timeOnDate(base, 0, 10, 0)

	spec := Spec{
		Days:      []string{getDayString(now.Weekday())},
		TimeRange: "09:00-17:00",
		Timezone:  "0",
	}

	startMinutes, endMinutes, err := parseTimeRangeString(spec.TimeRange)
	assert.NoError(t, err)

	next := tg.findNextValidTime(now, spec, startMinutes, endMinutes)
	assert.Equal(t, now, next)
}

func TestTimeGate_FindNextValidTime_BeforeWindow(t *testing.T) {
	tg := &TimeGate{}
	base := time.Now().UTC()
	now := timeOnDate(base, 0, 8, 0)
	expected := timeOnDate(base, 0, 9, 0)

	spec := Spec{
		Days:      []string{getDayString(now.Weekday())},
		TimeRange: "09:00-17:00",
		Timezone:  "0",
	}

	startMinutes, endMinutes, err := parseTimeRangeString(spec.TimeRange)
	assert.NoError(t, err)

	next := tg.findNextValidTime(now, spec, startMinutes, endMinutes)
	assert.Equal(t, expected, next)
}

func TestTimeGate_FindNextValidTime_AfterWindow(t *testing.T) {
	tg := &TimeGate{}
	base := time.Now().UTC()
	now := timeOnDate(base, 0, 18, 0)
	expected := timeOnDate(base, 1, 9, 0)

	spec := Spec{
		Days:      []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"},
		TimeRange: "09:00-17:00",
		Timezone:  "0",
	}

	startMinutes, endMinutes, err := parseTimeRangeString(spec.TimeRange)
	assert.NoError(t, err)

	next := tg.findNextValidTime(now, spec, startMinutes, endMinutes)
	assert.Equal(t, expected, next)
}

func TestTimeGate_FindNextValidTime_ExcludedDate(t *testing.T) {
	tg := &TimeGate{}
	base := time.Now().UTC()
	now := timeOnDate(base, 0, 10, 0)
	excluded := formatDayKey(int(now.Month()), now.Day())
	expected := timeOnDate(base, 1, 9, 0)

	spec := Spec{
		Days:         []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"},
		TimeRange:    "09:00-17:00",
		Timezone:     "0",
		ExcludeDates: []string{excluded},
	}

	startMinutes, endMinutes, err := parseTimeRangeString(spec.TimeRange)
	assert.NoError(t, err)

	next := tg.findNextValidTime(now, spec, startMinutes, endMinutes)
	assert.Equal(t, expected, next)
}

func TestParseTimeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		{"valid morning time", "09:30", 570, false},
		{"valid afternoon time", "14:45", 885, false},
		{"midnight", "00:00", 0, false},
		{"end of day", "23:59", 1439, false},
		{"single digit hour", "9:30", 570, false},
		{"single digit minute", "09:5", 545, false},
		{"empty string", "", 0, true},
		{"invalid format", "abc", 0, true},
		{"invalid hour", "25:30", 0, true},
		{"invalid minute", "09:70", 0, true},
		{"negative hour", "-1:30", 0, true},
		{"negative minute", "09:-5", 0, true},
		{"missing colon", "0930", 0, true},
		{"extra parts", "09:30:00", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTimeString(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetDayString(t *testing.T) {
	tests := []struct {
		weekday  time.Weekday
		expected string
	}{
		{time.Sunday, "sunday"},
		{time.Monday, "monday"},
		{time.Tuesday, "tuesday"},
		{time.Wednesday, "wednesday"},
		{time.Thursday, "thursday"},
		{time.Friday, "friday"},
		{time.Saturday, "saturday"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := getDayString(tt.weekday)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func timeOnDate(base time.Time, dayOffset int, hour int, minute int) time.Time {
	date := base.AddDate(0, 0, dayOffset)
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, base.Location())
}
