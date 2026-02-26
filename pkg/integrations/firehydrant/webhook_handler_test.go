package firehydrant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__FireHydrantWebhookHandler__CompareConfig(t *testing.T) {
	handler := &FireHydrantWebhookHandler{}

	t.Run("identical subscriptions -> match", func(t *testing.T) {
		match, err := handler.CompareConfig(
			map[string]any{"subscriptions": []any{"incidents"}},
			map[string]any{"subscriptions": []any{"incidents"}},
		)

		require.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("first is subset of second -> match", func(t *testing.T) {
		match, err := handler.CompareConfig(
			map[string]any{"subscriptions": []any{"incidents"}},
			map[string]any{"subscriptions": []any{"incidents", "incidents.private"}},
		)

		require.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("second is subset of first -> match", func(t *testing.T) {
		match, err := handler.CompareConfig(
			map[string]any{"subscriptions": []any{"incidents", "incidents.private"}},
			map[string]any{"subscriptions": []any{"incidents"}},
		)

		require.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("disjoint subscriptions -> no match", func(t *testing.T) {
		match, err := handler.CompareConfig(
			map[string]any{"subscriptions": []any{"incidents"}},
			map[string]any{"subscriptions": []any{"incidents.private"}},
		)

		require.NoError(t, err)
		assert.False(t, match)
	})

	t.Run("both empty subscriptions -> match", func(t *testing.T) {
		match, err := handler.CompareConfig(
			map[string]any{"subscriptions": []any{}},
			map[string]any{"subscriptions": []any{}},
		)

		require.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("invalid first config -> error", func(t *testing.T) {
		_, err := handler.CompareConfig("invalid", map[string]any{})
		require.Error(t, err)
	})

	t.Run("invalid second config -> error", func(t *testing.T) {
		_, err := handler.CompareConfig(map[string]any{}, "invalid")
		require.Error(t, err)
	})
}

func Test__FireHydrantWebhookHandler__Merge(t *testing.T) {
	handler := &FireHydrantWebhookHandler{}

	t.Run("current is strict subset of requested -> expands", func(t *testing.T) {
		result, changed, err := handler.Merge(
			map[string]any{"subscriptions": []any{"incidents"}},
			map[string]any{"subscriptions": []any{"incidents", "incidents.private"}},
		)

		require.NoError(t, err)
		assert.True(t, changed)

		merged, ok := result.(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"incidents", "incidents.private"}, merged.Subscriptions)
	})

	t.Run("current equals requested -> no change", func(t *testing.T) {
		current := map[string]any{"subscriptions": []any{"incidents", "incidents.private"}}
		result, changed, err := handler.Merge(
			current,
			map[string]any{"subscriptions": []any{"incidents", "incidents.private"}},
		)

		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, current, result)
	})

	t.Run("requested is subset of current -> no change", func(t *testing.T) {
		current := map[string]any{"subscriptions": []any{"incidents", "incidents.private"}}
		result, changed, err := handler.Merge(
			current,
			map[string]any{"subscriptions": []any{"incidents"}},
		)

		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, current, result)
	})

	t.Run("disjoint subscriptions -> no change", func(t *testing.T) {
		current := map[string]any{"subscriptions": []any{"incidents"}}
		result, changed, err := handler.Merge(
			current,
			map[string]any{"subscriptions": []any{"incidents.private"}},
		)

		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, current, result)
	})

	t.Run("invalid current config -> error", func(t *testing.T) {
		_, _, err := handler.Merge("invalid", map[string]any{})
		require.Error(t, err)
	})

	t.Run("invalid requested config -> error", func(t *testing.T) {
		_, _, err := handler.Merge(map[string]any{}, "invalid")
		require.Error(t, err)
	})
}
