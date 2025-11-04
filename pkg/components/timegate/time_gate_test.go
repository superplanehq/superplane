package timegate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	assert.Len(t, config, 4) // mode, startTime, endTime, days

	// Check mode field
	assert.Equal(t, "mode", config[0].Name)
	assert.Equal(t, "Mode", config[0].Label)
	assert.True(t, config[0].Required)

	// Check startTime field
	assert.Equal(t, "startTime", config[1].Name)
	assert.Equal(t, "Start Time (HH:MM)", config[1].Label)
	assert.True(t, config[1].Required)
	assert.Equal(t, "09:00", config[1].Default)

	// Check endTime field
	assert.Equal(t, "endTime", config[2].Name)
	assert.Equal(t, "End Time (HH:MM)", config[2].Label)
	assert.True(t, config[2].Required)
	assert.Equal(t, "17:00", config[2].Default)

	// Check days field
	assert.Equal(t, "days", config[3].Name)
	assert.Equal(t, "Days of Week", config[3].Label)
	assert.True(t, config[3].Required)
}

func TestTimeGate_Actions(t *testing.T) {
	tg := &TimeGate{}
	actions := tg.Actions()
	assert.Len(t, actions, 1)
	assert.Equal(t, "timeReached", actions[0].Name)
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
			name: "valid include spec",
			spec: Spec{
				Mode:      "include",
				StartTime: "09:00",
				EndTime:   "17:00",
				Days:      []string{"monday", "tuesday"},
			},
			hasError: false,
		},
		{
			name: "valid exclude spec",
			spec: Spec{
				Mode:      "exclude",
				StartTime: "13:00",
				EndTime:   "14:00",
				Days:      []string{"friday"},
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
			name: "invalid start time",
			spec: Spec{
				Mode:      "include",
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
				Mode:      "include",
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
				Mode:      "include",
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
				Mode:      "include",
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
				Mode:      "include",
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
				Mode:      "include",
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
		Mode:      "include",
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
				Mode:      "include",
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
				Mode:      "exclude", // different mode
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
				Mode:      "include",
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
				Mode:      "include",
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
				Mode:      "include",
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
				Mode:      "include",
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
				Mode:      "include",
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
				Mode:      "include",
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
				Mode:      "include",
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
				Mode:      "include",
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
				Mode:      "exclude",
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
				Mode:      "exclude",
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
				Mode:      "exclude",
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
				Mode:      "exclude",
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
				Mode:      "include",
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
				Mode:      "include",
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
				Mode:      "exclude",
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
				Mode:      "exclude",
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
