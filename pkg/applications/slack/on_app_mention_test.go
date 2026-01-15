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

func Test__OnAppMention__Setup(t *testing.T) {
	trigger := &OnAppMention{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			AppInstallation: &contexts.AppInstallationContext{},
			Metadata:        &contexts.MetadataContext{},
			Configuration:   "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("empty channel -> subscribes without channel metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		appCtx := &contexts.AppInstallationContext{}
		err := trigger.Setup(core.TriggerContext{
			AppInstallation: appCtx,
			Metadata:        metadata,
			Configuration:   map[string]any{"channel": ""},
		})

		require.NoError(t, err)
		require.Len(t, appCtx.Subscriptions, 1)

		subConfig, ok := appCtx.Subscriptions[0].Configuration.(SubscriptionConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"app_mention"}, subConfig.EventTypes)

		stored, ok := metadata.Metadata.(AppMentionMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.AppSubscriptionID)
		assert.Nil(t, stored.Channel)
	})

	t.Run("metadata already set -> no-op", func(t *testing.T) {
		subscriptionID := uuid.NewString()
		metadata := &contexts.MetadataContext{Metadata: AppMentionMetadata{
			AppSubscriptionID: &subscriptionID,
			Channel:           &ChannelMetadata{ID: "C123", Name: "general"},
		}}

		appCtx := &contexts.AppInstallationContext{}
		err := trigger.Setup(core.TriggerContext{
			AppInstallation: appCtx,
			Metadata:        metadata,
			Configuration:   map[string]any{"channel": "C123"},
		})

		require.NoError(t, err)
		assert.Empty(t, appCtx.Subscriptions)
	})

	t.Run("valid configuration -> subscribes and stores metadata", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://slack.com/api/conversations.info", req.URL.Scheme+"://"+req.URL.Host+req.URL.Path)
			assert.Equal(t, "C123", req.URL.Query().Get("channel"))
			return jsonResponse(http.StatusOK, `{"ok": true, "channel": {"id": "C123", "name": "general"}}`), nil
		})

		metadata := &contexts.MetadataContext{}
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"botToken": "token-123",
			},
		}

		err := trigger.Setup(core.TriggerContext{
			AppInstallation: appCtx,
			Metadata:        metadata,
			Configuration:   map[string]any{"channel": "C123"},
		})

		require.NoError(t, err)
		require.Len(t, appCtx.Subscriptions, 1)

		subConfig, ok := appCtx.Subscriptions[0].Configuration.(SubscriptionConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"app_mention"}, subConfig.EventTypes)

		stored, ok := metadata.Metadata.(AppMentionMetadata)
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
		metadata := &contexts.MetadataContext{Metadata: AppMentionMetadata{
			AppSubscriptionID: &subscriptionID,
		}}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"botToken": "token-123",
			},
		}

		err := trigger.Setup(core.TriggerContext{
			AppInstallation: appCtx,
			Metadata:        metadata,
			Configuration:   map[string]any{"channel": "C123"},
		})

		require.NoError(t, err)
		require.Len(t, appCtx.Subscriptions, 1)

		stored, ok := metadata.Metadata.(AppMentionMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.AppSubscriptionID)
		require.NotNil(t, stored.Channel)
		assert.Equal(t, "C123", stored.Channel.ID)
		assert.Equal(t, "general", stored.Channel.Name)
	})
}

func Test__OnAppMention__OnAppMessage(t *testing.T) {
	trigger := &OnAppMention{}

	t.Run("channel mismatch -> ignore", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnAppMessage(core.AppMessageContext{
			Configuration: map[string]any{"channel": "C999"},
			Message:       map[string]any{"channel": "C123"},
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("channel match -> emit", func(t *testing.T) {
		message := map[string]any{"channel": "C123", "text": "hi"}
		events := &contexts.EventContext{}
		err := trigger.OnAppMessage(core.AppMessageContext{
			Configuration: map[string]any{"channel": "C123"},
			Message:       message,
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, "slack.app.mention", events.Payloads[0].Type)
		assert.Equal(t, message, events.Payloads[0].Data)
	})

	t.Run("no channel configured -> emit", func(t *testing.T) {
		message := map[string]any{"channel": "C123", "text": "hi"}
		events := &contexts.EventContext{}
		err := trigger.OnAppMessage(core.AppMessageContext{
			Configuration: map[string]any{},
			Message:       message,
			Logger:        logrus.NewEntry(logrus.New()),
			Events:        events,
		})

		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, "slack.app.mention", events.Payloads[0].Type)
		assert.Equal(t, message, events.Payloads[0].Data)
	})
}
