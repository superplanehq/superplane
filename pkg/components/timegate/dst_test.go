package timegate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// A time gate configured for a named city must interpret its window in that
// city's wall-clock time all year, across DST transitions. Previously the picker
// submitted a numeric offset that became a time.FixedZone with no DST rules, so
// a business-hours window was effectively shifted an hour during DST. See issue
// #6565. At 2026-08-05T13:30:00Z it is 09:30 in New York, inside a 09:00-17:00
// gate.
func TestTimeGate_ParseTimezone_AppliesDST(t *testing.T) {
	tg := &TimeGate{}
	loc := tg.parseTimezone("America/New_York")

	instant := time.Date(2026, 8, 5, 13, 30, 0, 0, time.UTC)
	got := instant.In(loc).Format("15:04")
	assert.Equal(t, "09:30", got, "summer instant should be 09:30 in New York (EDT, offset -4h)")

	winter := time.Date(2026, 1, 15, 13, 30, 0, 0, time.UTC)
	assert.Equal(t, "08:30", winter.In(loc).Format("15:04"), "winter instant should be 08:30 in New York (EST, offset -5h)")
}

// A bare numeric offset is still honoured as a genuine fixed zone for backward
// compatibility with configs saved before IANA identifiers were introduced.
func TestTimeGate_ParseTimezone_NumericOffsetRemainsFixed(t *testing.T) {
	tg := &TimeGate{}
	loc := tg.parseTimezone("-5")

	_, offsetSeconds := time.Date(2026, 8, 5, 12, 0, 0, 0, loc).Zone()
	assert.Equal(t, -5*3600, offsetSeconds)
}

// Empty and "current" resolve to UTC rather than the server's local zone, so the
// same config behaves identically regardless of the host's TZ setting.
func TestTimeGate_ParseTimezone_EmptyAndCurrentAreUTC(t *testing.T) {
	tg := &TimeGate{}
	assert.Equal(t, time.UTC, tg.parseTimezone(""))
	assert.Equal(t, time.UTC, tg.parseTimezone("current"))
}
