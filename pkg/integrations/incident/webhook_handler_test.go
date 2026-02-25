package incident

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__IncidentIOWebhookHandler__CompareConfig(t *testing.T) {
	handler := &IncidentIOWebhookHandler{}

	t.Run("identical events", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("A superset of B", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2, EventIncidentUpdatedV2}},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("A subset of B -> true (reuse so URL stays same when user adds events)", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2, EventIncidentUpdatedV2}},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("different events -> false", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentUpdatedV2}},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}},
		)
		require.NoError(t, err)
		assert.False(t, equal)
	})
}

func Test__IncidentIOWebhookHandler__Merge(t *testing.T) {
	handler := &IncidentIOWebhookHandler{}

	t.Run("Merge adds events when requested is superset of current", func(t *testing.T) {
		merged, changed, err := handler.Merge(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2, EventIncidentUpdatedV2}},
		)
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, WebhookConfiguration{Events: []string{EventIncidentCreatedV2, EventIncidentUpdatedV2}}, merged)
	})

	t.Run("Merge returns current when no change", func(t *testing.T) {
		current := WebhookConfiguration{Events: []string{EventIncidentCreatedV2}}
		merged, changed, err := handler.Merge(
			current,
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}},
		)
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, current, merged)
	})
}
