package slack

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__SendAndWaitMessage__Setup(t *testing.T) {
	component := &SendAndWaitMessage{}

	t.Run("missing channel -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"channel": ""},
		})

		require.ErrorContains(t, err, "channel is required")
	})

	t.Run("missing buttons -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"channel": "C123", "buttons": []any{}},
		})

		require.ErrorContains(t, err, "at least one button must be configured")
	})

	t.Run("valid configuration -> creates subscription and stores metadata", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"ok": true, "channel": {"id": "C123", "name": "general"}}`), nil
		})

		metadata := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"botToken": "token-123",
			},
		}

		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      metadata,
			Configuration: map[string]any{"channel": "C123", "buttons": []any{map[string]any{"name": "Approve", "value": "approve"}}},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(SendAndWaitMessageMetadata)
		require.True(t, ok)
		assert.Equal(t, "C123", stored.Channel.ID)
		assert.NotNil(t, stored.AppSubscriptionID)
		assert.NotEmpty(t, *stored.AppSubscriptionID)
	})
}

func Test__SendAndWaitMessage__Execute(t *testing.T) {
	component := &SendAndWaitMessage{}

	t.Run("valid configuration -> sends message with blocks and schedules timeout", func(t *testing.T) {
		execID := uuid.New()
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			if req.URL.String() == "https://slack.com/api/chat.postMessage" {
				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				var payload ChatPostMessageRequest
				require.NoError(t, json.Unmarshal(body, &payload))
				assert.Equal(t, "C123", payload.Channel)
				assert.Len(t, payload.Blocks, 2)
				return jsonResponse(http.StatusOK, `{"ok": true, "message": {"ts": "1234.5678"}}`), nil
			}
			return jsonResponse(http.StatusNotFound, `{"ok": false}`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		requests := &contexts.RequestContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": "token-123"},
		}

		err := component.Execute(core.ExecutionContext{
			ID:             execID,
			Integration:    integrationCtx,
			ExecutionState: execState,
			Requests:       requests,
			Configuration: map[string]any{
				"channel": "C123",
				"message": "Approve this?",
				"timeout": 60,
				"buttons": []any{map[string]any{"name": "Approve", "value": "approve"}},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, execID.String(), execState.KVs["execution_id"])
		assert.Equal(t, "timeout", requests.Action)
	})
}

func Test__SendAndWaitMessage__OnIntegrationMessage(t *testing.T) {
	component := &SendAndWaitMessage{}

	t.Run("interaction message -> emits received", func(t *testing.T) {
		execID := uuid.New()
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{"execution_id": execID.String()}}

		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"actions": []any{
					map[string]any{
						"value": "sp_exec:" + execID.String() + ":approve",
					},
				},
			},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				if key == "execution_id" && value == execID.String() {
					return &core.ExecutionContext{
						ExecutionState: execState,
					}, nil
				}
				return nil, nil
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "received", execState.Channel)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)
		assert.Equal(t, "approve", payload["data"].(map[string]any)["value"])
	})
}
