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

	t.Run("missing channel -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{},
			Metadata:        &contexts.MetadataContext{},
			Configuration:   map[string]any{"content": "Hello"},
		})

		require.ErrorContains(t, err, "channel is required")
	})

	t.Run("no content or embed -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{
				Configuration: map[string]any{"botToken": "test-token"},
			},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"channel": "123456789"},
		})

		require.ErrorContains(t, err, "either content or embed")
	})

	t.Run("invalid embed color -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{
				Configuration: map[string]any{"botToken": "test-token"},
			},
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"channel":    "123456789",
				"embedTitle": "Title",
				"embedColor": "not-a-color",
			},
		})

		require.ErrorContains(t, err, "invalid embed color")
	})

	t.Run("valid content -> validates channel and stores metadata", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.String(), "/channels/123456789")
			return jsonResponse(http.StatusOK, `{
				"id": "123456789",
				"name": "general",
				"type": 0
			}`), nil
		})

		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{
				Configuration: map[string]any{"botToken": "test-token"},
			},
			Metadata: metadata,
			Configuration: map[string]any{
				"channel": "123456789",
				"content": "Hello, Discord!",
			},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(SendTextMessageMetadata)
		require.True(t, ok)
		assert.False(t, stored.HasEmbed)
		assert.Equal(t, "123456789", stored.Channel.ID)
		assert.Equal(t, "general", stored.Channel.Name)
	})

	t.Run("valid embed -> stores metadata", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{
				"id": "123456789",
				"name": "general",
				"type": 0
			}`), nil
		})

		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{
				Configuration: map[string]any{"botToken": "test-token"},
			},
			Metadata: metadata,
			Configuration: map[string]any{
				"channel":          "123456789",
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
			assert.Contains(t, req.URL.String(), "/channels/123456789/messages")
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "Bot test-bot-token", req.Header.Get("Authorization"))

			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			var payload CreateMessageRequest
			require.NoError(t, json.Unmarshal(body, &payload))
			assert.Equal(t, "Hello, Discord!", payload.Content)

			return jsonResponse(http.StatusOK, `{
				"id": "1234567890",
				"type": 0,
				"content": "Hello, Discord!",
				"channel_id": "123456789",
				"author": {"id": "999888777", "username": "TestBot", "bot": true},
				"timestamp": "2025-01-16T12:00:00.000Z"
			}`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{"botToken": "test-bot-token"},
		}

		err := component.Execute(core.ExecutionContext{
			AppInstallation: appCtx,
			ExecutionState:  execState,
			Configuration: map[string]any{
				"channel": "123456789",
				"content": "Hello, Discord!",
			},
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

	t.Run("message with embed -> sends correctly", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			var payload CreateMessageRequest
			require.NoError(t, json.Unmarshal(body, &payload))
			assert.Equal(t, "Hello!", payload.Content)
			require.Len(t, payload.Embeds, 1)
			assert.Equal(t, "Test Title", payload.Embeds[0].Title)
			assert.Equal(t, "Test Description", payload.Embeds[0].Description)
			assert.Equal(t, 5793266, payload.Embeds[0].Color) // #5865F2

			return jsonResponse(http.StatusOK, `{
				"id": "1234567890",
				"type": 0,
				"content": "Hello!",
				"channel_id": "123456789",
				"author": {"id": "999888777", "username": "TestBot", "bot": true},
				"timestamp": "2025-01-16T12:00:00.000Z"
			}`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{"botToken": "test-bot-token"},
		}

		err := component.Execute(core.ExecutionContext{
			AppInstallation: appCtx,
			ExecutionState:  execState,
			Configuration: map[string]any{
				"channel":          "123456789",
				"content":          "Hello!",
				"embedTitle":       "Test Title",
				"embedDescription": "Test Description",
				"embedColor":       "#5865F2",
			},
		})

		require.NoError(t, err)
	})

	t.Run("API failure -> error", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusForbidden, `{"message": "Missing Access"}`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{"botToken": "test-bot-token"},
		}

		err := component.Execute(core.ExecutionContext{
			AppInstallation: appCtx,
			ExecutionState:  execState,
			Configuration: map[string]any{
				"channel": "123456789",
				"content": "Hello",
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send message")
	})

	t.Run("missing channel -> error", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{"botToken": "test-bot-token"},
		}

		err := component.Execute(core.ExecutionContext{
			AppInstallation: appCtx,
			ExecutionState:  execState,
			Configuration:   map[string]any{"content": "Hello"},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "channel is required")
	})
}
