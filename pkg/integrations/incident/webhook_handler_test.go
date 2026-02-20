package incident

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__SigningSecretHash(t *testing.T) {
	assert.Empty(t, SigningSecretHash(""))
	h := SigningSecretHash("whsec_abc")
	require.NotEmpty(t, h)
	assert.Len(t, h, 64) // hex-encoded SHA256
	assert.Equal(t, SigningSecretHash("whsec_abc"), h)
	assert.NotEqual(t, SigningSecretHash("whsec_xyz"), h)
}

func Test__IncidentIOWebhookHandler__CompareConfig(t *testing.T) {
	handler := &IncidentIOWebhookHandler{}

	hashABC := SigningSecretHash("whsec_abc")
	hashXYZ := SigningSecretHash("whsec_xyz")
	hashSame := SigningSecretHash("whsec_same")
	hashK := SigningSecretHash("whsec_k")

	t.Run("identical events and signing secret hash", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecretHash: hashABC},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecretHash: hashABC},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("identical events but different signing secret hash -> false", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecretHash: hashABC},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecretHash: hashXYZ},
		)
		require.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("identical events both empty signing secret hash", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("A superset of B same secret hash", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2, EventIncidentUpdatedV2}, SigningSecretHash: hashSame},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecretHash: hashSame},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("A subset of B -> false", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecretHash: hashK},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2, EventIncidentUpdatedV2}, SigningSecretHash: hashK},
		)
		require.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("same events, A has no hash and B has hash -> true (reuse so URL stays same when user adds secret)", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecretHash: ""},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecretHash: hashXYZ},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("Merge returns current unchanged when both have different non-empty hashes", func(t *testing.T) {
		hashCur := SigningSecretHash("whsec_cur")
		hashReq := SigningSecretHash("whsec_req")
		merged, changed, err := handler.Merge(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecretHash: hashCur},
			WebhookConfiguration{Events: []string{EventIncidentUpdatedV2}, SigningSecretHash: hashReq},
		)
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecretHash: hashCur}, merged)
	})

	t.Run("Merge adds signing secret hash when current has none", func(t *testing.T) {
		hashNew := SigningSecretHash("whsec_new")
		merged, changed, err := handler.Merge(
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecretHash: ""},
			WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecretHash: hashNew},
		)
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, WebhookConfiguration{Events: []string{EventIncidentCreatedV2}, SigningSecretHash: hashNew}, merged)
	})
}
