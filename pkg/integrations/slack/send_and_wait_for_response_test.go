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
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"channel": "", "message": "test", "buttons": []any{}},
		})

		require.ErrorContains(t, err, "channel is required")
	})

	t.Run("missing message -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"channel": "C123", "message": "", "buttons": []any{}},
		})

		require.ErrorContains(t, err, "message is required")
	})

	t.Run("missing buttons -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"channel": "C123", "message": "test", "buttons": []any{}},
		})

		require.ErrorContains(t, err, "at least one button is required")
	})

	t.Run("too many buttons -> error", func(t *testing.T) {
		buttons := []any{
			map[string]any{"name": "Button 1", "value": "val1"},
			map[string]any{"name": "Button 2", "value": "val2"},
			map[string]any{"name": "Button 3", "value": "val3"},
			map[string]any{"name": "Button 4", "value": "val4"},
			map[string]any{"name": "Button 5", "value": "val5"},
		}
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"channel": "C123", "message": "test", "buttons": buttons},
		})

		require.ErrorContains(t, err, "maximum of 4 buttons allowed")
	})

	t.Run("button missing name -> error", func(t *testing.T) {
		buttons := []any{
			map[string]any{"name": "", "value": "val1"},
		}
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"channel": "C123", "message": "test", "buttons": buttons},
		})

		require.ErrorContains(t, err, "button 1: label is required")
	})

	t.Run("button missing value -> error", func(t *testing.T) {
		buttons := []any{
			map[string]any{"name": "Approve", "value": ""},
		}
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"channel": "C123", "message": "test", "buttons": buttons},
		})

		require.ErrorContains(t, err, "button 1: value is required")
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

		buttons := []any{
			map[string]any{"name": "Approve", "value": "approved"},
			map[string]any{"name": "Reject", "value": "rejected"},
		}

		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      metadata,
			Configuration: map[string]any{"channel": "C123", "message": "Please approve", "buttons": buttons},
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

	t.Run("missing channel -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
			Configuration:  map[string]any{"channel": "", "message": "test", "buttons": []any{}},
		})

		require.ErrorContains(t, err, "channel is required")
	})

	t.Run("valid configuration -> sends message with buttons and schedules timeout", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			if req.URL.String() == "https://slack.com/api/chat.postMessage" {
				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				var payload ChatPostMessageRequest
				require.NoError(t, json.Unmarshal(body, &payload))
				assert.Equal(t, "C123", payload.Channel)
				assert.Equal(t, "Please approve this request", payload.Text)
				assert.NotEmpty(t, payload.Blocks)
				// Verify blocks structure
				require.Len(t, payload.Blocks, 2) // section + actions
				return jsonResponse(http.StatusOK, `{"ok": true, "ts": "1234567890.123456", "message": {"text": "Please approve this request"}}`), nil
			}
			return jsonResponse(http.StatusNotFound, `{"ok": false}`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		requestCtx := &contexts.RequestContext{}
		metadata := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"botToken": "token-123",
			},
		}

		buttons := []any{
			map[string]any{"name": "Approve", "value": "approved"},
			map[string]any{"name": "Reject", "value": "rejected"},
		}

		timeout := 300 // 5 minutes

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			Metadata:       metadata,
			Requests:       requestCtx,
			Configuration: map[string]any{
				"channel": "C123",
				"message": "Please approve this request",
				"buttons": buttons,
				"timeout": timeout,
			},
		})

		require.NoError(t, err)

		// Execution should NOT be finished - waiting for response
		assert.False(t, execState.Finished)

		// Verify metadata was set
		stored, ok := metadata.Metadata.(SendAndWaitForResponseMetadata)
		require.True(t, ok)
		assert.Equal(t, SendAndWaitStateWaiting, stored.State)
		assert.Equal(t, "1234567890.123456", stored.MessageTS)
		assert.NotNil(t, stored.TimeoutAt)

		// Verify timeout was scheduled
		assert.Equal(t, "check_timeout", requestCtx.Action)

		// Verify KV was set for message TS lookup
		assert.Equal(t, "1234567890.123456", execState.KVs["slack_message_ts"])
	})
}

func Test__SendAndWaitForResponse__HandleAction__CheckTimeout(t *testing.T) {
	component := &SendAndWaitForResponse{}

	t.Run("already received -> does nothing", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: SendAndWaitForResponseMetadata{
				State: SendAndWaitStateReceived,
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "check_timeout",
			Metadata:       metadata,
			ExecutionState: execState,
			Parameters:     map[string]any{"messageTs": "1234567890.123456"},
		})

		require.NoError(t, err)
		assert.False(t, execState.Finished)
	})

	t.Run("waiting state -> emits timeout", func(t *testing.T) {
		timeoutAt := "2024-01-15T10:00:00Z"
		metadata := &contexts.MetadataContext{
			Metadata: SendAndWaitForResponseMetadata{
				State:     SendAndWaitStateWaiting,
				TimeoutAt: &timeoutAt,
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "check_timeout",
			Metadata:       metadata,
			ExecutionState: execState,
			Parameters:     map[string]any{"messageTs": "1234567890.123456"},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.Equal(t, SendAndWaitChannelTimeout, execState.Channel)
		assert.Equal(t, "slack.response.timeout", execState.Type)

		// Verify metadata was updated
		stored := metadata.Metadata.(SendAndWaitForResponseMetadata)
		assert.Equal(t, SendAndWaitStateTimedOut, stored.State)
	})
}

func Test__SendAndWaitForResponse__HandleAction__HandleResponse(t *testing.T) {
	component := &SendAndWaitForResponse{}

	t.Run("already timed out -> does nothing", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: SendAndWaitForResponseMetadata{
				State: SendAndWaitStateTimedOut,
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "handle_response",
			Metadata:       metadata,
			ExecutionState: execState,
			Parameters: map[string]any{
				"buttonName":  "Approve",
				"buttonValue": "approved",
				"userId":      "U123",
			},
		})

		require.NoError(t, err)
		assert.False(t, execState.Finished)
	})

	t.Run("waiting state -> emits received", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: SendAndWaitForResponseMetadata{
				State:     SendAndWaitStateWaiting,
				MessageTS: "1234567890.123456",
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "handle_response",
			Metadata:       metadata,
			ExecutionState: execState,
			Parameters: map[string]any{
				"buttonName":  "Approve",
				"buttonValue": "approved",
				"userId":      "U123456",
				"username":    "john.doe",
				"userName":    "John Doe",
			},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.Equal(t, SendAndWaitChannelReceived, execState.Channel)
		assert.Equal(t, "slack.response.received", execState.Type)

		// Verify metadata was updated
		stored := metadata.Metadata.(SendAndWaitForResponseMetadata)
		assert.Equal(t, SendAndWaitStateReceived, stored.State)
		require.NotNil(t, stored.ClickedButton)
		assert.Equal(t, "Approve", stored.ClickedButton.Name)
		assert.Equal(t, "approved", stored.ClickedButton.Value)
		require.NotNil(t, stored.ClickedBy)
		assert.Equal(t, "U123456", stored.ClickedBy.ID)
		assert.Equal(t, "john.doe", stored.ClickedBy.Username)
		assert.Equal(t, "John Doe", stored.ClickedBy.Name)
		assert.NotNil(t, stored.ClickedAt)
	})
}

func Test__SendAndWaitForResponse__Configuration(t *testing.T) {
	component := &SendAndWaitForResponse{}
	config := component.Configuration()

	require.Len(t, config, 4)

	// Channel field
	assert.Equal(t, "channel", config[0].Name)
	assert.True(t, config[0].Required)

	// Message field
	assert.Equal(t, "message", config[1].Name)
	assert.True(t, config[1].Required)

	// Timeout field
	assert.Equal(t, "timeout", config[2].Name)
	assert.False(t, config[2].Required)

	// Buttons field
	assert.Equal(t, "buttons", config[3].Name)
	assert.True(t, config[3].Required)
}

func Test__SendAndWaitForResponse__OutputChannels(t *testing.T) {
	component := &SendAndWaitForResponse{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 2)
	assert.Equal(t, SendAndWaitChannelReceived, channels[0].Name)
	assert.Equal(t, SendAndWaitChannelTimeout, channels[1].Name)
}

func Test__SendAndWaitForResponse__buildMessageBlocks(t *testing.T) {
	component := &SendAndWaitForResponse{}

	config := SendAndWaitForResponseConfiguration{
		Message: "Please approve this request",
		Buttons: []SendAndWaitButtonItem{
			{Name: "Approve", Value: "approved"},
			{Name: "Reject", Value: "rejected"},
		},
	}

	blocks := component.buildMessageBlocks("test-execution-123", config)

	require.Len(t, blocks, 2)

	// First block should be section with message
	section := blocks[0].(map[string]interface{})
	assert.Equal(t, "section", section["type"])
	text := section["text"].(map[string]string)
	assert.Equal(t, "mrkdwn", text["type"])
	assert.Equal(t, "Please approve this request", text["text"])

	// Second block should be actions with buttons
	actions := blocks[1].(map[string]interface{})
	assert.Equal(t, "actions", actions["type"])
	assert.Equal(t, "superplane_actions_test-execution-123", actions["block_id"])
	elements := actions["elements"].([]interface{})
	require.Len(t, elements, 2)

	// First button
	btn1 := elements[0].(map[string]interface{})
	assert.Equal(t, "button", btn1["type"])
	assert.Equal(t, "approved", btn1["value"])
	assert.Equal(t, "superplane_response_test-execution-123_0", btn1["action_id"])

	// Second button
	btn2 := elements[1].(map[string]interface{})
	assert.Equal(t, "button", btn2["type"])
	assert.Equal(t, "rejected", btn2["value"])
	assert.Equal(t, "superplane_response_test-execution-123_1", btn2["action_id"])
}
