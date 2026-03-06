package teams

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
			Configuration: map[string]any{"channel": ""},
		})

		require.ErrorContains(t, err, "channel is required")
	})

	t.Run("valid configuration -> stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      metadata,
			Configuration: map[string]any{"channel": "19:channel-123"},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(SendTextMessageMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.Channel)
		assert.Equal(t, "19:channel-123", stored.Channel.ID)
	})
}

func Test__SendTextMessage__Execute(t *testing.T) {
	component := &SendTextMessage{}

	t.Run("missing channel -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Configuration:  map[string]any{"channel": ""},
		})

		require.ErrorContains(t, err, "channel is required")
	})

	t.Run("missing text -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"appId":       "test-app-id",
					"appPassword": "test-password",
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Configuration:  map[string]any{"channel": "19:channel-123", "text": ""},
		})

		require.ErrorContains(t, err, "text is required")
	})

	t.Run("valid configuration -> sends message and emits", func(t *testing.T) {
		requestCount := 0
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			requestCount++

			// First request: token request
			if requestCount == 1 {
				assert.Contains(t, req.URL.String(), "oauth2/v2.0/token")
				return jsonResponse(http.StatusOK, `{
					"access_token": "test-token",
					"token_type": "Bearer",
					"expires_in": 3600
				}`), nil
			}

			// Second request: send activity
			assert.Contains(t, req.URL.String(), "v3/conversations")
			assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))

			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			var activity Activity
			require.NoError(t, json.Unmarshal(body, &activity))
			assert.Equal(t, "message", activity.Type)
			assert.Equal(t, "hello team", activity.Text)

			return jsonResponse(http.StatusOK, `{
				"id": "msg-123",
				"timestamp": "2026-01-16T12:00:00.000Z"
			}`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"appId":       "test-app-id",
				"appPassword": "test-password",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			Configuration:  map[string]any{"channel": "19:channel-123", "text": "hello team"},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, "teams.message.sent", execState.Type)
		require.Len(t, execState.Payloads, 1)

		payload := execState.Payloads[0].(map[string]any)
		assert.Equal(t, "teams.message.sent", payload["type"])

		data := payload["data"].(map[string]any)
		assert.Equal(t, "msg-123", data["id"])
		assert.Equal(t, "19:channel-123", data["conversationId"])
		assert.Equal(t, "hello team", data["text"])
	})
}
