package timegate

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestTimeGate_HandleHook_PushThrough_Finishes(t *testing.T) {
	tg := &TimeGate{}

	stateCtx := &contexts.ExecutionStateContext{}
	ctx := core.ActionHookContext{
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

	err := tg.HandleHook(ctx)
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

// deferringSpec builds a configuration whose only active day is tomorrow, so
// the gate always defers regardless of the wall-clock time the test runs at.
func deferringSpec(base time.Time) map[string]any {
	return map[string]any{
		"days":      []string{getDayString(base.AddDate(0, 0, 1).Weekday())},
		"timeRange": "09:00-17:00",
		"timezone":  "0",
	}
}

func TestTimeGate_TimeReached_ForwardsGatedEvent(t *testing.T) {
	tg := &TimeGate{}
	data := map[string]any{"deployment": map[string]any{"sha": "abc123"}}

	metadata := &contexts.MetadataContext{}
	state := &contexts.ExecutionStateContext{}

	err := tg.Execute(core.ExecutionContext{
		Data:           data,
		Configuration:  deferringSpec(time.Now().UTC()),
		Metadata:       metadata,
		Requests:       &contexts.RequestContext{},
		ExecutionState: state,
	})
	assert.NoError(t, err)
	assert.False(t, state.Finished, "gate must defer, not emit immediately")

	err = tg.HandleHook(core.ActionHookContext{
		Name:           "timeReached",
		ExecutionState: state,
		Metadata:       metadata,
	})
	assert.NoError(t, err)
	assert.True(t, state.Finished)
	assert.Len(t, state.Payloads, 1)

	payload := state.Payloads[0].(map[string]any)
	assert.Equal(t, data, payload["data"])
}

func TestTimeGate_PushThrough_ForwardsGatedEvent(t *testing.T) {
	tg := &TimeGate{}
	data := map[string]any{"deployment": map[string]any{"sha": "abc123"}}

	metadata := &contexts.MetadataContext{}
	state := &contexts.ExecutionStateContext{}

	err := tg.Execute(core.ExecutionContext{
		Data:           data,
		Configuration:  deferringSpec(time.Now().UTC()),
		Metadata:       metadata,
		Requests:       &contexts.RequestContext{},
		ExecutionState: state,
	})
	assert.NoError(t, err)
	assert.False(t, state.Finished, "gate must defer, not emit immediately")

	err = tg.HandleHook(core.ActionHookContext{
		Name:           "pushThrough",
		ExecutionState: state,
		Metadata:       metadata,
		Auth: &contexts.AuthContext{
			User: &core.User{ID: "123", Name: "Test User", Email: "test@example.com"},
		},
	})
	assert.NoError(t, err)
	assert.True(t, state.Finished)
	assert.Len(t, state.Payloads, 1)

	payload := state.Payloads[0].(map[string]any)
	assert.Equal(t, data, payload["data"])
}

// jsonMetadataContext mimics the production metadata context, which round-trips
// metadata through JSON before handing it back to the component.
type jsonMetadataContext struct {
	stored map[string]any
}

func (m *jsonMetadataContext) Get() any {
	return m.stored
}

func (m *jsonMetadataContext) Set(metadata any) error {
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	return json.Unmarshal(encoded, &m.stored)
}

func TestTimeGate_ForwardsGatedEventStoredAsJSON(t *testing.T) {
	tg := &TimeGate{}
	data := map[string]any{
		"repository": "superplanehq/superplane",
		"commits":    []any{map[string]any{"sha": "abc123"}},
	}

	metadata := &jsonMetadataContext{}
	state := &contexts.ExecutionStateContext{}

	err := tg.Execute(core.ExecutionContext{
		Data:           data,
		Configuration:  deferringSpec(time.Now().UTC()),
		Metadata:       metadata,
		Requests:       &contexts.RequestContext{},
		ExecutionState: state,
	})
	assert.NoError(t, err)
	assert.False(t, state.Finished, "gate must defer, not emit immediately")

	err = tg.HandleHook(core.ActionHookContext{
		Name:           "timeReached",
		ExecutionState: state,
		Metadata:       metadata,
	})
	assert.NoError(t, err)
	assert.Len(t, state.Payloads, 1)

	payload := state.Payloads[0].(map[string]any)
	assert.Equal(t, data, payload["data"])
}

func TestTimeGate_EmitsEmptyObjectWhenNoEventStored(t *testing.T) {
	tg := &TimeGate{}
	state := &contexts.ExecutionStateContext{}

	err := tg.HandleHook(core.ActionHookContext{
		Name:           "timeReached",
		ExecutionState: state,
		Metadata:       &contexts.MetadataContext{},
	})
	assert.NoError(t, err)
	assert.Len(t, state.Payloads, 1)

	payload := state.Payloads[0].(map[string]any)
	assert.Equal(t, map[string]any{}, payload["data"])
}
