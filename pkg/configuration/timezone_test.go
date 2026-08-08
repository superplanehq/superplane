package configuration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadTimezone_IANAIdentifierFollowsDST(t *testing.T) {
	location, err := LoadTimezone("America/New_York")
	require.NoError(t, err)

	winter := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC).In(location)
	summer := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC).In(location)

	_, winterOffset := winter.Zone()
	_, summerOffset := summer.Zone()

	assert.Equal(t, -5*3600, winterOffset, "New York is UTC-5 outside daylight saving")
	assert.Equal(t, -4*3600, summerOffset, "New York is UTC-4 during daylight saving")
}

func TestLoadTimezone_NumericOffsetStaysFixed(t *testing.T) {
	//
	// Numeric offsets are still accepted for configurations saved before IANA
	// identifiers were supported. They intentionally do not follow DST.
	//
	location, err := LoadTimezone("-5")
	require.NoError(t, err)

	_, offset := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC).In(location).Zone()
	assert.Equal(t, -5*3600, offset)
}

func TestLoadTimezone_AcceptsHalfHourAndPlusPrefix(t *testing.T) {
	location, err := LoadTimezone("5.5")
	require.NoError(t, err)
	_, offset := time.Now().In(location).Zone()
	assert.Equal(t, 5*3600+1800, offset)

	location, err = LoadTimezone("+8")
	require.NoError(t, err)
	_, offset = time.Now().In(location).Zone()
	assert.Equal(t, 8*3600, offset)
}

func TestLoadTimezone_Invalid(t *testing.T) {
	for _, tc := range []struct{ name, value string }{
		{"empty", ""},
		{"unknown identifier", "Mars/Olympus_Mons"},
		{"offset below range", "-13"},
		{"offset above range", "15"},
		{"quarter hour offset", "5.75"},
		{"server local", "Local"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadTimezone(tc.value)
			assert.Error(t, err)
		})
	}
}

func TestValidateTimezone(t *testing.T) {
	field := Field{Name: "timezone", Type: FieldTypeTimezone}

	for _, value := range []string{"America/New_York", "Europe/London", "UTC", "-5", "+8", "5.5"} {
		assert.NoError(t, validateTimezone(field, value), "expected %q to be valid", value)
	}

	for _, value := range []string{"", "current", "Local", "Mars/Olympus_Mons", "-13", "5.75"} {
		assert.Error(t, validateTimezone(field, value), "expected %q to be rejected", value)
	}

	assert.Error(t, validateTimezone(field, 5), "non-string values are rejected")
}
