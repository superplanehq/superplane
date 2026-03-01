package telegram

import (
	"fmt"
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

	t.Run("missing chatId -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"chatId": ""},
		})

		require.ErrorContains(t, err, "chatId is required")
	})

	t.Run("missing message -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "token-123"},
			},
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"chatId":  "-100123456",
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
				"chatId":  "-100123456",
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
				"chatId":  "-100123456",
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
		token := "token-123"
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			expectedURL := fmt.Sprintf("https://api.telegram.org/bot%s/getChat", token)
			assert.Equal(t, expectedURL, req.URL.String())
			return jsonResponse(http.StatusOK, `{"ok":true,"result":{"id":12345,"type":"group","title":"MyGroup"}}`), nil
		})

		metadata := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": token},
		}

		err := component.Setup(core.SetupContext{
			Integration: integrationCtx,
			Metadata:    metadata,
			Configuration: map[string]any{
				"chatId":  "12345",
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
		require.NotNil(t, stored.Chat)
		assert.Equal(t, "12345", stored.Chat.ID)
		assert.Equal(t, "MyGroup", stored.Chat.Name)
	})
}

func Test__WaitForButtonClick__Execute(t *testing.T) {
	component := &WaitForButtonClick{}

	t.Run("missing chatId -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"chatId": "",
			},
		})

		require.ErrorContains(t, err, "chatId is required")
	})

	t.Run("valid configuration -> sends message with buttons", func(t *testing.T) {
		token := "token-123"
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			expectedURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
			assert.Equal(t, expectedURL, req.URL.String())
			return jsonResponse(http.StatusOK, `{"ok":true,"result":{"message_id":42,"chat":{"id":12345,"type":"group"},"text":"Choose an option","date":1234567890}}`), nil
		})

		executionID := uuid.New()
		metadata := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": token},
		}
		requestsCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			ID:          executionID,
			Integration: integrationCtx,
			Metadata:    metadata,
			Requests:    requestsCtx,
			Configuration: map[string]any{
				"chatId":  "12345",
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
		assert.NotNil(t, stored.MessageID)
		assert.Equal(t, int64(42), *stored.MessageID)
		assert.NotNil(t, stored.AppSubscriptionID)
	})
}

func Test__WaitForButtonClick__HandleAction(t *testing.T) {
	component := &WaitForButtonClick{}

	t.Run("button click -> emits received event", func(t *testing.T) {
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
		assert.Equal(t, "telegram.button.clicked", execState.Type)
		require.Len(t, execState.Payloads, 1)
		wrappedPayload := execState.Payloads[0].(map[string]any)
		payload := wrappedPayload["data"].(map[string]any)
		assert.Equal(t, "approve", payload["value"])
		assert.NotNil(t, payload["clicked_at"])
	})

	t.Run("timeout -> emits timeout event", func(t *testing.T) {
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
		assert.Equal(t, "telegram.button.timeout", execState.Type)
		require.Len(t, execState.Payloads, 1)
		wrappedPayload := execState.Payloads[0].(map[string]any)
		payload := wrappedPayload["data"].(map[string]any)
		assert.NotNil(t, payload["timeout_at"])
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
