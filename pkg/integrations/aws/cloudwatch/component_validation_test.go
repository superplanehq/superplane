package cloudwatch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimestamp(t *testing.T) {
	t.Run("RFC3339 -> parsed as-is", func(t *testing.T) {
		parsed, err := parseTimestamp("2026-05-29T09:00:00Z")
		require.NoError(t, err)
		assert.Equal(t, "2026-05-29T09:00:00Z", parsed.Format(time.RFC3339))
	})

	t.Run("datetime-local without seconds -> parsed as UTC", func(t *testing.T) {
		parsed, err := parseTimestamp("2026-05-29T09:00")
		require.NoError(t, err)
		assert.Equal(t, "2026-05-29T09:00:00Z", parsed.Format(time.RFC3339))
	})

	t.Run("datetime-local with seconds -> parsed as UTC", func(t *testing.T) {
		parsed, err := parseTimestamp("2026-05-29T09:00:30")
		require.NoError(t, err)
		assert.Equal(t, "2026-05-29T09:00:30Z", parsed.Format(time.RFC3339))
	})

	t.Run("garbage -> error", func(t *testing.T) {
		_, err := parseTimestamp("not-a-date")
		require.ErrorContains(t, err, "invalid timestamp")
	})
}
