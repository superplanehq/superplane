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
	assert.Len(t, config, 3) // mode, items, timezone

	// Check mode field
	assert.Equal(t, "mode", config[0].Name)
	assert.Equal(t, "Mode", config[0].Label)
	assert.True(t, config[0].Required)

	// Check items field (list)
	assert.Equal(t, "items", config[1].Name)
	assert.Equal(t, "Time Window", config[1].Label)
	assert.True(t, config[1].Required)
	assert.NotNil(t, config[1].TypeOptions)
	assert.NotNil(t, config[1].TypeOptions.List)
	assert.NotNil(t, config[1].TypeOptions.List.ItemDefinition)
	assert.Equal(t, "object", config[1].TypeOptions.List.ItemDefinition.Type)

	// Check timezone field
	assert.Equal(t, "timezone", config[2].Name)
	assert.Equal(t, "Timezone", config[2].Label)
	assert.True(t, config[2].Required)
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
	}

	err := tg.HandleAction(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)
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
			name: "valid weekly spec",
			spec: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "09:00",
						EndTime:   "17:00",
						Days:      []string{"monday", "tuesday"},
					},
				},
				Timezone: "0",
			},
			hasError: false,
		},
		{
			name: "valid weekly spec with friday",
			spec: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "13:00",
						EndTime:   "14:00",
						Days:      []string{"friday"},
					},
				},
				Timezone: "0",
			},
			hasError: false,
		},
		{
			name: "valid spec with exclude dates",
			spec: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "00:00",
						EndTime:   "23:59",
						Days:      []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
					},
				},
				ExcludeDates: []ExcludeDate{
					{
						Date:      "12-31",
						StartTime: "00:00",
						EndTime:   "23:59",
					},
				},
				Timezone: "0",
			},
			hasError: false,
		},
		{
			name: "no items",
			spec: Spec{
				Items:    []TimeGateItem{},
				Timezone: "0",
			},
			hasError: true,
			errorMsg: "at least one time window item is required",
		},
		{
			name: "invalid start time",
			spec: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "25:00",
						EndTime:   "17:00",
						Days:      []string{"monday"},
					},
				},
				Timezone: "0",
			},
			hasError: true,
			errorMsg: "startTime error",
		},
		{
			name: "invalid end time",
			spec: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "09:00",
						EndTime:   "25:00",
						Days:      []string{"monday"},
					},
				},
				Timezone: "0",
			},
			hasError: true,
			errorMsg: "endTime error",
		},
		{
			name: "start time after end time",
			spec: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "17:00",
						EndTime:   "09:00",
						Days:      []string{"monday"},
					},
				},
				Timezone: "0",
			},
			hasError: true,
			errorMsg: "start time (17:00) must be before end time (09:00)",
		},
		{
			name: "start time equals end time",
			spec: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "12:00",
						EndTime:   "12:00",
						Days:      []string{"monday"},
					},
				},
				Timezone: "0",
			},
			hasError: true,
			errorMsg: "start time (12:00) must be before end time (12:00)",
		},
		{
			name: "no days selected",
			spec: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "09:00",
						EndTime:   "17:00",
						Days:      []string{},
					},
				},
				Timezone: "0",
			},
			hasError: true,
			errorMsg: "at least one day must be selected",
		},
		{
			name: "invalid day",
			spec: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "09:00",
						EndTime:   "17:00",
						Days:      []string{"invalid_day"},
					},
				},
				Timezone: "0",
			},
			hasError: true,
			errorMsg: "invalid day 'invalid_day'",
		},
		{
			name: "empty timezone",
			spec: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "09:00",
						EndTime:   "17:00",
						Days:      []string{"monday"},
					},
				},
				Timezone: "",
			},
			hasError: true,
			errorMsg: "timezone is required",
		},
		{
			name: "invalid timezone",
			spec: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "09:00",
						EndTime:   "17:00",
						Days:      []string{"monday"},
					},
				},
				Timezone: "invalid",
			},
			hasError: true,
			errorMsg: "invalid timezone 'invalid'",
		},
		{
			name: "invalid timezone offset",
			spec: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "09:00",
						EndTime:   "17:00",
						Days:      []string{"monday"},
					},
				},
				Timezone: "15",
			},
			hasError: true,
			errorMsg: "invalid timezone '15'",
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
		Items: []TimeGateItem{
			{
				StartTime: "09:00",
				EndTime:   "17:00",
				Days:      []string{"monday", "tuesday"},
			},
		},
		Timezone: "0",
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
				Items: []TimeGateItem{
					{
						StartTime: "09:00",
						EndTime:   "17:00",
						Days:      []string{"tuesday", "monday"}, // different order
					},
				},
				Timezone: "0",
			},
			expected: true,
		},
		{
			name:  "different start time",
			specA: baseSpec,
			specB: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "10:00", // different start time
						EndTime:   "17:00",
						Days:      []string{"monday", "tuesday"},
					},
				},
				Timezone: "0",
			},
			expected: false,
		},
		{
			name:  "different end time",
			specA: baseSpec,
			specB: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "09:00",
						EndTime:   "18:00", // different end time
						Days:      []string{"monday", "tuesday"},
					},
				},
				Timezone: "0",
			},
			expected: false,
		},
		{
			name:  "different days",
			specA: baseSpec,
			specB: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "09:00",
						EndTime:   "17:00",
						Days:      []string{"monday", "wednesday"}, // different days
					},
				},
				Timezone: "0",
			},
			expected: false,
		},
		{
			name:  "different number of items",
			specA: baseSpec,
			specB: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "09:00",
						EndTime:   "17:00",
						Days:      []string{"monday", "tuesday"},
					},
					{
						StartTime: "09:00",
						EndTime:   "17:00",
						Days:      []string{"wednesday"},
					},
				},
				Timezone: "0",
			},
			expected: false,
		},
		{
			name:  "duplicate days in B should not equal unique days in A",
			specA: baseSpec,
			specB: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "09:00",
						EndTime:   "17:00",
						Days:      []string{"monday", "monday"}, // duplicate, but same length
					},
				},
				Timezone: "0",
			},
			expected: false,
		},
		{
			name: "duplicate exclude dates in B should not equal unique exclude dates in A",
			specA: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "09:00",
						EndTime:   "17:00",
						Days:      []string{"monday", "tuesday"},
					},
				},
				ExcludeDates: []ExcludeDate{
					{Date: "12-25", StartTime: "", EndTime: ""},
					{Date: "12-26", StartTime: "", EndTime: ""},
				},
				Timezone: "0",
			},
			specB: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "09:00",
						EndTime:   "17:00",
						Days:      []string{"monday", "tuesday"},
					},
				},
				ExcludeDates: []ExcludeDate{
					{Date: "12-25", StartTime: "", EndTime: ""},
					{Date: "12-25", StartTime: "", EndTime: ""}, // duplicate
				},
				Timezone: "0",
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

// TestFindNextIncludeTime - REMOVED: This method no longer exists
/*
func TestFindNextIncludeTime(t *testing.T) {
	// Test removed - findNextIncludeTime method no longer exists
}
*/

// TestFindNextExcludeEndTime - REMOVED: This method no longer exists
/*
func TestFindNextExcludeEndTime(t *testing.T) {
	// Test removed - findNextExcludeEndTime method no longer exists
}
*/

// TestParseDayInYear - REMOVED: This method no longer exists
/*
func TestParseDayInYear(t *testing.T) {
	// Test removed - parseDayInYear method no longer exists
}
*/

// TestValidateDayInYear - REMOVED: This method no longer exists
/*
func TestValidateDayInYear(t *testing.T) {
	// Test removed - validateDayInYear method no longer exists
}
*/

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
			name: "in window",
			spec: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "09:00",
						EndTime:   "17:00",
						Days:      []string{"tuesday"},
					},
				},
				Timezone: "0",
			},
			expectNow:   true,
			description: "Currently in window, should return now",
		},
		{
			name: "out of window",
			spec: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "20:00",
						EndTime:   "22:00",
						Days:      []string{"tuesday"},
					},
				},
				Timezone: "0",
			},
			expectFuture: true,
			description:  "Out of window, should return future time",
		},
		{
			name: "with exclude date - excluded",
			spec: Spec{
				Items: []TimeGateItem{
					{
						StartTime: "09:00",
						EndTime:   "17:00",
						Days:      []string{"tuesday"},
					},
				},
				ExcludeDates: []ExcludeDate{
					{
						Date:      "11-05", // November 5 (test date)
						StartTime: "00:00",
						EndTime:   "23:59",
					},
				},
				Timezone: "0",
			},
			expectFuture: true,
			description:  "Date is excluded, should return future time (next valid day)",
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
