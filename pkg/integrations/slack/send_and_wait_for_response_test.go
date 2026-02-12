package slack

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__SendAndWaitForResponse__Setup(t *testing.T) {
	component := &SendAndWaitForResponse{}

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
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "token-123"},
			},
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"channel": "",
				"message": "hi",
				"buttons": []any{map[string]any{"name": "Approve", "value": "approve"}},
			},
		})

		require.ErrorContains(t, err, "channel is required")
	})

	t.Run("invalid buttons -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "token-123"},
			},
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"channel": "C123",
				"message": "hi",
				"buttons": []any{
					map[string]any{"name": "", "value": "approve"},
				},
			},
		})

		require.ErrorContains(t, err, "button 1: name is required")
	})

	t.Run("valid configuration -> stores channel metadata", func(t *testing.T) {
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

		err := component.Setup(core.SetupContext{
			Integration: integrationCtx,
			Metadata:    metadata,
			Configuration: map[string]any{
				"channel": "C123",
				"message": "choose one",
				"buttons": []any{
					map[string]any{"name": "Approve", "value": "approve"},
					map[string]any{"name": "Reject", "value": "reject"},
				},
			},
		})

		require.NoError(t, err)

		stored, ok := metadata.Metadata.(SendAndWaitForResponseMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.Channel)
		assert.Equal(t, "C123", stored.Channel.ID)
		assert.Equal(t, "general", stored.Channel.Name)
	})
}

func Test__SendAndWaitForResponse__Execute(t *testing.T) {
	component := &SendAndWaitForResponse{}

	t.Run("valid configuration -> sends message and subscribes", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://slack.com/api/chat.postMessage", req.URL.String())
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			payload := ChatPostMessageRequest{}
			require.NoError(t, json.Unmarshal(body, &payload))
			assert.Equal(t, "C123", payload.Channel)
			assert.Equal(t, "Choose", payload.Text)
			require.Len(t, payload.Blocks, 2)

			return jsonResponse(http.StatusOK, `{"ok": true, "ts": "1700000000.000100"}`), nil
		})

		executionID := uuid.New()
		metadata := &contexts.MetadataContext{Metadata: SendAndWaitForResponseMetadata{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": "token-123"},
		}
		requestCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			ID:          executionID,
			Integration: integrationCtx,
			Metadata:    metadata,
			Requests:    requestCtx,
			Configuration: map[string]any{
				"channel": "C123",
				"message": "Choose",
				"buttons": []any{
					map[string]any{"name": "Approve", "value": "approve"},
					map[string]any{"name": "Reject", "value": "reject"},
				},
			},
		})

		require.NoError(t, err)

		stored, ok := metadata.Metadata.(SendAndWaitForResponseMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.MessageTS)
		assert.Equal(t, "1700000000.000100", *stored.MessageTS)
		require.NotNil(t, stored.SubscriptionID)

		require.Len(t, integrationCtx.Subscriptions, 1)
		subConfig, ok := integrationCtx.Subscriptions[0].Configuration.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, sendAndWaitSubscriptionType, subConfig["type"])
		assert.Equal(t, "1700000000.000100", subConfig["message_ts"])
		assert.Equal(t, executionID.String(), subConfig["execution_id"])
	})

	t.Run("timeout configured -> schedules action", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"ok": true, "ts": "1700000000.000101"}`), nil
		})

		timeout := 45
		requestCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			ID:          uuid.New(),
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"botToken": "token-123"}},
			Metadata:    &contexts.MetadataContext{},
			Requests:    requestCtx,
			Configuration: map[string]any{
				"channel": "C123",
				"message": "Choose",
				"timeout": timeout,
				"buttons": []any{
					map[string]any{"name": "Approve", "value": "approve"},
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, sendAndWaitActionTimeout, requestCtx.Action)
		assert.Equal(t, 45*time.Second, requestCtx.Duration)
	})
}

func Test__SendAndWaitForResponse__HandleAction(t *testing.T) {
	component := &SendAndWaitForResponse{}

	t.Run("button click -> emits received and cleans subscription", func(t *testing.T) {
		subscriptionID := uuid.New()
		subscriptionIDStr := subscriptionID.String()
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Subscriptions: []contexts.Subscription{
				{ID: subscriptionID, Configuration: map[string]any{"type": sendAndWaitSubscriptionType}},
			},
		}

		err := component.HandleAction(core.ActionContext{
			Name:           sendAndWaitActionButtonClick,
			Integration:    integrationCtx,
			Metadata:       &contexts.MetadataContext{Metadata: SendAndWaitForResponseMetadata{SubscriptionID: &subscriptionIDStr}},
			ExecutionState: execState,
			Parameters:     map[string]any{"value": "approve"},
			Configuration: map[string]any{
				"channel": "C123",
				"message": "Choose",
				"buttons": []any{
					map[string]any{"name": "Approve", "value": "approve"},
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, sendAndWaitChannelReceived, execState.Channel)
		assert.Equal(t, "slack.button.clicked", execState.Type)
		require.Len(t, execState.Payloads, 1)

		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "approve", payload["value"])
		assert.NotNil(t, payload["clicked_at"])
		assert.Empty(t, integrationCtx.Subscriptions)
	})

	t.Run("timeout -> emits timeout and cleans subscription", func(t *testing.T) {
		subscriptionID := uuid.New()
		subscriptionIDStr := subscriptionID.String()
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Subscriptions: []contexts.Subscription{
				{ID: subscriptionID, Configuration: map[string]any{"type": sendAndWaitSubscriptionType}},
			},
		}

		err := component.HandleAction(core.ActionContext{
			Name:           sendAndWaitActionTimeout,
			Integration:    integrationCtx,
			Metadata:       &contexts.MetadataContext{Metadata: SendAndWaitForResponseMetadata{SubscriptionID: &subscriptionIDStr}},
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.Equal(t, sendAndWaitChannelTimeout, execState.Channel)
		assert.Equal(t, "slack.button.timeout", execState.Type)
		require.Len(t, execState.Payloads, 1)

		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.NotNil(t, payload["timeout_at"])
		assert.Empty(t, integrationCtx.Subscriptions)
	})

	t.Run("invalid button value -> error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name:           sendAndWaitActionButtonClick,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Parameters:     map[string]any{"value": "invalid"},
			Configuration: map[string]any{
				"channel": "C123",
				"message": "Choose",
				"buttons": []any{
					map[string]any{"name": "Approve", "value": "approve"},
				},
			},
		})

		require.ErrorContains(t, err, "button value not allowed")
	})
}
