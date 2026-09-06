package timegate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeGate_ParseTimezone_IANAIdentifierFollowsDST(t *testing.T) {
	tg := &TimeGate{}

	location := tg.parseTimezone("America/New_York")

	_, winterOffset := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC).In(location).Zone()
	_, summerOffset := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC).In(location).Zone()

	assert.Equal(t, -5*3600, winterOffset)
	assert.Equal(t, -4*3600, summerOffset)
}

func TestTimeGate_OpensInsideWindowDuringDST(t *testing.T) {
	tg := &TimeGate{}

	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	spec := Spec{
		Days:      []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"},
		TimeRange: "09:00-17:00",
		Timezone:  "America/New_York",
	}
	require.NoError(t, tg.validateSpec(spec))

	startMinutes, endMinutes, err := parseTimeRangeString(spec.TimeRange)
	require.NoError(t, err)

	//
	// 13:30 UTC is 09:30 in New York while daylight saving is active, which is
	// inside the configured window, so the gate must open immediately.
	//
	instant := time.Date(2026, 8, 5, 13, 30, 0, 0, time.UTC)
	now := instant.In(tg.parseTimezone(spec.Timezone))

	next := tg.findNextValidTime(now, spec, startMinutes, endMinutes)

	assert.Equal(t, "09:30", next.In(newYork).Format("15:04"),
		"gate should open now, not defer to a later window")
}

func TestTimeGate_ParseTimezone_NumericOffsetStillSupported(t *testing.T) {
	tg := &TimeGate{}

	_, offset := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC).In(tg.parseTimezone("-5")).Zone()
	assert.Equal(t, -5*3600, offset)
}

func TestTimeGate_ParseTimezone_Fallbacks(t *testing.T) {
	tg := &TimeGate{}

	assert.Equal(t, time.Local, tg.parseTimezone(""))
	assert.Equal(t, time.Local, tg.parseTimezone("current"))
	assert.Equal(t, time.UTC, tg.parseTimezone("Mars/Olympus_Mons"))
}
