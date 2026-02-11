package cursor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__CursorWebhookHandler__CompareConfig(t *testing.T) {
	h := &CursorWebhookHandler{}

	t.Run("always returns true", func(t *testing.T) {
		result, err := h.CompareConfig(nil, nil)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("returns true with different configs", func(t *testing.T) {
		result, err := h.CompareConfig(map[string]any{"a": 1}, map[string]any{"b": 2})
		require.NoError(t, err)
		assert.True(t, result)
	})
}

func Test__CursorWebhookHandler__Merge(t *testing.T) {
	h := &CursorWebhookHandler{}

	t.Run("returns current config unchanged", func(t *testing.T) {
		current := map[string]any{"key": "value"}
		requested := map[string]any{"other": "data"}

		result, changed, err := h.Merge(current, requested)
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, current, result)
	})

	t.Run("handles nil inputs", func(t *testing.T) {
		result, changed, err := h.Merge(nil, nil)
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Nil(t, result)
	})
}
