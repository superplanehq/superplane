package schedule

import (
	"testing"
	"time"
)

// A daily schedule configured for a named city must fire at the configured
// wall-clock time in that city all year, across DST transitions. Previously the
// picker submitted a numeric offset that became a time.FixedZone with no DST
// rules, so summer instants fired an hour late. See issue #6565.
func TestSchedule_DailyJobFiresOnTimeAcrossDST(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("loading America/New_York: %v", err)
	}

	hour, minute, interval := 9, 0, 1
	tz := "America/New_York"

	config := Configuration{
		Type:         TypeDays,
		DaysInterval: &interval,
		Hour:         &hour,
		Minute:       &minute,
		Timezone:     &tz,
	}

	tests := []struct {
		label   string
		instant time.Time
	}{
		{"winter (EST)", time.Date(2026, 1, 15, 3, 0, 0, 0, time.UTC)},
		{"summer (EDT)", time.Date(2026, 8, 5, 3, 0, 0, 0, time.UTC)},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			next, err := getNextTrigger(config, tc.instant, nil)
			if err != nil {
				t.Fatalf("getNextTrigger: %v", err)
			}

			got := next.In(newYork).Format("15:04")
			if got != "09:00" {
				t.Errorf("configured 09:00 New York time, but fired at %s", got)
			}
		})
	}
}

// A bare numeric offset is still honoured as a genuine fixed zone for backward
// compatibility with configs saved before IANA identifiers were introduced.
func TestSchedule_NumericOffsetRemainsFixed(t *testing.T) {
	offset := "-5"
	loc := parseTimezone(&offset)

	_, offsetSeconds := time.Date(2026, 8, 5, 12, 0, 0, 0, loc).Zone()
	if offsetSeconds != -5*3600 {
		t.Errorf("expected fixed -5h (%d s), got %d s", -5*3600, offsetSeconds)
	}
}
