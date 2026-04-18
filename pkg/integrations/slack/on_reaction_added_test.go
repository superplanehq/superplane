package slack

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnReactionAdded__Setup(t *testing.T) {
	trigger := &OnReactionAdded{}

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
			Configuration: map[string]any{"channel": "", "reaction": ""},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)

		subConfig, ok := integrationCtx.Subscriptions[0].Configuration.(SubscriptionConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"reaction_added"}, subConfig.EventTypes)

		stored, ok := metadata.Metadata.(OnReactionAddedMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.AppSubscriptionID)
		assert.Nil(t, stored.Channel)
	})

	t.Run("same channel -> no-op", func(t *testing.T) {
		subscriptionID := uuid.NewString()
		metadata := &contexts.MetadataContext{Metadata: OnReactionAddedMetadata{
			AppSubscriptionID: &subscriptionID,
			Channel:           &ChannelMetadata{ID: "C123", Name: "general"},
		}}

		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      metadata,
			Configuration: map[string]any{"channel": "C123", "reaction": ""},
		})

		require.NoError(t, err)
		assert.Empty(t, integrationCtx.Subscriptions)
	})

	t.Run("different channel -> subscribes and stores channel metadata", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://slack.com/api/conversations.info", req.URL.Scheme+"://"+req.URL.Host+req.URL.Path)
			assert.Equal(t, "C456", req.URL.Query().Get("channel"))
			return jsonResponse(http.StatusOK, `{"ok": true, "channel": {"id": "C456", "name": "random"}}`), nil
		})

		subscriptionID := uuid.NewString()
		metadata := &contexts.MetadataContext{
			Metadata: OnReactionAddedMetadata{
				AppSubscriptionID: &subscriptionID,
				Channel:           &ChannelMetadata{ID: "C123", Name: "general"},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"botToken": "token-123",
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      metadata,
			Configuration: map[string]any{"channel": "C456", "reaction": ""},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 0)

		stored, ok := metadata.Metadata.(OnReactionAddedMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.AppSubscriptionID)
		require.NotNil(t, stored.Channel)
		assert.Equal(t, "C456", stored.Channel.ID)
		assert.Equal(t, "random", stored.Channel.Name)
	})

	t.Run("valid configuration -> subscribes and stores metadata", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://slack.com/api/conversations.info", req.URL.Scheme+"://"+req.URL.Host+req.URL.Path)
			assert.Equal(t, "C123", req.URL.Query().Get("channel"))
			return jsonResponse(http.StatusOK, `{"ok": true, "channel": {"id": "C123", "name": "general"}}`), nil
		})

		metadata := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"botToken": "token-123",
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      metadata,
			Configuration: map[string]any{"channel": "C123", "reaction": ""},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)

		subConfig, ok := integrationCtx.Subscriptions[0].Configuration.(SubscriptionConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"reaction_added"}, subConfig.EventTypes)

		stored, ok := metadata.Metadata.(OnReactionAddedMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.AppSubscriptionID)
		require.NotNil(t, stored.Channel)
		assert.Equal(t, "C123", stored.Channel.ID)
		assert.Equal(t, "general", stored.Channel.Name)
	})

	t.Run("existing subscription without channel -> subscribes and stores channel metadata", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://slack.com/api/conversations.info", req.URL.Scheme+"://"+req.URL.Host+req.URL.Path)
			assert.Equal(t, "C123", req.URL.Query().Get("channel"))
			return jsonResponse(http.StatusOK, `{"ok": true, "channel": {"id": "C123", "name": "general"}}`), nil
		})

		subscriptionID := uuid.NewString()
		metadata := &contexts.MetadataContext{Metadata: OnReactionAddedMetadata{
			AppSubscriptionID: &subscriptionID,
		}}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"botToken": "token-123",
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      metadata,
			Configuration: map[string]any{"channel": "C123", "reaction": ""},
		})

		require.NoError(t, err)

		require.Len(t, integrationCtx.Subscriptions, 0)

		stored, ok := metadata.Metadata.(OnReactionAddedMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.AppSubscriptionID)
		require.NotNil(t, stored.Channel)
		assert.Equal(t, "C123", stored.Channel.ID)
		assert.Equal(t, "general", stored.Channel.Name)
	})
}

func Test__OnReactionAdded__OnIntegrationMessage(t *testing.T) {
	trigger := &OnReactionAdded{}

	t.Run("channel mismatch -> ignore", func(t *testing.T) {
		event := map[string]any{
			"reaction": "thumbsup",
			"item": map[string]any{
				"channel": "C123",
			},
		}
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"channel": "C999"},
			Message:       event,
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("channel match -> emit", func(t *testing.T) {
		event := map[string]any{
			"reaction": "thumbsup",
			"item": map[string]any{
				"channel": "C123",
			},
		}
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"channel": "C123"},
			Message:       event,
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, "slack.reaction.added", events.Payloads[0].Type)
		assert.Equal(t, event, events.Payloads[0].Data)
	})

	t.Run("no channel configured -> emit", func(t *testing.T) {
		event := map[string]any{
			"reaction": "thumbsup",
			"item": map[string]any{
				"channel": "C123",
			},
		}
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{},
			Message:       event,
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, "slack.reaction.added", events.Payloads[0].Type)
		assert.Equal(t, event, events.Payloads[0].Data)
	})

	t.Run("reaction mismatch -> ignore", func(t *testing.T) {
		event := map[string]any{
			"reaction": "thumbsup",
			"item": map[string]any{
				"channel": "C123",
			},
		}
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"reaction": "white_check_mark"},
			Message:       event,
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("reaction match -> emit", func(t *testing.T) {
		event := map[string]any{
			"reaction": "thumbsup",
			"item": map[string]any{
				"channel": "C123",
			},
		}
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"reaction": "thumbsup"},
			Message:       event,
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, "slack.reaction.added", events.Payloads[0].Type)
		assert.Equal(t, event, events.Payloads[0].Data)
	})

	t.Run("both channel and reaction match -> emit", func(t *testing.T) {
		event := map[string]any{
			"reaction": "thumbsup",
			"item": map[string]any{
				"channel": "C123",
			},
		}
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"channel": "C123", "reaction": "thumbsup"},
			Message:       event,
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, "slack.reaction.added", events.Payloads[0].Type)
		assert.Equal(t, event, events.Payloads[0].Data)
	})

	t.Run("channel match but reaction mismatch -> ignore", func(t *testing.T) {
		event := map[string]any{
			"reaction": "thumbsup",
			"item": map[string]any{
				"channel": "C123",
			},
		}
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"channel": "C123", "reaction": "white_check_mark"},
			Message:       event,
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})
}

func Test__OnReactionAdded__ExampleData(t *testing.T) {
	trigger := &OnReactionAdded{}
	data := trigger.ExampleData()

	require.NotNil(t, data)
	assert.Equal(t, "reaction_added", data["type"])
	assert.Equal(t, "thumbsup", data["reaction"])
}
