package teams

import (
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnMention__Setup(t *testing.T) {
	trigger := &OnMention{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("empty channel -> subscribes without channel metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      metadata,
			Configuration: map[string]any{"channel": ""},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)

		subConfig, ok := integrationCtx.Subscriptions[0].Configuration.(SubscriptionConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"mention"}, subConfig.EventTypes)

		stored, ok := metadata.Metadata.(OnMentionMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.AppSubscriptionID)
		assert.Nil(t, stored.Channel)
	})

	t.Run("same channel -> no new subscription", func(t *testing.T) {
		subscriptionID := uuid.NewString()
		metadata := &contexts.MetadataContext{Metadata: OnMentionMetadata{
			AppSubscriptionID: &subscriptionID,
			Channel:           &ChannelMetadata{ID: "19:channel123", Name: "19:channel123"},
		}}

		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      metadata,
			Configuration: map[string]any{"channel": "19:channel123"},
		})

		require.NoError(t, err)
		assert.Empty(t, integrationCtx.Subscriptions)
	})

	t.Run("with channel -> subscribes and stores channel metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      metadata,
			Configuration: map[string]any{"channel": "19:abc123"},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)

		subConfig, ok := integrationCtx.Subscriptions[0].Configuration.(SubscriptionConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"mention"}, subConfig.EventTypes)

		stored, ok := metadata.Metadata.(OnMentionMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.AppSubscriptionID)
		require.NotNil(t, stored.Channel)
		assert.Equal(t, "19:abc123", stored.Channel.ID)
	})
}

func Test__OnMention__OnIntegrationMessage(t *testing.T) {
	trigger := &OnMention{}

	t.Run("channel mismatch -> ignore", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"channel": "19:channel-A"},
			Message: map[string]any{
				"conversation": map[string]any{"id": "19:channel-B"},
				"text":         "hello",
			},
			Logger: logrus.NewEntry(logrus.New()),
			Events: events,
		})

		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("channel match -> emit", func(t *testing.T) {
		message := map[string]any{
			"conversation": map[string]any{"id": "19:channel-A"},
			"text":         "hello @bot",
		}
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"channel": "19:channel-A"},
			Message:       message,
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, "teams.bot.mention", events.Payloads[0].Type)
		assert.Equal(t, message, events.Payloads[0].Data)
	})

	t.Run("no channel configured -> emit", func(t *testing.T) {
		message := map[string]any{
			"conversation": map[string]any{"id": "19:channel-A"},
			"text":         "hello @bot",
		}
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{},
			Message:       message,
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, "teams.bot.mention", events.Payloads[0].Type)
		assert.Equal(t, message, events.Payloads[0].Data)
	})
}
