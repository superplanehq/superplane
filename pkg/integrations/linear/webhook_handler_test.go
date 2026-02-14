package linear

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__LinearWebhookHandler__Setup(t *testing.T) {
	handler := &LinearWebhookHandler{}

	t.Run("no-op returns nil metadata", func(t *testing.T) {
		metadata, err := handler.Setup(core.WebhookHandlerContext{})
		require.NoError(t, err)
		assert.Nil(t, metadata)
	})
}

func Test__LinearWebhookHandler__Cleanup(t *testing.T) {
	handler := &LinearWebhookHandler{}

	t.Run("no-op returns nil", func(t *testing.T) {
		err := handler.Cleanup(core.WebhookHandlerContext{})
		require.NoError(t, err)
	})
}

func Test__LinearWebhookHandler__CompareConfig(t *testing.T) {
	handler := &LinearWebhookHandler{}

	t.Run("always returns true", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			map[string]any{"teamId": "t1"},
			map[string]any{"teamId": "t2"},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})
}

func Test__LinearWebhookHandler__Merge(t *testing.T) {
	handler := &LinearWebhookHandler{}

	t.Run("returns current unchanged", func(t *testing.T) {
		current := map[string]any{"key": "value"}
		result, changed, err := handler.Merge(current, map[string]any{})
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, current, result)
	})
}
