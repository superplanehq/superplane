package discord

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__SendTextMessage__Setup(t *testing.T) {
	component := &SendTextMessage{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{},
			Metadata:        &contexts.MetadataContext{},
			Configuration:   "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("no content or embed -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{},
			Metadata:        &contexts.MetadataContext{},
			Configuration:   map[string]any{},
		})

		require.ErrorContains(t, err, "either content or embed")
	})

	t.Run("invalid embed color -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{},
			Metadata:        &contexts.MetadataContext{},
			Configuration: map[string]any{
				"embedTitle": "Title",
				"embedColor": "not-a-color",
			},
		})

		require.ErrorContains(t, err, "invalid embed color")
	})

	t.Run("valid content -> stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{},
			Metadata:        metadata,
			Configuration:   map[string]any{"content": "Hello, Discord!"},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(SendTextMessageMetadata)
		require.True(t, ok)
		assert.False(t, stored.HasEmbed)
	})

	t.Run("valid embed -> stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{},
			Metadata:        metadata,
			Configuration: map[string]any{
				"embedTitle":       "My Embed",
				"embedDescription": "A description",
			},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(SendTextMessageMetadata)
		require.True(t, ok)
		assert.True(t, stored.HasEmbed)
	})
}

func Test__SendTextMessage__Execute(t *testing.T) {
	component := &SendTextMessage{}

	t.Run("valid configuration -> sends message and emits", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.String(), "discord.com/api/webhooks")
			assert.Equal(t, http.MethodPost, req.Method)

			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			var payload ExecuteWebhookRequest
			require.NoError(t, json.Unmarshal(body, &payload))
			assert.Equal(t, "Hello, Discord!", payload.Content)

			return jsonResponse(http.StatusOK, `{
				"id": "1234567890",
				"type": 0,
				"content": "Hello, Discord!",
				"channel_id": "111222333",
				"author": {"id": "999888777", "username": "Webhook", "bot": true},
				"timestamp": "2025-01-16T12:00:00.000Z"
			}`), nil
		})

		webhookURL := "https://discord.com/api/webhooks/123456789/abc-token"
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{"webhookUrl": webhookURL},
		}

		err := component.Execute(core.ExecutionContext{
			AppInstallation: appCtx,
			ExecutionState:  execState,
			Configuration:   map[string]any{"content": "Hello, Discord!"},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, "discord.message.sent", execState.Type)
		require.Len(t, execState.Payloads, 1)

		payload := execState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		assert.Equal(t, "1234567890", data["id"])
		assert.Equal(t, "Hello, Discord!", data["content"])
	})

	t.Run("webhook failure -> error", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusBadRequest, `{"message": "Invalid Webhook Token"}`), nil
		})

		webhookURL := "https://discord.com/api/webhooks/123456789/invalid-token"
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{"webhookUrl": webhookURL},
		}

		err := component.Execute(core.ExecutionContext{
			AppInstallation: appCtx,
			ExecutionState:  execState,
			Configuration:   map[string]any{"content": "Hello"},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send message")
	})
}
