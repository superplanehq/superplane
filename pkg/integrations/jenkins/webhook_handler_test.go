package jenkins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__WebhookHandler__CompareConfig(t *testing.T) {
	handler := &JenkinsWebhookHandler{}

	t.Run("always returns true", func(t *testing.T) {
		result, err := handler.CompareConfig(WebhookConfiguration{}, WebhookConfiguration{})
		require.NoError(t, err)
		assert.True(t, result)
	})
}

func Test__WebhookHandler__Merge(t *testing.T) {
	handler := &JenkinsWebhookHandler{}

	t.Run("returns current unchanged", func(t *testing.T) {
		current := WebhookConfiguration{}
		requested := WebhookConfiguration{}

		merged, changed, err := handler.Merge(current, requested)
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, current, merged)
	})
}

func Test__WebhookHandler__Setup(t *testing.T) {
	handler := &JenkinsWebhookHandler{}

	t.Run("returns nil metadata", func(t *testing.T) {
		metadata, err := handler.Setup(core.WebhookHandlerContext{})
		require.NoError(t, err)
		assert.Nil(t, metadata)
	})
}

func Test__WebhookHandler__Cleanup(t *testing.T) {
	handler := &JenkinsWebhookHandler{}

	t.Run("no-op", func(t *testing.T) {
		err := handler.Cleanup(core.WebhookHandlerContext{})
		require.NoError(t, err)
	})
}
