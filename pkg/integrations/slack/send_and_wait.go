package slack

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ChannelReceived = "received"
	ChannelTimeout  = "timeout"
)

type SendAndWait struct{}

type SendAndWaitConfiguration struct {
	Channel string              `json:"channel" mapstructure:"channel"`
	Message string              `json:"message" mapstructure:"message"`
	Timeout float64             `json:"timeout" mapstructure:"timeout"`
	Buttons []SendAndWaitButton `json:"buttons" mapstructure:"buttons"`
}

type SendAndWaitButton struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type SendAndWaitMetadata struct {
	Channel           *ChannelMetadata `json:"channel" mapstructure:"channel"`
	State             string           `json:"state" mapstructure:"state"`
	MessageTS         string           `json:"messageTs" mapstructure:"messageTs"`
	AppSubscriptionID *string          `json:"appSubscriptionID,omitempty" mapstructure:"appSubscriptionID,omitempty"`
}

func (c *SendAndWait) Name() string {
	return "slack.sendAndWait"
}

func (c *SendAndWait) Label() string {
	return "Send and Wait for Response"
}

func (c *SendAndWait) Description() string {
	return "Send a message with buttons and wait for a user response"
}

func (c *SendAndWait) Documentation() string {
	return `The Send and Wait for Response component sends a message to a Slack channel with configurable buttons and pauses the workflow until a user clicks one of the buttons.

## Use Cases

- **Approvals**: Send approval/rejection buttons to stakeholders
- **Decision gates**: Present options and wait for user selection
- **Incident response**: Acknowledge or escalate incidents from Slack
- **Deployment confirmations**: Confirm or cancel deployments

## Configuration

- **Channel**: Select the Slack channel to send the message to
- **Message**: The message text (supports Slack markdown formatting)
- **Timeout**: Optional timeout in seconds; if no response within the timeout, the workflow continues on the "Timeout" channel
- **Buttons**: 1-4 buttons, each with a display name and a value

## Output Channels

- **Received**: Emitted when a user clicks a button; includes the button value and user info
- **Timeout**: Emitted when the timeout expires without a button click

## How It Works

1. A message with interactive buttons is posted to the configured Slack channel
2. The workflow pauses and waits for a user to click a button
3. When clicked, the message is updated to show which option was selected
4. The workflow continues on the "Received" channel with the button value

## Notes

- The Slack app must have interactivity enabled
- Only button clicks are supported as responses (no free-text)
- Each message can have 1-4 buttons`
}

func (c *SendAndWait) Icon() string {
	return "slack"
}

func (c *SendAndWait) Color() string {
	return "gray"
}

func (c *SendAndWait) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ChannelReceived, Label: "Received", Description: "A user clicked a button"},
		{Name: ChannelTimeout, Label: "Timeout", Description: "No response within the configured timeout"},
	}
}

func (c *SendAndWait) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "channel",
			Label:    "Channel",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "channel",
				},
			},
		},
		{
			Name:     "message",
			Label:    "Message",
			Type:     configuration.FieldTypeText,
			Required: true,
		},
		{
			Name:        "timeout",
			Label:       "Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Maximum time to wait for a response, in seconds",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
				},
			},
		},
		{
			Name:     "buttons",
			Label:    "Buttons",
			Type:     configuration.FieldTypeList,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Button",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "name",
								Label:    "Label",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "value",
								Label:    "Value",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
	}
}

func intPtr(i int) *int {
	return &i
}

func (c *SendAndWait) Setup(ctx core.SetupContext) error {
	var config SendAndWaitConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Channel == "" {
		return errors.New("channel is required")
	}

	if config.Message == "" {
		return errors.New("message is required")
	}

	if len(config.Buttons) == 0 {
		return errors.New("at least one button is required")
	}

	if len(config.Buttons) > 4 {
		return errors.New("maximum 4 buttons allowed")
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Slack client: %w", err)
	}

	channelInfo, err := client.GetChannelInfo(config.Channel)
	if err != nil {
		return fmt.Errorf("channel validation failed: %w", err)
	}

	// Check for existing subscription
	var metadata SendAndWaitMetadata
	_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)

	subscriptionID, err := c.subscribe(ctx, metadata)
	if err != nil {
		return fmt.Errorf("failed to subscribe to interactivity events: %w", err)
	}

	return ctx.Metadata.Set(SendAndWaitMetadata{
		AppSubscriptionID: subscriptionID,
		Channel: &ChannelMetadata{
			ID:   channelInfo.ID,
			Name: channelInfo.Name,
		},
	})
}

func (c *SendAndWait) subscribe(ctx core.SetupContext, metadata SendAndWaitMetadata) (*string, error) {
	if metadata.AppSubscriptionID != nil {
		return metadata.AppSubscriptionID, nil
	}

	subscriptionID, err := ctx.Integration.Subscribe(SubscriptionConfiguration{
		InteractivityTypes: []string{"block_actions"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	s := subscriptionID.String()
	return &s, nil
}

func (c *SendAndWait) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	//
	// Check if the input is an interaction event from OnIntegrationMessage.
	// If so, find the waiting execution and complete it.
	//
	input, ok := ctx.Input.(map[string]any)
	if !ok {
		return ctx.DefaultProcessing()
	}

	interactionType, _ := input["type"].(string)
	if interactionType != "block_actions" {
		return ctx.DefaultProcessing()
	}

	// Extract message timestamp to correlate with the waiting execution
	messageTS := extractMessageTS(input)
	if messageTS == "" {
		_ = ctx.DequeueItem()
		return nil, nil
	}

	executionCtx, err := ctx.FindExecutionByKV("message_ts", messageTS)
	if err != nil {
		return nil, fmt.Errorf("failed to find execution: %w", err)
	}

	if executionCtx == nil {
		// Execution not found (already timed out or cancelled)
		_ = ctx.DequeueItem()
		return nil, nil
	}

	if executionCtx.ExecutionState.IsFinished() {
		_ = ctx.DequeueItem()
		return nil, nil
	}

	// Extract button value and user info
	buttonValue := extractButtonValue(input)
	userName := extractUserName(input)
	channelID := extractChannelID(input)

	// Update metadata
	var metadata SendAndWaitMetadata
	_ = mapstructure.Decode(executionCtx.Metadata.Get(), &metadata)
	metadata.State = "received"
	_ = executionCtx.Metadata.Set(metadata)

	// Update the Slack message to show which button was clicked
	updateMessageAfterClick(executionCtx.Integration, channelID, messageTS, metadata.Channel, buttonValue, userName)

	// Emit on the "received" channel
	responsePayload := map[string]any{
		"value": buttonValue,
		"user":  extractUser(input),
	}

	err = executionCtx.ExecutionState.Emit(
		ChannelReceived,
		"slack.interaction.received",
		[]any{responsePayload},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to emit response: %w", err)
	}

	_ = ctx.DequeueItem()
	return nil, nil
}

func (c *SendAndWait) Execute(ctx core.ExecutionContext) error {
	var config SendAndWaitConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Channel == "" {
		return errors.New("channel is required")
	}

	if len(config.Buttons) == 0 {
		return errors.New("at least one button is required")
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Slack client: %w", err)
	}

	// Build Block Kit message with buttons
	blocks := buildButtonBlocks(config.Message, config.Buttons)

	response, err := client.PostMessage(ChatPostMessageRequest{
		Channel: config.Channel,
		Text:    config.Message,
		Blocks:  blocks,
	})
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Store message timestamp as KV for correlation
	if err := ctx.ExecutionState.SetKV("message_ts", response.TS); err != nil {
		return fmt.Errorf("failed to store message timestamp: %w", err)
	}

	// Update metadata
	var nodeMetadata SendAndWaitMetadata
	_ = mapstructure.Decode(ctx.NodeMetadata.Get(), &nodeMetadata)

	metadata := SendAndWaitMetadata{
		Channel:   nodeMetadata.Channel,
		State:     "waiting",
		MessageTS: response.TS,
	}
	_ = ctx.Metadata.Set(metadata)

	// Schedule timeout if configured
	if config.Timeout > 0 {
		duration := time.Duration(config.Timeout * float64(time.Second))
		if err := ctx.Requests.ScheduleActionCall("timeout", map[string]any{
			"message_ts": response.TS,
			"channel":    config.Channel,
		}, duration); err != nil {
			return fmt.Errorf("failed to schedule timeout: %w", err)
		}
	}

	// Return nil without emitting - leaves execution in "waiting" state
	return nil
}

func (c *SendAndWait) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "timeout",
			Description:    "Handle timeout for pending response",
			UserAccessible: false,
		},
	}
}

func (c *SendAndWait) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "timeout" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	// Update metadata
	var metadata SendAndWaitMetadata
	_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	metadata.State = "timed_out"
	_ = ctx.Metadata.Set(metadata)

	// Update the Slack message to indicate timeout
	messageTS, _ := ctx.Parameters["message_ts"].(string)
	channel, _ := ctx.Parameters["channel"].(string)
	if messageTS != "" && channel != "" {
		updateMessageOnTimeout(ctx.Integration, channel, messageTS)
	}

	return ctx.ExecutionState.Emit(
		ChannelTimeout,
		"slack.interaction.timeout",
		[]any{map[string]any{"reason": "timeout"}},
	)
}

func (c *SendAndWait) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *SendAndWait) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	interaction, ok := ctx.Message.(map[string]any)
	if !ok {
		return nil
	}

	// Only handle block_actions type
	interactionType, _ := interaction["type"].(string)
	if interactionType != "block_actions" {
		return nil
	}

	// Check if this interaction has a superplane action_id
	if !isSuperplaneInteraction(interaction) {
		return nil
	}

	// Emit the interaction as an event for ProcessQueueItem to handle
	return ctx.Events.Emit("slack.interaction", ctx.Message)
}

func (c *SendAndWait) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SendAndWait) Cleanup(ctx core.SetupContext) error {
	return nil
}

// buildButtonBlocks creates Block Kit blocks with a message section and action buttons.
func buildButtonBlocks(message string, buttons []SendAndWaitButton) []interface{} {
	elements := make([]interface{}, 0, len(buttons))
	for i, btn := range buttons {
		elements = append(elements, map[string]any{
			"type": "button",
			"text": map[string]any{
				"type": "plain_text",
				"text": btn.Name,
			},
			"value":     btn.Value,
			"action_id": fmt.Sprintf("superplane_btn_%d", i),
		})
	}

	return []interface{}{
		map[string]any{
			"type": "section",
			"text": map[string]any{
				"type": "mrkdwn",
				"text": message,
			},
		},
		map[string]any{
			"type":     "actions",
			"elements": elements,
		},
	}
}

func extractMessageTS(input map[string]any) string {
	message, ok := input["message"].(map[string]any)
	if !ok {
		return ""
	}
	ts, _ := message["ts"].(string)
	return ts
}

func extractButtonValue(input map[string]any) string {
	actions, ok := input["actions"].([]any)
	if !ok || len(actions) == 0 {
		return ""
	}
	action, ok := actions[0].(map[string]any)
	if !ok {
		return ""
	}
	value, _ := action["value"].(string)
	return value
}

func extractUserName(input map[string]any) string {
	user, ok := input["user"].(map[string]any)
	if !ok {
		return ""
	}
	username, _ := user["username"].(string)
	return username
}

func extractChannelID(input map[string]any) string {
	channel, ok := input["channel"].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := channel["id"].(string)
	return id
}

func extractUser(input map[string]any) map[string]any {
	user, ok := input["user"].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return user
}

func isSuperplaneInteraction(interaction map[string]any) bool {
	actions, ok := interaction["actions"].([]any)
	if !ok || len(actions) == 0 {
		return false
	}
	action, ok := actions[0].(map[string]any)
	if !ok {
		return false
	}
	actionID, _ := action["action_id"].(string)
	return strings.HasPrefix(actionID, "superplane_btn_")
}

func updateMessageAfterClick(integration core.IntegrationContext, channelID, messageTS string, channelMeta *ChannelMetadata, buttonValue, userName string) {
	client, err := NewClient(integration)
	if err != nil {
		return
	}

	if channelID == "" && channelMeta != nil {
		channelID = channelMeta.ID
	}
	if channelID == "" {
		return
	}

	text := fmt.Sprintf("_%s selected *%s*_", userName, buttonValue)
	_ = client.UpdateMessage(ChatUpdateMessageRequest{
		Channel:   channelID,
		Timestamp: messageTS,
		Text:      text,
		Blocks: []interface{}{
			map[string]any{
				"type": "section",
				"text": map[string]any{
					"type": "mrkdwn",
					"text": text,
				},
			},
		},
	})
}

func updateMessageOnTimeout(integration core.IntegrationContext, channel, messageTS string) {
	client, err := NewClient(integration)
	if err != nil {
		return
	}

	text := "This request has timed out."
	_ = client.UpdateMessage(ChatUpdateMessageRequest{
		Channel:   channel,
		Timestamp: messageTS,
		Text:      text,
		Blocks: []interface{}{
			map[string]any{
				"type": "section",
				"text": map[string]any{
					"type": "mrkdwn",
					"text": text,
				},
			},
		},
	})
}
