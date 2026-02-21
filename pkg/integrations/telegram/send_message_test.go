package telegram

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

func Test__SendMessage__Setup(t *testing.T) {
	component := &SendMessage{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing chat ID -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"text": "Hello"},
		})

		require.ErrorContains(t, err, "chatId is required")
	})

	t.Run("missing text -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"chatId": "123456789"},
		})

		require.ErrorContains(t, err, "text is required")
	})

	t.Run("invalid parse mode -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"chatId": "123456789", "text": "Hi", "parseMode": "HTML"},
		})

		require.ErrorContains(t, err, "invalid parseMode")
	})

	t.Run("valid configuration -> stores metadata", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.String(), "/getChat")

			return jsonResponse(http.StatusOK, `{
				"ok": true,
				"result": {
					"id": 123456789,
					"type": "private",
					"first_name": "Test"
				}
			}`), nil
		})

		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
			},
			Metadata: metadata,
			Configuration: map[string]any{
				"chatId": "123456789",
				"text":   "Hello, Telegram!",
			},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(SendMessageMetadata)
		require.True(t, ok)
		assert.Equal(t, "123456789", stored.Chat.ID)
	})
}

func Test__SendMessage__Execute(t *testing.T) {
	component := &SendMessage{}

	t.Run("valid configuration -> sends message and emits", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.String(), "/sendMessage")
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Contains(t, req.URL.String(), "test-bot-token")

			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			var payload map[string]any
			require.NoError(t, json.Unmarshal(body, &payload))
			assert.Equal(t, "123456789", payload["chat_id"])
			assert.Equal(t, "Hello, Telegram!", payload["text"])

			return jsonResponse(http.StatusOK, `{
				"ok": true,
				"result": {
					"message_id": 42,
					"from": {"id": 999, "is_bot": true, "first_name": "TestBot"},
					"chat": {"id": 123456789, "type": "private"},
					"text": "Hello, Telegram!",
					"date": 1737028800
				}
			}`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": "test-bot-token"},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			Configuration: map[string]any{
				"chatId": "123456789",
				"text":   "Hello, Telegram!",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, "telegram.message.sent", execState.Type)
		require.Len(t, execState.Payloads, 1)

		wrappedPayload := execState.Payloads[0].(map[string]any)
		payload := wrappedPayload["data"].(map[string]any)
		assert.Equal(t, int64(42), payload["message_id"])
		assert.Equal(t, int64(123456789), payload["chat_id"])
		assert.Equal(t, "Hello, Telegram!", payload["text"])
		assert.Equal(t, int64(1737028800), payload["date"])
	})

	t.Run("API failure -> error", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusForbidden, `{"ok": false, "description": "Forbidden"}`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": "test-bot-token"},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			Configuration: map[string]any{
				"chatId": "123456789",
				"text":   "Hello",
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send message")
	})

	t.Run("missing chat ID -> error", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": "test-bot-token"},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			Configuration:  map[string]any{"text": "Hello"},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "chatId is required")
	})
}
