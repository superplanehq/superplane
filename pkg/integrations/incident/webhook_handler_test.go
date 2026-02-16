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

	t.Run("A subset of B -> false", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2, EventIncidentUpdatedV2}},
		)
		require.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("Merge returns current unchanged", func(t *testing.T) {
		merged, changed, err := handler.Merge(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}},
			WebhookConfiguration{Events: []string{EventIncidentUpdatedV2}},
		)
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, WebhookConfiguration{Events: []string{EventIncidentCreatedV2}}, merged)
	})
}
