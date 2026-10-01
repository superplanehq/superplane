package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimezone_IANAIdentifierFollowsDST(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	identifier := "America/New_York"
	hour, minute, interval := 9, 0, 1

	config := Configuration{
		Type:         TypeDays,
		DaysInterval: &interval,
		Hour:         &hour,
		Minute:       &minute,
		Timezone:     &identifier,
	}

	//
	// A daily 09:00 schedule must fire at 09:00 local time on both sides of a
	// daylight saving transition.
	//
	for _, tc := range []struct {
		name    string
		instant time.Time
	}{
		{"outside daylight saving", time.Date(2026, 1, 15, 3, 0, 0, 0, time.UTC)},
		{"during daylight saving", time.Date(2026, 8, 5, 3, 0, 0, 0, time.UTC)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			next, err := getNextTrigger(config, tc.instant, nil)
			require.NoError(t, err)
			assert.Equal(t, "09:00", next.In(newYork).Format("15:04"))
		})
	}
}

func TestParseTimezone_NumericOffsetStillSupported(t *testing.T) {
	offset := "-5"
	location := parseTimezone(&offset)

	_, seconds := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC).In(location).Zone()
	assert.Equal(t, -5*3600, seconds)
}

func TestParseTimezone_FallsBackToUTC(t *testing.T) {
	assert.Equal(t, time.UTC, parseTimezone(nil))

	empty := ""
	assert.Equal(t, time.UTC, parseTimezone(&empty))

	invalid := "Mars/Olympus_Mons"
	assert.Equal(t, time.UTC, parseTimezone(&invalid))
}
