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

func Test__WaitForButtonClick__Setup(t *testing.T) {
	component := &WaitForButtonClick{}

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

	t.Run("missing message -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "token-123"},
			},
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"channel": "C123",
				"message": "",
			},
		})

		require.ErrorContains(t, err, "message is required")
	})

	t.Run("no buttons -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "token-123"},
			},
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"channel": "C123",
				"message": "Choose an option",
				"buttons": []any{},
			},
		})

		require.ErrorContains(t, err, "at least one button is required")
	})

	t.Run("too many buttons -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "token-123"},
			},
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"channel": "C123",
				"message": "Choose an option",
				"buttons": []any{
					map[string]any{"name": "1", "value": "1"},
					map[string]any{"name": "2", "value": "2"},
					map[string]any{"name": "3", "value": "3"},
					map[string]any{"name": "4", "value": "4"},
					map[string]any{"name": "5", "value": "5"},
				},
			},
		})

		require.ErrorContains(t, err, "maximum of 4 buttons allowed")
	})

	t.Run("valid configuration -> stores metadata", func(t *testing.T) {
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
				"message": "Choose an option",
				"buttons": []any{
					map[string]any{"name": "Approve", "value": "approve"},
					map[string]any{"name": "Reject", "value": "reject"},
				},
			},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(WaitForButtonClickMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.Channel)
		assert.Equal(t, "C123", stored.Channel.ID)
		assert.Equal(t, "general", stored.Channel.Name)
	})
}

func Test__WaitForButtonClick__Execute(t *testing.T) {
	component := &WaitForButtonClick{}

	t.Run("missing channel -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"channel": "",
			},
		})

		require.ErrorContains(t, err, "channel is required")
	})

	t.Run("valid configuration -> sends message with buttons", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			if req.URL.String() == "https://slack.com/api/chat.postMessage" {
				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				var payload ChatPostMessageRequest
				require.NoError(t, json.Unmarshal(body, &payload))
				assert.Equal(t, "C123", payload.Channel)
				assert.Equal(t, "Choose an option", payload.Text)
				require.Len(t, payload.Blocks, 2) // text section + actions
				return jsonResponse(http.StatusOK, `{"ok": true, "ts": "1234567890.123456"}`), nil
			}
			if req.URL.Scheme+"://"+req.URL.Host+req.URL.Path == "https://slack.com/api/conversations.info" {
				assert.Equal(t, "C123", req.URL.Query().Get("channel"))
				return jsonResponse(http.StatusOK, `{"ok": true, "channel": {"id": "C123", "name": "general"}}`), nil
			}
			return jsonResponse(http.StatusNotFound, `{"ok": false}`), nil
		})

		executionID := uuid.New()
		metadata := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"botToken": "token-123",
			},
		}
		requestsCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			ID:          executionID,
			Integration: integrationCtx,
			Metadata:    metadata,
			Requests:    requestsCtx,
			Configuration: map[string]any{
				"channel": "C123",
				"message": "Choose an option",
				"buttons": []any{
					map[string]any{"name": "Approve", "value": "approve"},
					map[string]any{"name": "Reject", "value": "reject"},
				},
			},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(WaitForButtonClickMetadata)
		require.True(t, ok)
		assert.NotNil(t, stored.MessageTS)
		assert.Equal(t, "1234567890.123456", *stored.MessageTS)
		assert.NotNil(t, stored.AppSubscriptionID)
	})

	t.Run("with timeout -> schedules timeout action", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			if req.URL.String() == "https://slack.com/api/chat.postMessage" {
				return jsonResponse(http.StatusOK, `{"ok": true, "ts": "1234567890.123456"}`), nil
			}
			if req.URL.Scheme+"://"+req.URL.Host+req.URL.Path == "https://slack.com/api/conversations.info" {
				assert.Equal(t, "C123", req.URL.Query().Get("channel"))
				return jsonResponse(http.StatusOK, `{"ok": true, "channel": {"id": "C123", "name": "general"}}`), nil
			}
			return jsonResponse(http.StatusNotFound, `{"ok": false}`), nil
		})

		executionID := uuid.New()
		metadata := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"botToken": "token-123",
			},
		}
		requestsCtx := &contexts.RequestContext{}
		timeout := 60

		err := component.Execute(core.ExecutionContext{
			ID:          executionID,
			Integration: integrationCtx,
			Metadata:    metadata,
			Requests:    requestsCtx,
			Configuration: map[string]any{
				"channel": "C123",
				"message": "Choose an option",
				"timeout": timeout,
				"buttons": []any{
					map[string]any{"name": "Approve", "value": "approve"},
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, ActionTimeout, requestsCtx.Action)
		assert.NotZero(t, requestsCtx.Duration)
	})
}

func Test__WaitForButtonClick__HandleAction(t *testing.T) {
	component := &WaitForButtonClick{}

	t.Run("button click -> emits received event and cleans up subscription", func(t *testing.T) {
		subscriptionID := uuid.New()
		subscriptionIDStr := subscriptionID.String()
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Subscriptions: []contexts.Subscription{
				{ID: subscriptionID, Configuration: map[string]any{"type": "button_click"}},
			},
		}
		metadata := &contexts.MetadataContext{
			Metadata: WaitForButtonClickMetadata{
				AppSubscriptionID: &subscriptionIDStr,
			},
		}

		err := component.HandleAction(core.ActionContext{
			Name:        ActionButtonClick,
			Metadata:    metadata,
			Integration: integrationCtx,
			Parameters: map[string]any{
				"value": "approve",
			},
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.Equal(t, ChannelReceived, execState.Channel)
		assert.Equal(t, "slack.button.clicked", execState.Type)
		require.Len(t, execState.Payloads, 1)
		wrappedPayload := execState.Payloads[0].(map[string]any)
		payload := wrappedPayload["data"].(map[string]any)
		assert.Equal(t, "approve", payload["value"])
		assert.NotNil(t, payload["clicked_at"])

		// Note: Subscriptions are no longer manually cleaned up.
		// They are automatically cleaned up when the node is deleted.
	})

	t.Run("button click with clicker info -> emits received event with clicker", func(t *testing.T) {
		subscriptionID := uuid.New()
		subscriptionIDStr := subscriptionID.String()
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Subscriptions: []contexts.Subscription{
				{ID: subscriptionID, Configuration: map[string]any{"type": "button_click"}},
			},
		}
		metadata := &contexts.MetadataContext{
			Metadata: WaitForButtonClickMetadata{
				AppSubscriptionID: &subscriptionIDStr,
			},
		}

		err := component.HandleAction(core.ActionContext{
			Name:        ActionButtonClick,
			Metadata:    metadata,
			Integration: integrationCtx,
			Parameters: map[string]any{
				"value": "approve",
				"clicked_by": map[string]any{
					"id":       "U123",
					"username": "pedro",
				},
			},
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.Equal(t, ChannelReceived, execState.Channel)
		assert.Equal(t, "slack.button.clicked", execState.Type)
		require.Len(t, execState.Payloads, 1)
		wrappedPayload := execState.Payloads[0].(map[string]any)
		payload := wrappedPayload["data"].(map[string]any)
		clickedBy := payload["clicked_by"].(map[string]any)
		assert.Equal(t, "U123", clickedBy["id"])
		assert.Equal(t, "pedro", clickedBy["username"])
	})

	t.Run("timeout -> emits timeout event and cleans up subscription", func(t *testing.T) {
		subscriptionID := uuid.New()
		subscriptionIDStr := subscriptionID.String()
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Subscriptions: []contexts.Subscription{
				{ID: subscriptionID, Configuration: map[string]any{"type": "button_click"}},
			},
		}
		metadata := &contexts.MetadataContext{
			Metadata: WaitForButtonClickMetadata{
				AppSubscriptionID: &subscriptionIDStr,
			},
		}

		err := component.HandleAction(core.ActionContext{
			Name:           ActionTimeout,
			Metadata:       metadata,
			Integration:    integrationCtx,
			Parameters:     map[string]any{},
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.Equal(t, ChannelTimeout, execState.Channel)
		assert.Equal(t, "slack.button.timeout", execState.Type)
		require.Len(t, execState.Payloads, 1)
		wrappedPayload := execState.Payloads[0].(map[string]any)
		payload := wrappedPayload["data"].(map[string]any)
		assert.NotNil(t, payload["timeout_at"])

		// Note: Subscriptions are no longer manually cleaned up.
		// They are automatically cleaned up when the node is deleted.
	})

	t.Run("already finished -> no emit", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{
			KVs:      map[string]string{},
			Finished: true,
		}
		metadata := &contexts.MetadataContext{
			Metadata: WaitForButtonClickMetadata{},
		}

		err := component.HandleAction(core.ActionContext{
			Name:     ActionButtonClick,
			Metadata: metadata,
			Parameters: map[string]any{
				"value": "approve",
			},
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.Empty(t, execState.Payloads)
	})
}

func Test__WaitForButtonClick__Cancel(t *testing.T) {
	component := &WaitForButtonClick{}

	t.Run("with active subscription -> no-op", func(t *testing.T) {
		subscriptionID := uuid.New()
		subscriptionIDStr := subscriptionID.String()
		integrationCtx := &contexts.IntegrationContext{
			Subscriptions: []contexts.Subscription{
				{ID: subscriptionID, Configuration: map[string]any{"type": "button_click"}},
			},
		}
		metadata := &contexts.MetadataContext{
			Metadata: WaitForButtonClickMetadata{
				AppSubscriptionID: &subscriptionIDStr,
			},
		}

		err := component.Cancel(core.ExecutionContext{
			Integration: integrationCtx,
			Metadata:    metadata,
		})

		require.NoError(t, err)
		// Note: Cancel no longer cleans up subscriptions.
		// Subscriptions are automatically cleaned up when the node is deleted.
	})

	t.Run("without subscription -> no error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Subscriptions: []contexts.Subscription{},
		}
		metadata := &contexts.MetadataContext{
			Metadata: WaitForButtonClickMetadata{
				AppSubscriptionID: nil,
			},
		}

		err := component.Cancel(core.ExecutionContext{
			Integration: integrationCtx,
			Metadata:    metadata,
		})

		require.NoError(t, err)
	})
}
