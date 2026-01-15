package slack

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
			Configuration:   map[string]any{"channel": ""},
		})

		require.ErrorContains(t, err, "channel is required")
	})

	t.Run("valid configuration -> stores metadata", func(t *testing.T) {
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

		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        metadata,
			Configuration:   map[string]any{"channel": "C123"},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(SendTextMessageMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.Channel)
		assert.Equal(t, "C123", stored.Channel.ID)
		assert.Equal(t, "general", stored.Channel.Name)
	})
}

func Test__SendTextMessage__Execute(t *testing.T) {
	component := &SendTextMessage{}

	t.Run("missing channel -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			AppInstallation: &contexts.AppInstallationContext{},
			ExecutionState:  &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Configuration:   map[string]any{"channel": ""},
		})

		require.ErrorContains(t, err, "channel is required")
	})

	t.Run("valid configuration -> sends message and emits", func(t *testing.T) {
		expectedMessage := map[string]any{"text": "hello"}

		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			if req.URL.String() == "https://slack.com/api/chat.postMessage" {
				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				var payload ChatPostMessageRequest
				require.NoError(t, json.Unmarshal(body, &payload))
				assert.Equal(t, "C123", payload.Channel)
				assert.Equal(t, "hello", payload.Text)
				return jsonResponse(http.StatusOK, `{"ok": true, "message": {"text": "hello"}}`), nil
			}
			return jsonResponse(http.StatusNotFound, `{"ok": false}`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"botToken": "token-123",
			},
		}

		err := component.Execute(core.ExecutionContext{
			AppInstallation: appCtx,
			ExecutionState:  execState,
			Configuration:   map[string]any{"channel": "C123", "text": "hello"},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, "slack.message.sent", execState.Type)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)
		assert.Equal(t, "slack.message.sent", payload["type"])
		assert.Equal(t, expectedMessage, payload["data"])
		assert.NotNil(t, payload["timestamp"])
	})
}
