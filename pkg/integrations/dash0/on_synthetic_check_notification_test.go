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

	t.Run("valid synthetic.alert.ongoing event with matching status -> emits event", func(t *testing.T) {
		events := &contexts.EventContext{}
		message := map[string]any{
			"type": "synthetic.alert.ongoing",
			"data": map[string]any{
				"issue": map[string]any{
					"id":              "issue-123",
					"issueIdentifier": "synthetic-check-1",
					"status":          "critical",
					"summary":         "Check failed",
					"url":             "https://example.com/issue",
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
		assert.Equal(t, "issue-123", payload.Issue.ID)
	})

	t.Run("test event type -> ignored", func(t *testing.T) {
		events := &contexts.EventContext{}
		message := map[string]any{
			"type": "test",
			"data": map[string]any{
				"issue": map[string]any{
					"status": "critical",
				},
			},
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

	t.Run("unsupported event type -> ignored", func(t *testing.T) {
		events := &contexts.EventContext{}
		message := map[string]any{
			"type": "synthetic.alert.resolved",
			"data": map[string]any{
				"issue": map[string]any{
					"status": "critical",
				},
			},
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

	t.Run("event without issue -> ignored", func(t *testing.T) {
		events := &contexts.EventContext{}
		message := map[string]any{
			"type": "synthetic.alert.ongoing",
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

	t.Run("status not in configured statuses -> ignored", func(t *testing.T) {
		events := &contexts.EventContext{}
		message := map[string]any{
			"type": "synthetic.alert.ongoing",
			"data": map[string]any{
				"issue": map[string]any{
					"status": "closed",
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

	t.Run("multiple configured statuses -> emits when matching", func(t *testing.T) {
		events := &contexts.EventContext{}
		message := map[string]any{
			"type": "synthetic.alert.ongoing",
			"data": map[string]any{
				"issue": map[string]any{
					"status": "degraded",
				},
			},
		}

		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message:       message,
			Configuration: map[string]any{"statuses": []string{"critical", "degraded", "closed"}},
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
		assert.Equal(t, "dash0.syntheticCheckNotification", events.Payloads[0].Type)
	})

	t.Run("invalid configuration -> returns error", func(t *testing.T) {
		events := &contexts.EventContext{}
		message := map[string]any{
			"type": "synthetic.alert.ongoing",
			"data": map[string]any{
				"issue": map[string]any{
					"status": "critical",
				},
			},
		}

		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message:       message,
			Configuration: "invalid",
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode configuration")
	})

	t.Run("invalid message format -> returns error", func(t *testing.T) {
		events := &contexts.EventContext{}
		message := "invalid message"

		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message:       message,
			Configuration: map[string]any{"statuses": []string{"critical"}},
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode synthetic check notification event")
	})
}
