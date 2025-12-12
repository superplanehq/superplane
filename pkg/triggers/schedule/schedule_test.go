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
		expectNext    time.Time
		expectError   bool
	}{
		{
			name:       "10 minute interval with reference time",
			interval:   10,
			now:        mustParseTime("2025-01-01T12:35:00Z"),
			expectNext: mustParseTime("2025-01-01T12:45:00Z"),
		},
		{
			name:       "5 minute interval exactly at interval boundary",
			interval:   5,
			now:        mustParseTime("2025-01-01T12:25:00Z"),
			expectNext: mustParseTime("2025-01-01T12:30:00Z"),
		},
		{
			name:          "15 minute interval with reference time in past",
			interval:      15,
			now:           mustParseTime("2025-01-01T12:40:00Z"),
			referenceTime: stringPtr("2025-01-01T12:00:00Z"),
			expectNext:    mustParseTime("2025-01-01T12:45:00Z"),
		},
		{
			name:       "30 minute interval crossing day boundary",
			interval:   30,
			now:        mustParseTime("2025-01-01T23:50:00Z"),
			expectNext: mustParseTime("2025-01-02T00:20:00Z"),
		},
		{
			name:       "1 minute interval high frequency",
			interval:   1,
			now:        mustParseTime("2025-01-01T12:30:30Z"),
			expectNext: mustParseTime("2025-01-01T12:31:30Z"),
		},
		{
			name:       "no reference time provided - use current time",
			interval:   20,
			now:        mustParseTime("2025-01-01T12:30:00Z"),
			expectNext: mustParseTime("2025-01-01T12:50:00Z"),
		},
		{
			name:          "reference time in future - should handle gracefully",
			interval:      10,
			now:           mustParseTime("2025-01-01T12:30:00Z"),
			referenceTime: stringPtr("2025-01-01T13:00:00Z"),
			expectNext:    mustParseTime("2025-01-01T13:10:00Z"),
		},
		{
			name:        "invalid interval - too small",
			interval:    0,
			now:         mustParseTime("2025-01-01T12:30:00Z"),
			expectError: true,
		},
		{
			name:        "invalid interval - too large",
			interval:    60,
			now:         mustParseTime("2025-01-01T12:30:00Z"),
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

			if !result.Equal(tt.expectNext) {
				t.Errorf("expected next trigger at %v, got %v", tt.expectNext, *result)
			}
		})
	}
}

func TestGetNextTrigger(t *testing.T) {
	refTime := "2025-01-01T12:30:00Z"
	tests := []struct {
		name           string
		config         Configuration
		now            time.Time
		referenceTime  *string
		expectNext     time.Time
		expectError    bool
		expectErrorMsg string
	}{
		{
			name: "minutes configuration with reference time",
			config: Configuration{
				Type:     TypeMinutes,
				Interval: intPtr(10),
			},
			now:           mustParseTime("2025-01-01T12:35:00Z"),
			referenceTime: &refTime,
			expectNext:    mustParseTime("2025-01-01T12:40:00Z"),
		},
		{
			name: "hours configuration",
			config: Configuration{
				Type:     TypeHours,
				Interval: intPtr(1),
				Minute:   intPtr(30),
			},
			now:        mustParseTime("2025-01-01T12:15:00Z"),
			expectNext: mustParseTime("2025-01-01T13:30:00Z"),
		},
		{
			name: "days configuration",
			config: Configuration{
				Type:     TypeDays,
				Interval: intPtr(1),
				Hour:     intPtr(14),
				Minute:   intPtr(30),
			},
			now:        mustParseTime("2025-01-01T10:00:00Z"),
			expectNext: mustParseTime("2025-01-02T14:30:00Z"),
		},
		{
			name: "weeks configuration",
			config: Configuration{
				Type:     TypeWeeks,
				Interval: intPtr(1),
				WeekDays: []string{"friday"},
				Hour:     intPtr(15),
				Minute:   intPtr(30),
			},
			now:        mustParseTime("2025-01-06T10:00:00Z"), // Monday
			expectNext: mustParseTime("2025-01-10T15:30:00Z"), // Friday
		},
		{
			name: "months configuration",
			config: Configuration{
				Type:       TypeMonths,
				Interval:   intPtr(1),
				DayOfMonth: intPtr(15),
				Hour:       intPtr(14),
				Minute:     intPtr(30),
			},
			now:        mustParseTime("2025-01-01T10:00:00Z"),
			expectNext: mustParseTime("2025-02-15T14:30:00Z"),
		},
		{
			name: "cron configuration",
			config: Configuration{
				Type:           TypeCron,
				CronExpression: stringPtr("0 30 14 * * *"), // Daily at 14:30
			},
			now:        mustParseTime("2025-01-01T10:00:00Z"),
			expectNext: mustParseTime("2025-01-01T14:30:00Z"),
		},
		{
			name: "unsupported type",
			config: Configuration{
				Type: "invalid",
			},
			expectError:    true,
			expectErrorMsg: "unsupported schedule type",
		},
		{
			name: "minutes without interval",
			config: Configuration{
				Type: TypeMinutes,
			},
			expectError:    true,
			expectErrorMsg: "interval is required for minutes schedule",
		},
		{
			name: "invalid interval for minutes",
			config: Configuration{
				Type:     TypeMinutes,
				Interval: intPtr(60), // Too high
			},
			expectError:    true,
			expectErrorMsg: "minutes interval must be between 1 and 59",
		},
		{
			name: "invalid interval for hours",
			config: Configuration{
				Type:     TypeHours,
				Interval: intPtr(25), // Too high
			},
			expectError:    true,
			expectErrorMsg: "hours interval must be between 1 and 23",
		},
		{
			name: "weeks without weekDays",
			config: Configuration{
				Type:     TypeWeeks,
				Interval: intPtr(1),
			},
			expectError:    true,
			expectErrorMsg: "weekDays is required for weeks schedule",
		},
		{
			name: "months without dayOfMonth",
			config: Configuration{
				Type:     TypeMonths,
				Interval: intPtr(1),
			},
			expectError:    true,
			expectErrorMsg: "dayOfMonth is required for months schedule",
		},
		{
			name: "cron without expression",
			config: Configuration{
				Type: TypeCron,
			},
			expectError:    true,
			expectErrorMsg: "cronExpression is required for cron schedule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getNextTrigger(tt.config, tt.now, tt.referenceTime)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.expectErrorMsg != "" && !contains(err.Error(), tt.expectErrorMsg) {
					t.Errorf("expected error message to contain '%s', got: %s", tt.expectErrorMsg, err.Error())
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

func TestValidateIntervalForType(t *testing.T) {
	tests := []struct {
		name         string
		scheduleType string
		interval     int
		expectError  bool
	}{
		{"valid minutes interval", TypeMinutes, 30, false},
		{"invalid minutes interval - too low", TypeMinutes, 0, true},
		{"invalid minutes interval - too high", TypeMinutes, 60, true},
		{"valid hours interval", TypeHours, 12, false},
		{"invalid hours interval - too high", TypeHours, 24, true},
		{"valid days interval", TypeDays, 15, false},
		{"invalid days interval - too high", TypeDays, 32, true},
		{"valid weeks interval", TypeWeeks, 26, false},
		{"invalid weeks interval - too high", TypeWeeks, 53, true},
		{"valid months interval", TypeMonths, 12, false},
		{"invalid months interval - too high", TypeMonths, 25, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIntervalForType(tt.scheduleType, tt.interval)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMinutesSchedulingConsistency(t *testing.T) {
	tests := []struct {
		name     string
		interval int
		refTime  string
		duration time.Duration
	}{
		{
			name:     "consistency test A",
			interval: 15,
			refTime:  "2025-01-01T09:00:00Z",
			duration: 2 * time.Hour,
		},
		{
			name:     "consistency test B",
			interval: 7,
			refTime:  "2025-01-01T14:22:00Z",
			duration: 90 * time.Minute,
		},
		{
			name:     "consistency test C",
			interval: 23,
			refTime:  "2025-01-01T08:15:00Z",
			duration: 4 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime := mustParseTime(tt.refTime)
			endTime := startTime.Add(tt.duration)

			var triggers []time.Time
			currentTime := startTime

			for currentTime.Before(endTime) {
				next, err := nextMinutesTrigger(tt.interval, currentTime, &tt.refTime)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				triggers = append(triggers, *next)
				currentTime = *next
			}

			if len(triggers) < 2 {
				t.Skip("need at least 2 triggers for consistency test")
			}

			for i := 1; i < len(triggers); i++ {
				actualInterval := triggers[i].Sub(triggers[i-1])
				expectedInterval := time.Duration(tt.interval) * time.Minute

				if actualInterval != expectedInterval {
					t.Errorf("inconsistent interval: expected %v, got %v between triggers %v and %v",
						expectedInterval, actualInterval, triggers[i-1], triggers[i])
				}
			}
		})
	}
}

// Helper functions
func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func stringPtr(s string) *string {
	return &s
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		   (len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		   (len(substr) < len(s) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}