package discord

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetLastMention__Setup(t *testing.T) {
	component := &GetLastMention{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing channel -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "channel is required")
	})

	t.Run("invalid since -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"channel": "123", "since": "invalid-date"},
		})

		require.ErrorContains(t, err, "invalid since")
	})

	t.Run("valid channel -> stores metadata", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "/api/v10/channels/123", req.URL.Path)
			return jsonResponse(http.StatusOK, `{"id":"123","name":"general","type":0}`), nil
		})

		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
			},
			Metadata:      metadata,
			Configuration: map[string]any{"channel": "123"},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(GetLastMentionMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.Channel)
		assert.Equal(t, "123", stored.Channel.ID)
		assert.Equal(t, "general", stored.Channel.Name)
	})

	t.Run("since expression does not fail setup", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "/api/v10/channels/123", req.URL.Path)
			return jsonResponse(http.StatusOK, `{"id":"123","name":"general","type":0}`), nil
		})

		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
			},
			Metadata: metadata,
			Configuration: map[string]any{
				"channel": "123",
				"since":   "{{ $['trigger'].data.since }}",
			},
		})

		require.NoError(t, err)
	})

	t.Run("go timestamp since does not fail setup", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "/api/v10/channels/123", req.URL.Path)
			return jsonResponse(http.StatusOK, `{"id":"123","name":"general","type":0}`), nil
		})

		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
			},
			Metadata: metadata,
			Configuration: map[string]any{
				"channel": "123",
				"since":   "2026-03-16 04:17:08.750328135 +0000 UTC",
			},
		})

		require.NoError(t, err)
	})
}

func Test__GetLastMention__Execute(t *testing.T) {
	component := &GetLastMention{}

	t.Run("returns latest mention", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "/api/v10/channels/123/messages", req.URL.Path)
			return jsonResponse(http.StatusOK, `[
				{
					"id": "2",
					"channel_id": "123",
					"content": "<@bot-1> deploy now",
					"timestamp": "2026-03-10T15:04:05.000Z",
					"author": {"id": "u-1", "username": "pedro", "bot": false},
					"mentions": [{"id": "bot-1", "username": "superplane-bot", "bot": true}]
				},
				{
					"id": "1",
					"channel_id": "123",
					"content": "older",
					"timestamp": "2026-03-10T15:00:00.000Z",
					"author": {"id": "u-2", "username": "john", "bot": false},
					"mentions": []
				}
			]`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
				Metadata: Metadata{
					BotID:    "bot-1",
					Username: "superplane-bot",
				},
			},
			ExecutionState: execState,
			Configuration:  map[string]any{"channel": "123"},
		})

		require.NoError(t, err)
		assert.Equal(t, GetLastMentionOutputChannelFound, execState.Channel)
		assert.Equal(t, GetLastMentionPayloadType, execState.Type)
		require.Len(t, execState.Payloads, 1)

		payload := execState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		_, hasFound := data["found"]
		assert.False(t, hasFound)
		mention := data["mention"].(map[string]any)
		assert.Equal(t, "2", mention["id"])
	})

	t.Run("no mention found -> emits notFound channel", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `[
				{
					"id": "3",
					"channel_id": "123",
					"content": "hello",
					"timestamp": "2026-03-10T15:04:05.000Z",
					"author": {"id": "u-1", "username": "pedro", "bot": false},
					"mentions": []
				}
			]`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
				Metadata: Metadata{
					BotID:    "bot-1",
					Username: "superplane-bot",
				},
			},
			ExecutionState: execState,
			Configuration:  map[string]any{"channel": "123"},
		})

		require.NoError(t, err)
		assert.Equal(t, GetLastMentionOutputChannelNotFound, execState.Channel)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		_, hasFound := data["found"]
		assert.False(t, hasFound)
		_, hasMention := data["mention"]
		assert.False(t, hasMention)
	})

	t.Run("since filters out older mention", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `[
				{
					"id": "2",
					"channel_id": "123",
					"content": "<@bot-1> deploy now",
					"timestamp": "2026-03-10T15:04:05.000Z",
					"author": {"id": "u-1", "username": "pedro", "bot": false},
					"mentions": [{"id": "bot-1", "username": "superplane-bot", "bot": true}]
				}
			]`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
				Metadata: Metadata{
					BotID:    "bot-1",
					Username: "superplane-bot",
				},
			},
			ExecutionState: execState,
			Configuration: map[string]any{
				"channel": "123",
				"since":   "2026-03-10T16:00:00Z",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, GetLastMentionOutputChannelNotFound, execState.Channel)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		_, hasMention := data["mention"]
		assert.False(t, hasMention)
	})

	t.Run("go timestamp since filters out older mention", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `[
				{
					"id": "2",
					"channel_id": "123",
					"content": "<@bot-1> deploy now",
					"timestamp": "2026-03-10T15:04:05.000Z",
					"author": {"id": "u-1", "username": "pedro", "bot": false},
					"mentions": [{"id": "bot-1", "username": "superplane-bot", "bot": true}]
				}
			]`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
				Metadata: Metadata{
					BotID:    "bot-1",
					Username: "superplane-bot",
				},
			},
			ExecutionState: execState,
			Configuration: map[string]any{
				"channel": "123",
				"since":   "2026-03-10 16:00:00 +0000 UTC",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, GetLastMentionOutputChannelNotFound, execState.Channel)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		_, hasMention := data["mention"]
		assert.False(t, hasMention)
	})
}
