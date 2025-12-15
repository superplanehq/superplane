package timegate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
)

func TestTimeGate_Name(t *testing.T) {
	tg := &TimeGate{}
	assert.Equal(t, "time_gate", tg.Name())
}

func TestTimeGate_Label(t *testing.T) {
	tg := &TimeGate{}
	assert.Equal(t, "Time Gate", tg.Label())
}

func TestTimeGate_Description(t *testing.T) {
	tg := &TimeGate{}
	expected := "Route events based on time conditions - include or exclude specific time windows"
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
	assert.Len(t, config, 7) // mode, days, startDayInYear, startTime, endDayInYear, endTime, timezone

	// Check mode field
	assert.Equal(t, "mode", config[0].Name)
	assert.Equal(t, "Mode", config[0].Label)
	assert.True(t, config[0].Required)

	// Check days field
	assert.Equal(t, "days", config[1].Name)
	assert.Equal(t, "Days of Week", config[1].Label)
	assert.False(t, config[1].Required) // Not required because of visibility conditions

	// Check startDayInYear field
	assert.Equal(t, "startDayInYear", config[2].Name)
	assert.Equal(t, "Start Day (MM/DD)", config[2].Label)
	assert.False(t, config[2].Required) // Not required because of visibility conditions

	// Check startTime field
	assert.Equal(t, "startTime", config[3].Name)
	assert.Equal(t, "Start Time (HH:MM)", config[3].Label)
	assert.True(t, config[3].Required) // Required for all modes
	assert.Equal(t, "09:00", config[3].Default)

	// Check endDayInYear field
	assert.Equal(t, "endDayInYear", config[4].Name)
	assert.Equal(t, "End Day (MM/DD)", config[4].Label)
	assert.False(t, config[4].Required) // Not required because of visibility conditions

	// Check endTime field
	assert.Equal(t, "endTime", config[5].Name)
	assert.Equal(t, "End Time (HH:MM)", config[5].Label)
	assert.True(t, config[5].Required) // Required for all modes
	assert.Equal(t, "17:00", config[5].Default)
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

// High-signal tests for action handling behavior
type actionMockExecutionStateContext struct {
	finished bool
	passed   bool
	failed   bool
}

func (m *actionMockExecutionStateContext) SetKV(key, value string) error {
	return nil
}

func (m *actionMockExecutionStateContext) IsFinished() bool { return m.finished }
func (m *actionMockExecutionStateContext) Pass(outputs map[string][]any) error {
	m.passed = true
	m.finished = true
	return nil
}
func (m *actionMockExecutionStateContext) Fail(reason, message string) error {
	m.failed = true
	m.finished = true
	return nil
}

func TestTimeGate_HandleAction_PushThrough_Finishes(t *testing.T) {
	tg := &TimeGate{}

	mockState := &actionMockExecutionStateContext{}
	ctx := core.ActionContext{
		Name:                  "pushThrough",
		ExecutionStateContext: mockState,
		Parameters:            map[string]any{},
	}

	err := tg.HandleAction(ctx)
	assert.NoError(t, err)
	assert.True(t, mockState.passed)
	assert.True(t, mockState.finished)
}

func TestParseTimeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		{"valid morning time", "09:30", 570, false},   // 9*60 + 30
		{"valid afternoon time", "14:45", 885, false}, // 14*60 + 45
		{"midnight", "00:00", 0, false},
		{"end of day", "23:59", 1439, false}, // 23*60 + 59
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

func TestIsTimeInRange(t *testing.T) {
	tests := []struct {
		name        string
		currentTime int
		startTime   int
		endTime     int
		expected    bool
	}{
		{"within normal range", 600, 540, 720, true},                // 10:00 between 09:00-12:00
		{"at start time", 540, 540, 720, true},                      // 09:00 at 09:00-12:00
		{"at end time", 720, 540, 720, true},                        // 12:00 at 09:00-12:00
		{"before range", 480, 540, 720, false},                      // 08:00 before 09:00-12:00
		{"after range", 780, 540, 720, false},                       // 13:00 after 09:00-12:00
		{"overnight range - in first part", 60, 1320, 120, true},    // 01:00 in 22:00-02:00
		{"overnight range - in second part", 1380, 1320, 120, true}, // 23:00 in 22:00-02:00
		{"overnight range - outside", 600, 1320, 120, false},        // 10:00 outside 22:00-02:00
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTimeInRange(tt.currentTime, tt.startTime, tt.endTime)
			assert.Equal(t, tt.expected, result)
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

func TestContains(t *testing.T) {
	slice := []string{"monday", "tuesday", "wednesday"}

	tests := []struct {
		item     string
		expected bool
	}{
		{"monday", true},
		{"tuesday", true},
		{"wednesday", true},
		{"thursday", false},
		{"friday", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.item, func(t *testing.T) {
			result := contains(slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateSpec(t *testing.T) {
	tg := &TimeGate{}

	tests := []struct {
		name     string
		spec     Spec
		hasError bool
		errorMsg string
	}{
		{
			name: "valid include range spec",
			spec: Spec{
				Mode:      "include_range",
				StartTime: "09:00",
				EndTime:   "17:00",
				Days:      []string{"monday", "tuesday"},
			},
			hasError: false,
		},
		{
			name: "valid exclude range spec",
			spec: Spec{
				Mode:      "exclude_range",
				StartTime: "13:00",
				EndTime:   "14:00",
				Days:      []string{"friday"},
			},
			hasError: false,
		},
		{
			name: "valid include specific spec",
			spec: Spec{
				Mode:           "include_specific",
				StartTime:      "00:00",
				EndTime:        "23:59",
				StartDayInYear: "12/31",
				EndDayInYear:   "01/01",
			},
			hasError: false,
		},
		{
			name: "valid exclude specific spec",
			spec: Spec{
				Mode:           "exclude_specific",
				StartTime:      "12:00",
				EndTime:        "13:00",
				StartDayInYear: "07/04",
				EndDayInYear:   "07/04",
			},
			hasError: false,
		},
		{
			name: "invalid mode",
			spec: Spec{
				Mode:      "invalid",
				StartTime: "09:00",
				EndTime:   "17:00",
				Days:      []string{"monday"},
			},
			hasError: true,
			errorMsg: "invalid mode",
		},
		{
			name: "missing days for specific mode",
			spec: Spec{
				Mode:      "include_specific",
				StartTime: "09:00",
				EndTime:   "17:00",
			},
			hasError: true,
			errorMsg: "startDayInYear and endDayInYear are required",
		},
		{
			name: "invalid start day format",
			spec: Spec{
				Mode:           "include_specific",
				StartTime:      "09:00",
				EndTime:        "17:00",
				StartDayInYear: "invalid-day",
				EndDayInYear:   "01/01",
			},
			hasError: true,
			errorMsg: "startDayInYear error",
		},
		{
			name: "start day after end day in same month",
			spec: Spec{
				Mode:           "include_specific",
				StartTime:      "10:00",
				EndTime:        "11:00",
				StartDayInYear: "01/15",
				EndDayInYear:   "01/10",
			},
			hasError: true,
			errorMsg: "start day",
		},
		{
			name: "invalid start time",
			spec: Spec{
				Mode:      "include_range",
				StartTime: "25:00",
				EndTime:   "17:00",
				Days:      []string{"monday"},
			},
			hasError: true,
			errorMsg: "startTime error",
		},
		{
			name: "invalid end time",
			spec: Spec{
				Mode:      "include_range",
				StartTime: "09:00",
				EndTime:   "25:00",
				Days:      []string{"monday"},
			},
			hasError: true,
			errorMsg: "endTime error",
		},
		{
			name: "start time after end time",
			spec: Spec{
				Mode:      "include_range",
				StartTime: "17:00",
				EndTime:   "09:00",
				Days:      []string{"monday"},
			},
			hasError: true,
			errorMsg: "start time (17:00) must be before end time (09:00)",
		},
		{
			name: "start time equals end time",
			spec: Spec{
				Mode:      "include_range",
				StartTime: "12:00",
				EndTime:   "12:00",
				Days:      []string{"monday"},
			},
			hasError: true,
			errorMsg: "start time (12:00) must be before end time (12:00)",
		},
		{
			name: "no days selected",
			spec: Spec{
				Mode:      "include_range",
				StartTime: "09:00",
				EndTime:   "17:00",
				Days:      []string{},
			},
			hasError: true,
			errorMsg: "at least one day must be selected",
		},
		{
			name: "invalid day",
			spec: Spec{
				Mode:      "include_range",
				StartTime: "09:00",
				EndTime:   "17:00",
				Days:      []string{"invalid_day"},
			},
			hasError: true,
			errorMsg: "invalid day 'invalid_day'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tg.validateSpec(tt.spec)
			if tt.hasError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigEqual(t *testing.T) {
	tg := &TimeGate{}

	baseSpec := Spec{
		Mode:      "include_range",
		StartTime: "09:00",
		EndTime:   "17:00",
		Days:      []string{"monday", "tuesday"},
	}

	tests := []struct {
		name     string
		specA    Spec
		specB    Spec
		expected bool
	}{
		{
			name:     "identical specs",
			specA:    baseSpec,
			specB:    baseSpec,
			expected: true,
		},
		{
			name:  "same content different order days",
			specA: baseSpec,
			specB: Spec{
				Mode:      "include_range",
				StartTime: "09:00",
				EndTime:   "17:00",
				Days:      []string{"tuesday", "monday"}, // different order
			},
			expected: true,
		},
		{
			name:  "different mode",
			specA: baseSpec,
			specB: Spec{
				Mode:      "exclude_range", // different mode
				StartTime: "09:00",
				EndTime:   "17:00",
				Days:      []string{"monday", "tuesday"},
			},
			expected: false,
		},
		{
			name:  "different start time",
			specA: baseSpec,
			specB: Spec{
				Mode:      "include_range",
				StartTime: "10:00", // different start time
				EndTime:   "17:00",
				Days:      []string{"monday", "tuesday"},
			},
			expected: false,
		},
		{
			name:  "different end time",
			specA: baseSpec,
			specB: Spec{
				Mode:      "include_range",
				StartTime: "09:00",
				EndTime:   "18:00", // different end time
				Days:      []string{"monday", "tuesday"},
			},
			expected: false,
		},
		{
			name:  "different days",
			specA: baseSpec,
			specB: Spec{
				Mode:      "include_range",
				StartTime: "09:00",
				EndTime:   "17:00",
				Days:      []string{"monday", "wednesday"}, // different days
			},
			expected: false,
		},
		{
			name:  "different number of days",
			specA: baseSpec,
			specB: Spec{
				Mode:      "include_range",
				StartTime: "09:00",
				EndTime:   "17:00",
				Days:      []string{"monday"}, // fewer days
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tg.configEqual(tt.specA, tt.specB)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindNextIncludeTime(t *testing.T) {
	tg := &TimeGate{}

	// Test on a Tuesday at 10:00 UTC
	testTime := time.Date(2024, 11, 5, 10, 0, 0, 0, time.UTC) // Tuesday

	tests := []struct {
		name             string
		spec             Spec
		expectedIsNow    bool
		expectedIsFuture bool
		description      string
	}{
		{
			name: "currently in include window",
			spec: Spec{
				Mode:      "include_range",
				StartTime: "09:00",
				EndTime:   "17:00",
				Days:      []string{"tuesday"},
			},
			expectedIsNow: true,
			description:   "Should return now since we're in the window",
		},
		{
			name: "outside time window, same day",
			spec: Spec{
				Mode:      "include_range",
				StartTime: "20:00",
				EndTime:   "22:00",
				Days:      []string{"tuesday"},
			},
			expectedIsFuture: true,
			description:      "Should return future time today",
		},
		{
			name: "outside day window",
			spec: Spec{
				Mode:      "include_range",
				StartTime: "09:00",
				EndTime:   "17:00",
				Days:      []string{"monday"},
			},
			expectedIsFuture: true,
			description:      "Should return next Monday",
		},
		{
			name: "before time window, same day",
			spec: Spec{
				Mode:      "include_range",
				StartTime: "11:00",
				EndTime:   "17:00",
				Days:      []string{"tuesday"},
			},
			expectedIsFuture: true,
			description:      "Should return 11:00 today",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tg.findNextIncludeTime(testTime, tt.spec)

			if tt.expectedIsNow {
				assert.Equal(t, testTime, result, tt.description)
			} else if tt.expectedIsFuture {
				assert.True(t, result.After(testTime), tt.description)
				assert.False(t, result.IsZero(), "Should not return zero time")
			}
		})
	}
}

func TestFindNextExcludeEndTime(t *testing.T) {
	tg := &TimeGate{}

	// Test on a Tuesday at 14:00 UTC (inside a typical exclude window)
	testTime := time.Date(2024, 11, 5, 14, 0, 0, 0, time.UTC) // Tuesday 2 PM

	tests := []struct {
		name             string
		spec             Spec
		expectedIsNow    bool
		expectedIsFuture bool
		description      string
	}{
		{
			name: "outside exclude window",
			spec: Spec{
				Mode:      "exclude_range",
				StartTime: "09:00",
				EndTime:   "12:00",
				Days:      []string{"tuesday"},
			},
			expectedIsNow: true,
			description:   "Should return now since we're outside exclude window",
		},
		{
			name: "inside exclude window",
			spec: Spec{
				Mode:      "exclude_range",
				StartTime: "13:00",
				EndTime:   "17:00",
				Days:      []string{"tuesday"},
			},
			expectedIsFuture: true,
			description:      "Should return end of exclude window",
		},
		{
			name: "exclude window on different day",
			spec: Spec{
				Mode:      "exclude_range",
				StartTime: "13:00",
				EndTime:   "17:00",
				Days:      []string{"monday"},
			},
			expectedIsNow: true,
			description:   "Should return now since exclude is on different day",
		},
		{
			name: "past exclude window end time",
			spec: Spec{
				Mode:      "exclude_range",
				StartTime: "09:00",
				EndTime:   "13:00",
				Days:      []string{"tuesday"},
			},
			expectedIsNow: true,
			description:   "Should return now since exclude window has ended",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tg.findNextExcludeEndTime(testTime, tt.spec)

			if tt.expectedIsNow {
				assert.Equal(t, testTime, result, tt.description)
			} else if tt.expectedIsFuture {
				assert.True(t, result.After(testTime), tt.description)
				assert.False(t, result.IsZero(), "Should not return zero time")
			}
		})
	}
}

func TestParseDayInYear(t *testing.T) {
	tg := &TimeGate{}

	tests := []struct {
		name          string
		input         string
		expectedMonth int
		expectedDay   int
		hasError      bool
	}{
		{"valid Christmas", "12/25", 12, 25, false},
		{"valid New Year", "01/01", 1, 1, false},
		{"valid leap day", "02/29", 2, 29, false},
		{"valid July 4th", "07/04", 7, 4, false},
		{"single digit month and day", "1/1", 1, 1, false},
		{"empty string", "", 0, 0, true},
		{"invalid format", "abc", 0, 0, true},
		{"invalid month", "13/01", 0, 0, true},
		{"invalid day", "01/32", 0, 0, true},
		{"negative month", "-1/15", 0, 0, true},
		{"negative day", "06/-5", 0, 0, true},
		{"missing slash", "1225", 0, 0, true},
		{"extra parts", "12/25/2024", 0, 0, true},
		{"zero month", "00/15", 0, 0, true},
		{"zero day", "06/00", 0, 0, true},
		{"invalid day for February", "02/30", 0, 0, true},
		{"invalid day for April", "04/31", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			month, day, err := tg.parseDayInYear(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMonth, month)
				assert.Equal(t, tt.expectedDay, day)
			}
		})
	}
}

func TestValidateDayInYear(t *testing.T) {
	tg := &TimeGate{}

	tests := []struct {
		name     string
		input    string
		hasError bool
	}{
		{"valid Christmas", "12/25", false},
		{"valid New Year", "01/01", false},
		{"invalid format", "invalid", true},
		{"invalid month", "13/01", true},
		{"invalid day", "01/32", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tg.validateDayInYear(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseTimezone(t *testing.T) {
	tg := &TimeGate{}

	tests := []struct {
		name           string
		timezoneStr    string
		expectedOffset int // offset in seconds
	}{
		{
			name:           "UTC",
			timezoneStr:    "0",
			expectedOffset: 0,
		},
		{
			name:           "EST (GMT-5)",
			timezoneStr:    "-5",
			expectedOffset: -5 * 3600,
		},
		{
			name:           "JST (GMT+9)",
			timezoneStr:    "9",
			expectedOffset: 9 * 3600,
		},
		{
			name:           "India (GMT+5.5)",
			timezoneStr:    "5.5",
			expectedOffset: int(5.5 * 3600),
		},
		{
			name:           "Empty string defaults to UTC",
			timezoneStr:    "",
			expectedOffset: 0,
		},
		{
			name:           "Invalid string defaults to UTC",
			timezoneStr:    "invalid",
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			location := tg.parseTimezone(tt.timezoneStr)

			// Test with a known time to verify offset
			testTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
			localTime := testTime.In(location)

			_, offset := localTime.Zone()
			assert.Equal(t, tt.expectedOffset, offset)
		})
	}
}

func TestFindNextValidTime(t *testing.T) {
	tg := &TimeGate{}

	// Test on a Tuesday at 10:00 UTC
	testTime := time.Date(2024, 11, 5, 10, 0, 0, 0, time.UTC) // Tuesday

	tests := []struct {
		name         string
		spec         Spec
		expectNow    bool
		expectFuture bool
		description  string
	}{
		{
			name: "include mode - in window",
			spec: Spec{
				Mode:      "include_range",
				StartTime: "09:00",
				EndTime:   "17:00",
				Days:      []string{"tuesday"},
			},
			expectNow:   true,
			description: "Include mode, currently in window, should return now",
		},
		{
			name: "include mode - out of window",
			spec: Spec{
				Mode:      "include_range",
				StartTime: "20:00",
				EndTime:   "22:00",
				Days:      []string{"tuesday"},
			},
			expectFuture: true,
			description:  "Include mode, out of window, should return future time",
		},
		{
			name: "exclude mode - outside exclude window",
			spec: Spec{
				Mode:      "exclude_range",
				StartTime: "13:00",
				EndTime:   "17:00",
				Days:      []string{"tuesday"},
			},
			expectNow:   true,
			description: "Exclude mode, outside exclude window, should return now",
		},
		{
			name: "exclude mode - inside exclude window",
			spec: Spec{
				Mode:      "exclude_range",
				StartTime: "09:00",
				EndTime:   "17:00",
				Days:      []string{"tuesday"},
			},
			expectFuture: true,
			description:  "Exclude mode, inside exclude window, should return end time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tg.findNextValidTime(testTime, tt.spec)

			if tt.expectNow {
				assert.Equal(t, testTime, result, tt.description)
			} else if tt.expectFuture {
				assert.True(t, result.After(testTime), tt.description)
				assert.False(t, result.IsZero(), "Should not return zero time")
			}
		})
	}
}
