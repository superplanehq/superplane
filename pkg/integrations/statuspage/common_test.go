package statuspage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_extractComponentIDs(t *testing.T) {
	t.Run("empty config returns nil", func(t *testing.T) {
		assert.Nil(t, extractComponentIDs(map[string]any{}))
		assert.Nil(t, extractComponentIDs(map[string]any{"components": nil}))
	})

	t.Run("valid components list extracts IDs", func(t *testing.T) {
		config := map[string]any{
			"components": []any{
				map[string]any{"componentId": "comp1", "status": "degraded_performance"},
				map[string]any{"componentId": "comp2", "status": "operational"},
			},
		}
		ids := extractComponentIDs(config)
		assert.Equal(t, []string{"comp1", "comp2"}, ids)
	})

	t.Run("skips items with empty componentId", func(t *testing.T) {
		config := map[string]any{
			"components": []any{
				map[string]any{"componentId": "comp1", "status": "degraded_performance"},
				map[string]any{"componentId": "", "status": "operational"},
				map[string]any{"status": "operational"},
				map[string]any{"componentId": "comp2", "status": "operational"},
			},
		}
		ids := extractComponentIDs(config)
		assert.Equal(t, []string{"comp1", "comp2"}, ids)
	})

	t.Run("handles malformed items", func(t *testing.T) {
		config := map[string]any{
			"components": []any{
				"not an object",
				123,
				map[string]any{"componentId": "comp1", "status": "degraded_performance"},
			},
		}
		ids := extractComponentIDs(config)
		assert.Equal(t, []string{"comp1"}, ids)
	})

	t.Run("non-list returns nil", func(t *testing.T) {
		assert.Nil(t, extractComponentIDs(map[string]any{"components": "not a list"}))
		assert.Nil(t, extractComponentIDs(map[string]any{"components": 123}))
	})
}

func Test_containsExpression(t *testing.T) {
	assert.False(t, containsExpression(nil))
	assert.False(t, containsExpression([]string{}))
	assert.False(t, containsExpression([]string{"comp1", "comp2"}))
	assert.True(t, containsExpression([]string{"comp1", "{{ $['X'].data.id }}"}))
	assert.True(t, containsExpression([]string{"{{ expression }}"}))
}

func Test_toUTCISO8601(t *testing.T) {
	t.Run("Z suffix means UTC regardless of timezone param", func(t *testing.T) {
		// "2026-02-15T02:00:00Z" is 02:00 UTC. Must not be re-interpreted as 02:00 America/New_York.
		out, err := toUTCISO8601("2026-02-15T02:00:00Z", "America/New_York")
		require.NoError(t, err)
		assert.Equal(t, "2026-02-15T02:00:00Z", out)
	})

	t.Run("no Z suffix interpreted in given timezone", func(t *testing.T) {
		// "2026-02-15T02:00" in America/New_York (EST, UTC-5) = 07:00 UTC
		out, err := toUTCISO8601("2026-02-15T02:00", "America/New_York")
		require.NoError(t, err)
		assert.Equal(t, "2026-02-15T07:00:00Z", out)
	})

	t.Run("no Z with UTC timezone", func(t *testing.T) {
		out, err := toUTCISO8601("2026-02-15T02:00", "UTC")
		require.NoError(t, err)
		assert.Equal(t, "2026-02-15T02:00:00Z", out)
	})
}
