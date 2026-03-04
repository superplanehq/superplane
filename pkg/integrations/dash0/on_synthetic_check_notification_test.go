package dash0

import (
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnSyntheticCheckNotification__Setup(t *testing.T) {
	trigger := &OnSyntheticCheckNotification{}

	t.Run("no previous subscription -> subscribes and stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		integration := &contexts.IntegrationContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration: integration,
			Metadata:    metadata,
		})

		require.NoError(t, err)
		require.Len(t, integration.Subscriptions, 1)

		stored, ok := metadata.Metadata.(OnSyntheticCheckNotificationMetadata)
		require.True(t, ok)
		require.NotEmpty(t, stored.SubscriptionID)
	})

	t.Run("subscription already exists -> no-op", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: OnSyntheticCheckNotificationMetadata{SubscriptionID: uuid.NewString()},
		}
		integration := &contexts.IntegrationContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration: integration,
			Metadata:    metadata,
		})

		require.NoError(t, err)
		require.Empty(t, integration.Subscriptions)
	})
}

func Test__OnSyntheticCheckNotification__OnIntegrationMessage(t *testing.T) {
	trigger := &OnSyntheticCheckNotification{}

	t.Run("emits event for matching status with synthetic check labels", func(t *testing.T) {
		events := &contexts.EventContext{}
		message := map[string]any{
			"type": "alert.ongoing",
			"data": map[string]any{
				"issue": map[string]any{
					"status":  "critical",
					"summary": "Synthetic check failed",
					"labels": []any{
						[]any{"0", map[string]any{"key": "dash0.resource.type", "value": map[string]any{"stringValue": "synthetic"}}},
						[]any{"1", map[string]any{"key": "dash0.synthetic_check.attempt_id", "value": map[string]any{"stringValue": "73768e2c"}}},
						[]any{"2", map[string]any{"key": "dash0.synthetic_check.id", "value": map[string]any{"stringValue": "api-health"}}},
					},
				},
			},
		}

		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message:       message,
			Configuration: map[string]any{"statuses": []string{"critical", "degraded"}},
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
		assert.Equal(t, "dash0.syntheticCheckNotification", events.Payloads[0].Type)

		payload, ok := events.Payloads[0].Data.(SyntheticCheckNotificationData)
		require.True(t, ok)
		require.NotNil(t, payload.Issue)
		assert.Equal(t, "critical", payload.Issue.Status)
		assert.Len(t, payload.Issue.Labels, 3)
	})

	t.Run("ignores event with non-matching status", func(t *testing.T) {
		events := &contexts.EventContext{}
		message := map[string]any{
			"type": "alert.ongoing",
			"data": map[string]any{
				"issue": map[string]any{
					"status": "closed",
					"labels": []any{
						[]any{"0", map[string]any{"key": "dash0.resource.type", "value": map[string]any{"stringValue": "synthetic"}}},
					},
				},
			},
		}

		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message:       message,
			Configuration: map[string]any{"statuses": []string{"critical", "degraded"}},
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		require.Empty(t, events.Payloads)
	})

	t.Run("ignores test event", func(t *testing.T) {
		events := &contexts.EventContext{}
		message := map[string]any{
			"type": "test",
			"data": map[string]any{},
		}

		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message:       message,
			Configuration: map[string]any{"statuses": []string{"critical"}},
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		require.Empty(t, events.Payloads)
	})

	t.Run("ignores event without issue", func(t *testing.T) {
		events := &contexts.EventContext{}
		message := map[string]any{
			"type": "alert.ongoing",
			"data": map[string]any{},
		}

		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message:       message,
			Configuration: map[string]any{"statuses": []string{"critical"}},
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		require.Empty(t, events.Payloads)
	})
}

func Test__NormalizeSyntheticCheckLabels(t *testing.T) {
	t.Run("normalizes tuple-format labels", func(t *testing.T) {
		raw := []any{
			[]any{"0", map[string]any{"key": "dash0.resource.type", "value": map[string]any{"stringValue": "synthetic"}}},
			[]any{"1", map[string]any{"key": "dash0.synthetic_check.attempt_id", "value": map[string]any{"stringValue": "73768e2c"}}},
		}

		labels := normalizeSyntheticCheckLabels(raw)
		require.Len(t, labels, 2)
		assert.Equal(t, "dash0.resource.type", labels[0].Key)
		assert.Equal(t, "synthetic", labels[0].Value)
		assert.Equal(t, "dash0.synthetic_check.attempt_id", labels[1].Key)
		assert.Equal(t, "73768e2c", labels[1].Value)
	})

	t.Run("handles empty labels", func(t *testing.T) {
		labels := normalizeSyntheticCheckLabels([]any{})
		assert.Empty(t, labels)
	})

	t.Run("handles nil labels", func(t *testing.T) {
		labels := normalizeSyntheticCheckLabels(nil)
		assert.Empty(t, labels)
	})

	t.Run("skips malformed entries", func(t *testing.T) {
		raw := []any{
			"not-an-array",
			[]any{"0"},
			[]any{"0", "not-a-map"},
			[]any{"0", map[string]any{"key": "valid.key", "value": map[string]any{"stringValue": "valid"}}},
		}

		labels := normalizeSyntheticCheckLabels(raw)
		require.Len(t, labels, 1)
		assert.Equal(t, "valid.key", labels[0].Key)
		assert.Equal(t, "valid", labels[0].Value)
	})
}
