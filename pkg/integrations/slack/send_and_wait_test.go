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

func Test__SendAndWait__Setup(t *testing.T) {
	component := &SendAndWait{}

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
			Configuration: map[string]any{
				"channel": "",
				"message": "hello",
				"buttons": []any{map[string]any{"name": "OK", "value": "ok"}},
			},
		})

		require.ErrorContains(t, err, "channel is required")
	})

	t.Run("missing message -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{
				"channel": "C123",
				"message": "",
				"buttons": []any{map[string]any{"name": "OK", "value": "ok"}},
			},
		})

		require.ErrorContains(t, err, "message is required")
	})

	t.Run("no buttons -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{
				"channel": "C123",
				"message": "hello",
				"buttons": []any{},
			},
		})

		require.ErrorContains(t, err, "at least one button is required")
	})

	t.Run("too many buttons -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{
				"channel": "C123",
				"message": "hello",
				"buttons": []any{
					map[string]any{"name": "1", "value": "1"},
					map[string]any{"name": "2", "value": "2"},
					map[string]any{"name": "3", "value": "3"},
					map[string]any{"name": "4", "value": "4"},
					map[string]any{"name": "5", "value": "5"},
				},
			},
		})

		require.ErrorContains(t, err, "maximum 4 buttons allowed")
	})

	t.Run("valid configuration -> stores metadata and subscribes", func(t *testing.T) {
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
			Integration:   integrationCtx,
			Metadata:      metadata,
			Configuration: map[string]any{
				"channel": "C123",
				"message": "Deploy to production?",
				"buttons": []any{
					map[string]any{"name": "Approve", "value": "approve"},
					map[string]any{"name": "Reject", "value": "reject"},
				},
			},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(SendAndWaitMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.Channel)
		assert.Equal(t, "C123", stored.Channel.ID)
		assert.Equal(t, "general", stored.Channel.Name)
		require.NotNil(t, stored.AppSubscriptionID)

		// Verify subscription was created
		require.Len(t, integrationCtx.Subscriptions, 1)
	})
}

func Test__SendAndWait__Execute(t *testing.T) {
	component := &SendAndWait{}

	t.Run("missing channel -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Configuration:  map[string]any{"channel": ""},
			Metadata:       &contexts.MetadataContext{},
			NodeMetadata:   &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "channel is required")
	})

	t.Run("valid config -> sends message with blocks and leaves execution waiting", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			if req.URL.String() == "https://slack.com/api/chat.postMessage" {
				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				var payload ChatPostMessageRequest
				require.NoError(t, json.Unmarshal(body, &payload))
				assert.Equal(t, "C123", payload.Channel)
				assert.Equal(t, "Deploy?", payload.Text)
				require.Len(t, payload.Blocks, 2)
				return jsonResponse(http.StatusOK, `{"ok": true, "ts": "1234.5678", "message": {"text": "Deploy?"}}`), nil
			}
			return jsonResponse(http.StatusNotFound, `{"ok": false}`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		requestCtx := &contexts.RequestContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"botToken": "token-123",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			Configuration: map[string]any{
				"channel": "C123",
				"message": "Deploy?",
				"timeout": 30.0,
				"buttons": []any{
					map[string]any{"name": "Yes", "value": "yes"},
					map[string]any{"name": "No", "value": "no"},
				},
			},
			Metadata:     &contexts.MetadataContext{},
			NodeMetadata: &contexts.MetadataContext{},
			Requests:     requestCtx,
		})

		require.NoError(t, err)

		// Execution should NOT be finished (waiting for response)
		assert.False(t, execState.Finished)

		// Message timestamp stored as KV
		assert.Equal(t, "1234.5678", execState.KVs["message_ts"])

		// Timeout action should be scheduled
		assert.Equal(t, "timeout", requestCtx.Action)
		assert.Equal(t, 30*time.Second, requestCtx.Duration)
		assert.Equal(t, "1234.5678", requestCtx.Params["message_ts"])
		assert.Equal(t, "C123", requestCtx.Params["channel"])
	})

	t.Run("no timeout -> no scheduled action", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"ok": true, "ts": "1234.5678", "message": {"text": "hi"}}`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		requestCtx := &contexts.RequestContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"botToken": "token-123",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			Configuration: map[string]any{
				"channel": "C123",
				"message": "hi",
				"buttons": []any{
					map[string]any{"name": "OK", "value": "ok"},
				},
			},
			Metadata:     &contexts.MetadataContext{},
			NodeMetadata: &contexts.MetadataContext{},
			Requests:     requestCtx,
		})

		require.NoError(t, err)
		assert.False(t, execState.Finished)
		assert.Empty(t, requestCtx.Action)
	})
}

func Test__SendAndWait__HandleAction(t *testing.T) {
	component := &SendAndWait{}

	t.Run("unknown action -> error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name:           "unknown",
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Metadata:       &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "unknown action")
	})

	t.Run("timeout on finished execution -> no-op", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}, Finished: true}

		err := component.HandleAction(core.ActionContext{
			Name:           "timeout",
			ExecutionState: execState,
			Metadata:       &contexts.MetadataContext{},
			Parameters:     map[string]any{},
		})

		require.NoError(t, err)
	})

	t.Run("timeout on waiting execution -> emits timeout", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "timeout",
			ExecutionState: execState,
			Metadata:       &contexts.MetadataContext{},
			Integration:    &contexts.IntegrationContext{},
			Parameters: map[string]any{
				"message_ts": "1234.5678",
				"channel":    "C123",
			},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.Equal(t, ChannelTimeout, execState.Channel)
		assert.Equal(t, "slack.interaction.timeout", execState.Type)
	})
}

func Test__SendAndWait__ProcessQueueItem(t *testing.T) {
	component := &SendAndWait{}

	t.Run("non-interaction input -> default processing", func(t *testing.T) {
		defaultCalled := false
		ctx := core.ProcessQueueContext{
			Input: "regular input",
			DefaultProcessing: func() (*uuid.UUID, error) {
				defaultCalled = true
				id := uuid.New()
				return &id, nil
			},
		}

		result, err := component.ProcessQueueItem(ctx)

		require.NoError(t, err)
		assert.True(t, defaultCalled)
		assert.NotNil(t, result)
	})

	t.Run("interaction with missing message_ts -> dequeues silently", func(t *testing.T) {
		dequeued := false
		ctx := core.ProcessQueueContext{
			Input: map[string]any{
				"type":    "block_actions",
				"actions": []any{map[string]any{"value": "approve", "action_id": "superplane_btn_0"}},
			},
			DequeueItem: func() error {
				dequeued = true
				return nil
			},
		}

		result, err := component.ProcessQueueItem(ctx)

		require.NoError(t, err)
		assert.Nil(t, result)
		assert.True(t, dequeued)
	})

	t.Run("interaction with no matching execution -> dequeues silently", func(t *testing.T) {
		dequeued := false
		ctx := core.ProcessQueueContext{
			Input: map[string]any{
				"type":    "block_actions",
				"message": map[string]any{"ts": "1234.5678"},
				"actions": []any{map[string]any{"value": "approve", "action_id": "superplane_btn_0"}},
			},
			DequeueItem: func() error {
				dequeued = true
				return nil
			},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				assert.Equal(t, "message_ts", key)
				assert.Equal(t, "1234.5678", value)
				return nil, nil
			},
		}

		result, err := component.ProcessQueueItem(ctx)

		require.NoError(t, err)
		assert.Nil(t, result)
		assert.True(t, dequeued)
	})

	t.Run("interaction with matching execution -> emits received", func(t *testing.T) {
		dequeued := false
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"state":     "waiting",
				"messageTs": "1234.5678",
			},
		}
		integrationCtx := &contexts.IntegrationContext{}

		ctx := core.ProcessQueueContext{
			Input: map[string]any{
				"type":    "block_actions",
				"message": map[string]any{"ts": "1234.5678"},
				"actions": []any{map[string]any{
					"value":     "approve",
					"action_id": "superplane_btn_0",
				}},
				"user":    map[string]any{"id": "U123", "username": "john"},
				"channel": map[string]any{"id": "C123"},
			},
			DequeueItem: func() error {
				dequeued = true
				return nil
			},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					ExecutionState: execState,
					Metadata:       metadataCtx,
					Integration:    integrationCtx,
					Configuration:  map[string]any{"channel": "C123"},
				}, nil
			},
		}

		result, err := component.ProcessQueueItem(ctx)

		require.NoError(t, err)
		assert.Nil(t, result)
		assert.True(t, dequeued)
		assert.True(t, execState.Finished)
		assert.Equal(t, ChannelReceived, execState.Channel)
		assert.Equal(t, "slack.interaction.received", execState.Type)
		require.Len(t, execState.Payloads, 1)

		payload := execState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		assert.Equal(t, "approve", data["value"])
	})
}

func Test__SendAndWait__OnIntegrationMessage(t *testing.T) {
	component := &SendAndWait{}

	t.Run("non-map message -> ignored", func(t *testing.T) {
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: "not a map",
			Events:  &contexts.EventContext{},
		})

		require.NoError(t, err)
	})

	t.Run("non-block_actions type -> ignored", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{"type": "view_submission"},
			Events:  events,
		})

		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("non-superplane action -> ignored", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"type":    "block_actions",
				"actions": []any{map[string]any{"action_id": "other_action", "value": "v"}},
			},
			Events: events,
		})

		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("superplane block_actions -> emits event", func(t *testing.T) {
		events := &contexts.EventContext{}
		interaction := map[string]any{
			"type":    "block_actions",
			"actions": []any{map[string]any{"action_id": "superplane_btn_0", "value": "approve"}},
			"message": map[string]any{"ts": "1234.5678"},
		}

		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: interaction,
			Events:  events,
		})

		require.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})
}

func Test__SendAndWait__BuildButtonBlocks(t *testing.T) {
	buttons := []SendAndWaitButton{
		{Name: "Approve", Value: "approve"},
		{Name: "Reject", Value: "reject"},
	}

	blocks := buildButtonBlocks("Deploy to production?", buttons)

	require.Len(t, blocks, 2)

	// First block is the message section
	section := blocks[0].(map[string]any)
	assert.Equal(t, "section", section["type"])
	text := section["text"].(map[string]any)
	assert.Equal(t, "mrkdwn", text["type"])
	assert.Equal(t, "Deploy to production?", text["text"])

	// Second block is the actions block
	actions := blocks[1].(map[string]any)
	assert.Equal(t, "actions", actions["type"])
	elements := actions["elements"].([]interface{})
	require.Len(t, elements, 2)

	btn0 := elements[0].(map[string]any)
	assert.Equal(t, "button", btn0["type"])
	assert.Equal(t, "approve", btn0["value"])
	assert.Equal(t, "superplane_btn_0", btn0["action_id"])

	btn1 := elements[1].(map[string]any)
	assert.Equal(t, "reject", btn1["value"])
	assert.Equal(t, "superplane_btn_1", btn1["action_id"])
}

func Test__SendAndWait__ExtractHelpers(t *testing.T) {
	input := map[string]any{
		"type":    "block_actions",
		"message": map[string]any{"ts": "1234.5678"},
		"actions": []any{map[string]any{
			"action_id": "superplane_btn_0",
			"value":     "approve",
		}},
		"user":    map[string]any{"id": "U123", "username": "john"},
		"channel": map[string]any{"id": "C456"},
	}

	assert.Equal(t, "1234.5678", extractMessageTS(input))
	assert.Equal(t, "approve", extractButtonValue(input))
	assert.Equal(t, "john", extractUserName(input))
	assert.Equal(t, "C456", extractChannelID(input))

	user := extractUser(input)
	assert.Equal(t, "U123", user["id"])
	assert.Equal(t, "john", user["username"])
}

func Test__SendAndWait__IsSuperplaneInteraction(t *testing.T) {
	assert.True(t, isSuperplaneInteraction(map[string]any{
		"actions": []any{map[string]any{"action_id": "superplane_btn_0"}},
	}))

	assert.False(t, isSuperplaneInteraction(map[string]any{
		"actions": []any{map[string]any{"action_id": "other_action"}},
	}))

	assert.False(t, isSuperplaneInteraction(map[string]any{
		"actions": []any{},
	}))

	assert.False(t, isSuperplaneInteraction(map[string]any{}))
}
