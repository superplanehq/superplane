package display

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatRelativeTime(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	zero := time.Time{}
	future := now.Add(2 * time.Minute)
	oneSecond := now.Add(-time.Second)
	manySeconds := now.Add(-45 * time.Second)
	oneMinute := now.Add(-time.Minute)
	manyMinutes := now.Add(-5 * time.Minute)
	compoundMinute := now.Add(-90 * time.Second)
	oneHour := now.Add(-time.Hour)
	manyHours := now.Add(-6 * time.Hour)
	oneDay := now.Add(-24 * time.Hour)
	manyDays := now.Add(-72 * time.Hour)

	cases := []struct {
		name  string
		value *time.Time
		want  string
	}{
		{name: "nil", value: nil, want: "-"},
		{name: "zero", value: &zero, want: "-"},
		{name: "future", value: &future, want: "1s ago"},
		{name: "one second", value: &oneSecond, want: "1s ago"},
		{name: "many seconds", value: &manySeconds, want: "45s ago"},
		{name: "one minute", value: &oneMinute, want: "1m ago"},
		{name: "many minutes", value: &manyMinutes, want: "5m ago"},
		{name: "largest unit only", value: &compoundMinute, want: "1m ago"},
		{name: "one hour", value: &oneHour, want: "1h ago"},
		{name: "many hours", value: &manyHours, want: "6h ago"},
		{name: "one day", value: &oneDay, want: "1d ago"},
		{name: "many days", value: &manyDays, want: "3d ago"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, FormatRelativeTime(tc.value, now))
		})
	}
}
