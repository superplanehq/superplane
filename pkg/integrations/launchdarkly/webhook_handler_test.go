package launchdarkly

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__LaunchDarklyWebhookHandler__CompareConfig(t *testing.T) {
	handler := &LaunchDarklyWebhookHandler{}

	t.Run("identical events", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{KindFlag}},
			WebhookConfiguration{Events: []string{KindFlag}},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("A superset of B", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{KindFlag, "project"}},
			WebhookConfiguration{Events: []string{KindFlag}},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("A subset of B -> true", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{KindFlag}},
			WebhookConfiguration{Events: []string{KindFlag, "project"}},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("different events -> false", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{"project"}},
			WebhookConfiguration{Events: []string{KindFlag}},
		)
		require.NoError(t, err)
		assert.False(t, equal)
	})
}

func Test__LaunchDarklyWebhookHandler__Merge(t *testing.T) {
	handler := &LaunchDarklyWebhookHandler{}

	t.Run("Merge adds events when requested is superset of current", func(t *testing.T) {
		merged, changed, err := handler.Merge(
			WebhookConfiguration{Events: []string{KindFlag}},
			WebhookConfiguration{Events: []string{KindFlag, "project"}},
		)
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, WebhookConfiguration{Events: []string{KindFlag, "project"}}, merged)
	})

	t.Run("Merge returns current when no change", func(t *testing.T) {
		current := WebhookConfiguration{Events: []string{KindFlag}}
		merged, changed, err := handler.Merge(
			current,
			WebhookConfiguration{Events: []string{KindFlag}},
		)
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, current, merged)
	})
}
