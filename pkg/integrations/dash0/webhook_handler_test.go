package dash0

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

// noopWebhookContext is a minimal test double for core.WebhookContext.
type noopWebhookContext struct{}

// GetID returns an empty test ID.
func (w *noopWebhookContext) GetID() string {
	return ""
}

// GetURL returns an empty webhook URL for tests.
func (w *noopWebhookContext) GetURL() string {
	return ""
}

// GetSecret returns an empty webhook secret for tests.
func (w *noopWebhookContext) GetSecret() ([]byte, error) {
	return []byte{}, nil
}

// GetMetadata returns no metadata for tests.
func (w *noopWebhookContext) GetMetadata() any {
	return map[string]any{}
}

// GetConfiguration returns no configuration for tests.
func (w *noopWebhookContext) GetConfiguration() any {
	return map[string]any{}
}

// SetSecret is a no-op for tests.
func (w *noopWebhookContext) SetSecret(secret []byte) error {
	return nil
}

// Test__Dash0WebhookHandler__CompareConfig verifies webhook configuration matching behavior.
func Test__Dash0WebhookHandler__CompareConfig(t *testing.T) {
	handler := &Dash0WebhookHandler{}

	t.Run("superset config matches subset config", func(t *testing.T) {
		matches, err := handler.CompareConfig(
			map[string]any{"eventTypes": []string{"fired", "resolved"}},
			map[string]any{"eventTypes": []string{"fired"}},
		)
		require.NoError(t, err)
		assert.True(t, matches)
	})

	t.Run("subset config does not match superset config", func(t *testing.T) {
		matches, err := handler.CompareConfig(
			map[string]any{"eventTypes": []string{"resolved"}},
			map[string]any{"eventTypes": []string{"fired", "resolved"}},
		)
		require.NoError(t, err)
		assert.False(t, matches)
	})

	t.Run("normalization maps aliases", func(t *testing.T) {
		matches, err := handler.CompareConfig(
			map[string]any{"eventTypes": []string{"fire", "resolve"}},
			map[string]any{"eventTypes": []string{"fired", "resolved"}},
		)
		require.NoError(t, err)
		assert.True(t, matches)
	})
}

// Test__Dash0WebhookHandler__SetupCleanup validates no-op lifecycle behavior.
func Test__Dash0WebhookHandler__SetupCleanup(t *testing.T) {
	handler := &Dash0WebhookHandler{}

	ctx := core.WebhookHandlerContext{
		Webhook: &noopWebhookContext{},
	}

	metadata, err := handler.Setup(ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{}, metadata)

	require.NoError(t, handler.Cleanup(ctx))
}

// Test__Dash0WebhookHandler__Merge verifies event-type union behavior for shared webhooks.
func Test__Dash0WebhookHandler__Merge(t *testing.T) {
	handler := &Dash0WebhookHandler{}

	merged, changed, err := handler.Merge(
		map[string]any{"eventTypes": []string{"fired"}},
		map[string]any{"eventTypes": []string{"resolved"}},
	)
	require.NoError(t, err)
	assert.True(t, changed)

	mergedConfig, ok := merged.(OnAlertEventConfiguration)
	require.True(t, ok)
	assert.ElementsMatch(t, []string{"fired", "resolved"}, mergedConfig.EventTypes)

	merged, changed, err = handler.Merge(
		map[string]any{"eventTypes": []string{"fired", "resolved"}},
		map[string]any{"eventTypes": []string{"resolved"}},
	)
	require.NoError(t, err)
	assert.False(t, changed)

	mergedConfig, ok = merged.(OnAlertEventConfiguration)
	require.True(t, ok)
	assert.ElementsMatch(t, []string{"fired", "resolved"}, mergedConfig.EventTypes)
}
