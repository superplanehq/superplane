package incident

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__IncidentIOWebhookHandler__CompareConfig(t *testing.T) {
	handler := &IncidentIOWebhookHandler{}

	t.Run("identical events and signing secret", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecret: "whsec_abc"},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecret: "whsec_abc"},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("identical events but different signing secret -> false", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecret: "whsec_abc"},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecret: "whsec_xyz"},
		)
		require.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("identical events both empty signing secret", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("A superset of B same secret", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2, EventIncidentUpdatedV2}, SigningSecret: "whsec_same"},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecret: "whsec_same"},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("A subset of B -> false", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecret: "whsec_k"},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2, EventIncidentUpdatedV2}, SigningSecret: "whsec_k"},
		)
		require.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("same events, A has no secret and B has secret -> true (reuse so URL stays same when user adds secret)", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecret: ""},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecret: "whsec_xyz"},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("Merge returns current unchanged when both have different non-empty secrets", func(t *testing.T) {
		merged, changed, err := handler.Merge(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecret: "whsec_cur"},
			WebhookConfiguration{Events: []string{EventIncidentUpdatedV2}, SigningSecret: "whsec_req"},
		)
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecret: "whsec_cur"}, merged)
	})

	t.Run("Merge adds signing secret when current has none", func(t *testing.T) {
		merged, changed, err := handler.Merge(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecret: ""},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecret: "whsec_new"},
		)
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecret: "whsec_new"}, merged)
	})
}
