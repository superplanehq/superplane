package schedule

import (
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestNextMinutesTrigger(t *testing.T) {
	tests := []struct {
		name        string
		interval    int
		now         time.Time
		expectNext  time.Time
		expectError bool
	}{

		{
			name:       "5 minute interval exactly at interval boundary",
			interval:   5,
			now:        mustParseTime("2025-01-01T12:25:00Z"),
			expectNext: mustParseTime("2025-01-01T12:30:00Z"),
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
			result, err := nextMinutesTrigger(tt.interval, tt.now, nil)

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
	tests := []struct {
		name           string
		config         Configuration
		now            time.Time
		expectNext     time.Time
		expectError    bool
		expectErrorMsg string
	}{

		{
			name: "hours configuration",
			config: Configuration{
				Type:          TypeHours,
				HoursInterval: intPtr(1),
				Minute:        intPtr(30),
			},
			now:        mustParseTime("2025-01-01T12:15:00Z"),
			expectNext: mustParseTime("2025-01-01T13:30:00Z"),
		},
		{
			name: "days configuration",
			config: Configuration{
				Type:         TypeDays,
				DaysInterval: intPtr(1),
				Hour:         intPtr(14),
				Minute:       intPtr(30),
			},
			now:        mustParseTime("2025-01-01T10:00:00Z"),
			expectNext: mustParseTime("2025-01-02T14:30:00Z"),
		},
		{
			name: "weeks configuration",
			config: Configuration{
				Type:          TypeWeeks,
				WeeksInterval: intPtr(1),
				WeekDays:      []string{"friday"},
				Hour:          intPtr(15),
				Minute:        intPtr(30),
			},
			now:        mustParseTime("2025-01-06T10:00:00Z"), // Monday
			expectNext: mustParseTime("2025-01-17T15:30:00Z"), // Friday of next week
		},
		{
			name: "months configuration",
			config: Configuration{
				Type:           TypeMonths,
				MonthsInterval: intPtr(1),
				DayOfMonth:     intPtr(15),
				Hour:           intPtr(14),
				Minute:         intPtr(30),
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
			expectErrorMsg: "minutesInterval is required for minutes schedule",
		},
		{
			name: "invalid interval for minutes",
			config: Configuration{
				Type:            TypeMinutes,
				MinutesInterval: intPtr(60), // Too high
			},
			expectError:    true,
			expectErrorMsg: "interval must be between 1 and 59 minutes",
		},
		{
			name: "invalid interval for hours",
			config: Configuration{
				Type:          TypeHours,
				HoursInterval: intPtr(25), // Too high
			},
			expectError:    true,
			expectErrorMsg: "interval must be between 1 and 23 hours",
		},
		{
			name: "weeks without weekDays",
			config: Configuration{
				Type:          TypeWeeks,
				WeeksInterval: intPtr(1),
			},
			expectError:    true,
			expectErrorMsg: "weekDays is required for weeks schedule",
		},
		{
			name: "months without dayOfMonth",
			config: Configuration{
				Type:           TypeMonths,
				MonthsInterval: intPtr(1),
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
			result, err := getNextTrigger(tt.config, tt.now, nil)

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
				next, err := nextMinutesTrigger(tt.interval, currentTime, nil)
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

func TestTimezoneHandling(t *testing.T) {
	tests := []struct {
		name       string
		config     Configuration
		now        time.Time
		expectNext time.Time
	}{
		{
			name: "hours schedule in GMT-3 timezone",
			config: Configuration{
				Type:          TypeHours,
				HoursInterval: intPtr(1),
				Minute:        intPtr(1),       // 1 minute past every hour
				Timezone:      stringPtr("-3"), // GMT-3
			},
			now:        mustParseTime("2025-01-01T07:00:00Z"), // 4 AM GMT-3 (7 AM UTC)
			expectNext: mustParseTime("2025-01-01T08:01:00Z"), // 5:01 AM GMT-3 (8:01 AM UTC)
		},
		{
			name: "day schedule in GMT+5 timezone",
			config: Configuration{
				Type:         TypeDays,
				DaysInterval: intPtr(1),
				Hour:         intPtr(14),     // 2 PM in GMT+5
				Minute:       intPtr(30),     // 30 minutes past
				Timezone:     stringPtr("5"), // GMT+5
			},
			now:        mustParseTime("2025-01-01T08:00:00Z"), // 1 PM GMT+5 (8 AM UTC)
			expectNext: mustParseTime("2025-01-02T09:30:00Z"), // 2:30 PM GMT+5 (9:30 AM UTC)
		},
		{
			name: "week schedule in GMT-8 timezone (PST)",
			config: Configuration{
				Type:          TypeWeeks,
				WeeksInterval: intPtr(1),
				WeekDays:      []string{"monday"},
				Hour:          intPtr(9),       // 9 AM PST
				Minute:        intPtr(0),       // on the hour
				Timezone:      stringPtr("-8"), // GMT-8 (PST)
			},
			now:        mustParseTime("2025-01-06T16:00:00Z"), // Monday 8 AM PST (4 PM UTC)
			expectNext: mustParseTime("2025-01-13T17:00:00Z"), // Monday 9 AM PST (5 PM UTC) of the next week
		},
		{
			name: "month schedule in GMT+9 timezone (JST)",
			config: Configuration{
				Type:           TypeMonths,
				MonthsInterval: intPtr(1),
				DayOfMonth:     intPtr(15),     // 15th of the month
				Hour:           intPtr(12),     // Noon JST
				Minute:         intPtr(0),      // on the hour
				Timezone:       stringPtr("9"), // GMT+9 (JST)
			},
			now:        mustParseTime("2025-01-01T02:00:00Z"), // Jan 1st 11 AM JST (2 AM UTC)
			expectNext: mustParseTime("2025-02-15T03:00:00Z"), // Jan 15th Noon JST (3 AM UTC) of the next month
		},
		{
			name: "minutes schedule timezone should not affect calculation",
			config: Configuration{
				Type:            TypeMinutes,
				MinutesInterval: intPtr(30),
				Timezone:        stringPtr("-5"), // GMT-5
			},
			now:        mustParseTime("2025-01-01T12:15:00Z"),
			expectNext: mustParseTime("2025-01-01T12:45:00Z"), // 30 minutes later
		},
		{
			name: "cron schedule in GMT+2 timezone",
			config: Configuration{
				Type:           TypeCron,
				CronExpression: stringPtr("0 30 14 * * *"), // Daily at 2:30 PM
				Timezone:       stringPtr("2"),             // GMT+2
			},
			now:        mustParseTime("2025-01-01T11:00:00Z"), // 1 PM GMT+2 (11 AM UTC)
			expectNext: mustParseTime("2025-01-01T12:30:00Z"), // 2:30 PM GMT+2 (12:30 PM UTC)
		},
		{
			name: "cross-day boundary in negative timezone",
			config: Configuration{
				Type:          TypeHours,
				HoursInterval: intPtr(1),
				Minute:        intPtr(30),      // 30 minutes past every hour
				Timezone:      stringPtr("-5"), // GMT-5
			},
			now:        mustParseTime("2025-01-01T03:00:00Z"), // 10 PM GMT-5 (3 AM UTC next day)
			expectNext: mustParseTime("2025-01-01T04:30:00Z"), // 11:30 PM GMT-5 (4:30 AM UTC next day)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getNextTrigger(tt.config, tt.now, nil)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !result.Equal(tt.expectNext) {
				t.Errorf("expected next trigger at %v, got %v", tt.expectNext, *result)
				t.Errorf("Expected in local time: %v", tt.expectNext.In(parseTimezone(tt.config.Timezone)))
				t.Errorf("Got in local time: %v", result.In(parseTimezone(tt.config.Timezone)))
			}
		})
	}
}

func TestEmitEvent(t *testing.T) {
	tests := []struct {
		name               string
		config             Configuration
		timezone           string
		shouldHaveTimezone bool
	}{
		{
			name: "emit event with minutes interval (no timezone)",
			config: Configuration{
				Type:            TypeMinutes,
				MinutesInterval: intPtr(5),
			},
			shouldHaveTimezone: false,
		},
		{
			name: "emit event with hours interval (no timezone)",
			config: Configuration{
				Type:          TypeHours,
				HoursInterval: intPtr(1),
				Minute:        intPtr(0),
			},
			shouldHaveTimezone: false,
		},
		{
			name: "emit event with days interval (with timezone)",
			config: Configuration{
				Type:         TypeDays,
				DaysInterval: intPtr(1),
				Hour:         intPtr(9),
				Minute:       intPtr(0),
				Timezone:     stringPtr("-5"),
			},
			timezone:           "GMT-5.0 (UTC-05:00)",
			shouldHaveTimezone: true,
		},
		{
			name: "emit event with weeks interval (with timezone)",
			config: Configuration{
				Type:          TypeWeeks,
				WeeksInterval: intPtr(1),
				WeekDays:      []string{"monday"},
				Hour:          intPtr(14),
				Minute:        intPtr(30),
				Timezone:      stringPtr("1"),
			},
			timezone:           "GMT+1.0 (UTC+01:00)",
			shouldHaveTimezone: true,
		},
		{
			name: "emit event with cron interval (with timezone)",
			config: Configuration{
				Type:           TypeCron,
				CronExpression: stringPtr("0 30 14 * * *"),
				Timezone:       stringPtr("2"),
			},
			timezone:           "GMT+2.0 (UTC+02:00)",
			shouldHaveTimezone: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := &Schedule{}

			// Mock event context that captures the emitted payload
			var emittedPayload map[string]any
			mockEventContext := &mockEventContext{
				emitFunc: func(payload any) error {
					if p, ok := payload.(map[string]any); ok {
						emittedPayload = p
					}
					return nil
				},
			}

			// Mock request context
			mockRequestContext := &mockRequestContext{}

			ctx := core.TriggerActionContext{
				Name:            "emitEvent",
				Configuration:   tt.config,
				Logger:          log.NewEntry(log.StandardLogger()),
				EventContext:    mockEventContext,
				MetadataContext: &contexts.MetadataContext{},
				RequestContext:  mockRequestContext,
			}

			err := schedule.emitEvent(ctx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Validate payload structure
			if emittedPayload == nil {
				t.Errorf("expected payload to be emitted, but got nil")
				return
			}

			baseFields := []string{
				"timestamp", "Readable date", "Readable time", "Day of week",
				"Year", "Month", "Day of month", "Hour", "Minute", "Second",
			}

			for _, field := range baseFields {
				if _, ok := emittedPayload[field]; !ok {
					t.Errorf("expected field %q in payload, but it was missing", field)
				}
			}

			// Validate timezone field presence based on schedule type
			if tt.shouldHaveTimezone {
				if timezone, ok := emittedPayload["Timezone"].(string); ok {
					if timezone != tt.timezone {
						t.Errorf("expected timezone %q, got %q", tt.timezone, timezone)
					}
				} else {
					t.Errorf("expected Timezone field to be present and be a string")
				}
			} else {
				if _, ok := emittedPayload["Timezone"]; ok {
					t.Errorf("expected Timezone field to be absent for schedule type %q", tt.config.Type)
				}
			}

			// Validate timestamp format
			if timestamp, ok := emittedPayload["timestamp"].(string); ok {
				_, err := time.Parse(time.RFC3339, timestamp)
				if err != nil {
					t.Errorf("expected timestamp to be in RFC3339 format, but parsing failed: %v", err)
				}
			} else {
				t.Errorf("expected timestamp field to be a string")
			}
		})
	}
}

// Mock implementations for testing
type mockEventContext struct {
	emitFunc func(any) error
}

func (m *mockEventContext) Emit(payload any) error {
	if m.emitFunc != nil {
		return m.emitFunc(payload)
	}
	return nil
}

type mockMetadataContext struct {
	data any
}

func (m *mockMetadataContext) Get() any {
	return m.data
}

func (m *mockMetadataContext) Set(data any) {
	m.data = data
}

type mockRequestContext struct{}

func (m *mockRequestContext) ScheduleActionCall(actionName string, payload map[string]any, delay time.Duration) error {
	// Not implemented for this test
	return nil
}
