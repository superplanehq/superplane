package schedule

import (
	"testing"
	"time"
)

func TestNextMinutesTrigger(t *testing.T) {
	tests := []struct {
		name          string
		interval      int
		now           time.Time
		referenceTime *string
		expectError   bool
		expectNext    time.Time
	}{
		{
			name:     "10 minute interval with reference time",
			interval: 10,
			now:      mustParseTime("2025-01-01T12:35:00Z"),
			referenceTime: stringPtr("2025-01-01T12:00:00Z"),
			expectNext: mustParseTime("2025-01-01T12:40:00Z"), // 4 intervals of 10 min from 12:00
		},
		{
			name:     "5 minute interval exactly at interval boundary",
			interval: 5,
			now:      mustParseTime("2025-01-01T12:20:00Z"),
			referenceTime: stringPtr("2025-01-01T12:00:00Z"),
			expectNext: mustParseTime("2025-01-01T12:25:00Z"), // next interval after 12:20
		},
		{
			name:     "15 minute interval with reference time in past",
			interval: 15,
			now:      mustParseTime("2025-01-01T12:45:00Z"),
			referenceTime: stringPtr("2025-01-01T11:30:00Z"),
			expectNext: mustParseTime("2025-01-01T13:00:00Z"), // 6 intervals of 15 min from 11:30 (11:30, 11:45, 12:00, 12:15, 12:30, 12:45, 13:00)
		},
		{
			name:     "30 minute interval crossing day boundary",
			interval: 30,
			now:      mustParseTime("2025-01-01T23:50:00Z"),
			referenceTime: stringPtr("2025-01-01T23:00:00Z"),
			expectNext: mustParseTime("2025-01-02T00:00:00Z"), // 2 intervals from 23:00 (23:30, 00:00)
		},
		{
			name:     "1 minute interval high frequency",
			interval: 1,
			now:      mustParseTime("2025-01-01T12:05:30Z"),
			referenceTime: stringPtr("2025-01-01T12:00:00Z"),
			expectNext: mustParseTime("2025-01-01T12:06:00Z"), // next minute after 5:30
		},
		{
			name:     "no reference time provided - use current time",
			interval: 10,
			now:      mustParseTime("2025-01-01T12:05:00Z"),
			referenceTime: nil,
			expectNext: mustParseTime("2025-01-01T12:15:00Z"), // 10 minutes from now
		},
		{
			name:     "reference time in future - should handle gracefully",
			interval: 5,
			now:      mustParseTime("2025-01-01T12:00:00Z"),
			referenceTime: stringPtr("2025-01-01T12:10:00Z"),
			expectNext: mustParseTime("2025-01-01T12:15:00Z"), // first interval after reference time (since minutesElapsed is 0, next is reference + interval)
		},
		{
			name:        "invalid interval - too small",
			interval:    0,
			now:         mustParseTime("2025-01-01T12:00:00Z"),
			referenceTime: stringPtr("2025-01-01T12:00:00Z"),
			expectError: true,
		},
		{
			name:        "invalid interval - too large",
			interval:    1441,
			now:         mustParseTime("2025-01-01T12:00:00Z"),
			referenceTime: stringPtr("2025-01-01T12:00:00Z"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := nextMinutesTrigger(tt.interval, tt.now, tt.referenceTime)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("result is nil")
				return
			}

			if !result.Equal(tt.expectNext) {
				t.Errorf("expected next trigger at %v, got %v", tt.expectNext, *result)
			}

			// Verify the next trigger is always in the future
			if !result.After(tt.now) {
				t.Errorf("next trigger %v should be after now %v", *result, tt.now)
			}
		})
	}
}

func TestNextHourlyTrigger(t *testing.T) {
	tests := []struct {
		name        string
		minute      int
		now         time.Time
		expectNext  time.Time
		expectError bool
	}{
		{
			name:       "trigger at minute 30",
			minute:     30,
			now:        mustParseTime("2025-01-01T12:15:00Z"),
			expectNext: mustParseTime("2025-01-01T12:30:00Z"),
		},
		{
			name:       "trigger at minute 0, current time past",
			minute:     0,
			now:        mustParseTime("2025-01-01T12:15:00Z"),
			expectNext: mustParseTime("2025-01-01T13:00:00Z"),
		},
		{
			name:        "invalid minute - too large",
			minute:      60,
			now:         mustParseTime("2025-01-01T12:15:00Z"),
			expectError: true,
		},
		{
			name:        "invalid minute - negative",
			minute:      -1,
			now:         mustParseTime("2025-01-01T12:15:00Z"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := nextHourlyTrigger(tt.minute, tt.now)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !result.Equal(tt.expectNext) {
				t.Errorf("expected next trigger at %v, got %v", tt.expectNext, *result)
			}
		})
	}
}

func TestNextDailyTrigger(t *testing.T) {
	tests := []struct {
		name        string
		timeValue   string
		now         time.Time
		expectNext  time.Time
		expectError bool
	}{
		{
			name:       "trigger at 14:30 UTC, before time",
			timeValue:  "14:30",
			now:        mustParseTime("2025-01-01T10:00:00Z"),
			expectNext: mustParseTime("2025-01-01T14:30:00Z"),
		},
		{
			name:       "trigger at 14:30 UTC, after time",
			timeValue:  "14:30",
			now:        mustParseTime("2025-01-01T16:00:00Z"),
			expectNext: mustParseTime("2025-01-02T14:30:00Z"),
		},
		{
			name:        "invalid time format",
			timeValue:   "25:00",
			now:         mustParseTime("2025-01-01T12:00:00Z"),
			expectError: true,
		},
		{
			name:        "invalid time format - not HH:MM",
			timeValue:   "14",
			now:         mustParseTime("2025-01-01T12:00:00Z"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := nextDailyTrigger(tt.timeValue, tt.now)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !result.Equal(tt.expectNext) {
				t.Errorf("expected next trigger at %v, got %v", tt.expectNext, *result)
			}
		})
	}
}

func TestNextWeeklyTrigger(t *testing.T) {
	tests := []struct {
		name        string
		weekDay     string
		timeValue   string
		now         time.Time
		expectNext  time.Time
		expectError bool
	}{
		{
			name:       "trigger on Friday 15:30, current day is Monday",
			weekDay:    "friday",
			timeValue:  "15:30",
			now:        mustParseTime("2025-01-06T10:00:00Z"), // Monday
			expectNext: mustParseTime("2025-01-10T15:30:00Z"), // Friday
		},
		{
			name:       "trigger on Monday 09:00, current day is Monday after time",
			weekDay:    "monday",
			timeValue:  "09:00",
			now:        mustParseTime("2025-01-06T12:00:00Z"), // Monday 12:00
			expectNext: mustParseTime("2025-01-13T09:00:00Z"), // Next Monday
		},
		{
			name:        "invalid weekday",
			weekDay:     "funday",
			timeValue:   "09:00",
			now:         mustParseTime("2025-01-06T12:00:00Z"),
			expectError: true,
		},
		{
			name:        "invalid time",
			weekDay:     "monday",
			timeValue:   "25:00",
			now:         mustParseTime("2025-01-06T12:00:00Z"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := nextWeeklyTrigger(tt.weekDay, tt.timeValue, tt.now)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !result.Equal(tt.expectNext) {
				t.Errorf("expected next trigger at %v, got %v", tt.expectNext, *result)
			}
		})
	}
}

func TestGetNextTrigger(t *testing.T) {
	refTime := "2025-01-01T12:00:00Z"

	tests := []struct {
		name          string
		config        Configuration
		now           time.Time
		referenceTime *string
		expectError   bool
		expectNext    time.Time
	}{
		{
			name: "minutes configuration with reference time",
			config: Configuration{
				Type:     TypeMinutes,
				Interval: intPtr(15),
			},
			now:           mustParseTime("2025-01-01T12:35:00Z"),
			referenceTime: &refTime,
			expectNext:    mustParseTime("2025-01-01T12:45:00Z"),
		},
		{
			name: "hourly configuration",
			config: Configuration{
				Type:   TypeHourly,
				Minute: intPtr(30),
			},
			now:        mustParseTime("2025-01-01T12:15:00Z"),
			expectNext: mustParseTime("2025-01-01T12:30:00Z"),
		},
		{
			name: "daily configuration",
			config: Configuration{
				Type: TypeDaily,
				Time: stringPtr("14:30"),
			},
			now:        mustParseTime("2025-01-01T10:00:00Z"),
			expectNext: mustParseTime("2025-01-01T14:30:00Z"),
		},
		{
			name: "weekly configuration",
			config: Configuration{
				Type:    TypeWeekly,
				Time:    stringPtr("15:30"),
				WeekDay: stringPtr("friday"),
			},
			now:        mustParseTime("2025-01-06T10:00:00Z"), // Monday
			expectNext: mustParseTime("2025-01-10T15:30:00Z"), // Friday
		},
		{
			name: "unsupported type",
			config: Configuration{
				Type: "unsupported",
			},
			now:         mustParseTime("2025-01-01T12:00:00Z"),
			expectError: true,
		},
		{
			name: "minutes without interval",
			config: Configuration{
				Type: TypeMinutes,
			},
			now:         mustParseTime("2025-01-01T12:00:00Z"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getNextTrigger(tt.config, tt.now, tt.referenceTime)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !result.Equal(tt.expectNext) {
				t.Errorf("expected next trigger at %v, got %v", tt.expectNext, *result)
			}
		})
	}
}

func TestMinutesSchedulingConsistency(t *testing.T) {
	// Test that intervals remain consistent regardless of when the function is called
	referenceTime := "2025-01-01T12:00:00Z"
	interval := 10

	// Simulate calls at different times within the same interval
	testTimes := []time.Time{
		mustParseTime("2025-01-01T12:15:00Z"), // 15 minutes after reference
		mustParseTime("2025-01-01T12:18:30Z"), // 18.5 minutes after reference
		mustParseTime("2025-01-01T12:19:59Z"), // 19:59 after reference
	}

	expectedNext := mustParseTime("2025-01-01T12:20:00Z") // All should return 12:20

	for i, now := range testTimes {
		t.Run("consistency_test_"+string(rune('A'+i)), func(t *testing.T) {
			result, err := nextMinutesTrigger(interval, now, &referenceTime)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !result.Equal(expectedNext) {
				t.Errorf("inconsistent result for time %v: expected %v, got %v",
					now, expectedNext, *result)
			}
		})
	}
}

// Helper functions
func mustParseTime(timeStr string) time.Time {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		panic("failed to parse time: " + err.Error())
	}
	return t
}

func stringPtr(s string) *string {
	return &s
}

