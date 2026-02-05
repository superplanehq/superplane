package slack

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	SendAndWaitStateWaiting  = "waiting"
	SendAndWaitStateReceived = "received"
	SendAndWaitStateTimedOut = "timed_out"

	SendAndWaitChannelReceived = "received"
	SendAndWaitChannelTimeout  = "timeout"

	DefaultTimeoutSeconds = 3600 // 1 hour
)

type SendAndWaitForResponse struct{}

type SendAndWaitForResponseConfiguration struct {
	Channel string                   `json:"channel" mapstructure:"channel"`
	Message string                   `json:"message" mapstructure:"message"`
	Timeout *int                     `json:"timeout,omitempty" mapstructure:"timeout,omitempty"`
	Buttons []SendAndWaitButtonItem  `json:"buttons" mapstructure:"buttons"`
}

type SendAndWaitButtonItem struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type SendAndWaitForResponseMetadata struct {
	Channel       *ChannelMetadata `json:"channel,omitempty" mapstructure:"channel,omitempty"`
	State         string           `json:"state" mapstructure:"state"`
	MessageTS     string           `json:"messageTs,omitempty" mapstructure:"messageTs,omitempty"`
	ClickedButton *ClickedButton   `json:"clickedButton,omitempty" mapstructure:"clickedButton,omitempty"`
	ClickedBy     *SlackUser       `json:"clickedBy,omitempty" mapstructure:"clickedBy,omitempty"`
	ClickedAt     *string          `json:"clickedAt,omitempty" mapstructure:"clickedAt,omitempty"`
	TimeoutAt     *string          `json:"timeoutAt,omitempty" mapstructure:"timeoutAt,omitempty"`
}

type ClickedButton struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type SlackUser struct {
	ID       string `json:"id" mapstructure:"id"`
	Username string `json:"username,omitempty" mapstructure:"username,omitempty"`
	Name     string `json:"name,omitempty" mapstructure:"name,omitempty"`
}

func (c *SendAndWaitForResponse) Name() string {
	return "slack.sendAndWaitForResponse"
}

func (c *SendAndWaitForResponse) Label() string {
	return "Send and Wait for Response"
}

func (c *SendAndWaitForResponse) Description() string {
	return "Send a message with interactive buttons and wait for a user to click one"
}

func (c *SendAndWaitForResponse) Documentation() string {
	return `The Send and Wait for Response component sends a message with interactive buttons to a Slack channel and waits for a user to click one of the buttons.

## Use Cases

- **Approval workflows**: Ask for approval or rejection before proceeding
- **Decision points**: Let users choose between different workflow paths
- **Confirmations**: Require explicit user confirmation before critical actions
- **Interactive notifications**: Send actionable messages that require user input

## Configuration

- **Channel**: Select the Slack channel to send the message to
- **Message**: The message text to send (supports expressions and Slack markdown formatting)
- **Timeout**: Maximum time in seconds to wait for a response (optional, defaults to 1 hour)
- **Buttons**: 1-4 interactive buttons, each with a display name (label) and a value

## Output Channels

- **Received**: Emitted when a user clicks one of the buttons. Includes the clicked button's value and information about who clicked it.
- **Timeout**: Emitted when no user clicks a button within the configured timeout period.

## Output Data

When a button is clicked, the output includes:
- **button.name**: The display label of the clicked button
- **button.value**: The configured value of the clicked button
- **user.id**: Slack user ID of the person who clicked
- **user.username**: Username of the person who clicked
- **user.name**: Display name of the person who clicked
- **clickedAt**: Timestamp when the button was clicked

## Notes

- The message is updated after a button is clicked to show which button was selected
- Only the first button click is processed; subsequent clicks are ignored
- The Slack app must be installed and have permission to post interactive messages
- Supports Slack markdown formatting in message text`
}

func (c *SendAndWaitForResponse) Icon() string {
	return "slack"
}

func (c *SendAndWaitForResponse) Color() string {
	return "gray"
}

func (c *SendAndWaitForResponse) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: SendAndWaitChannelReceived, Label: "Received", Description: "Emits when a button is clicked"},
		{Name: SendAndWaitChannelTimeout, Label: "Timeout", Description: "Emits when no response is received within the timeout"},
	}
}

func (c *SendAndWaitForResponse) Configuration() []configuration.Field {
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
			Description: "Maximum time to wait for a response (default: 3600 seconds / 1 hour)",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(86400), // Max 24 hours
				},
			},
		},
		{
			Name:        "buttons",
			Label:       "Buttons",
			Description: "Interactive buttons (1-4) for user response",
			Type:        configuration.FieldTypeList,
			Required:    true,
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
								TypeOptions: &configuration.TypeOptions{
									String: &configuration.StringTypeOptions{
										MaxLength: intPtr(75), // Slack button text limit
									},
								},
							},
							{
								Name:     "value",
								Label:    "Value",
								Type:     configuration.FieldTypeString,
								Required: true,
								TypeOptions: &configuration.TypeOptions{
									String: &configuration.StringTypeOptions{
										MaxLength: intPtr(2000), // Slack action value limit
									},
								},
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

func (c *SendAndWaitForResponse) Setup(ctx core.SetupContext) error {
	var config SendAndWaitForResponseConfiguration
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
		return errors.New("maximum of 4 buttons allowed")
	}

	// Validate each button
	for i, button := range config.Buttons {
		if button.Name == "" {
			return fmt.Errorf("button %d: label is required", i+1)
		}
		if button.Value == "" {
			return fmt.Errorf("button %d: value is required", i+1)
		}
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Slack client: %w", err)
	}

	channelInfo, err := client.GetChannelInfo(config.Channel)
	if err != nil {
		return fmt.Errorf("channel validation failed: %w", err)
	}

	metadata := SendAndWaitForResponseMetadata{
		Channel: &ChannelMetadata{
			ID:   channelInfo.ID,
			Name: channelInfo.Name,
		},
		State: "",
	}

	return ctx.Metadata.Set(metadata)
}

func (c *SendAndWaitForResponse) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SendAndWaitForResponse) Execute(ctx core.ExecutionContext) error {
	var config SendAndWaitForResponseConfiguration
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

	// Build the interactive message with buttons
	blocks := c.buildMessageBlocks(ctx.ID.String(), config)

	response, err := client.PostMessage(ChatPostMessageRequest{
		Channel: config.Channel,
		Text:    config.Message, // Fallback text for notifications
		Blocks:  blocks,
	})

	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Calculate timeout
	timeoutSeconds := DefaultTimeoutSeconds
	if config.Timeout != nil && *config.Timeout > 0 {
		timeoutSeconds = *config.Timeout
	}
	timeoutAt := time.Now().Add(time.Duration(timeoutSeconds) * time.Second).Format(time.RFC3339)

	// Store execution metadata
	metadata := SendAndWaitForResponseMetadata{
		State:     SendAndWaitStateWaiting,
		MessageTS: response.TS,
		TimeoutAt: &timeoutAt,
	}

	// Preserve channel metadata if it exists
	var existingMetadata SendAndWaitForResponseMetadata
	if ctx.Metadata.Get() != nil {
		_ = mapstructure.Decode(ctx.Metadata.Get(), &existingMetadata)
		if existingMetadata.Channel != nil {
			metadata.Channel = existingMetadata.Channel
		}
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	// Store the execution ID in KV for looking up later from interaction callback
	if err := ctx.ExecutionState.SetKV("slack_message_ts", response.TS); err != nil {
		return fmt.Errorf("failed to store message timestamp: %w", err)
	}

	// Schedule a timeout check action
	err = ctx.Requests.ScheduleActionCall("check_timeout", map[string]any{
		"messageTs": response.TS,
	}, time.Duration(timeoutSeconds)*time.Second)

	if err != nil {
		return fmt.Errorf("failed to schedule timeout check: %w", err)
	}

	// Don't finish execution - wait for button click or timeout
	return nil
}

func (c *SendAndWaitForResponse) buildMessageBlocks(executionID string, config SendAndWaitForResponseConfiguration) []interface{} {
	// Build button elements
	buttons := make([]interface{}, 0, len(config.Buttons))
	for i, btn := range config.Buttons {
		buttons = append(buttons, map[string]interface{}{
			"type": "button",
			"text": map[string]string{
				"type":  "plain_text",
				"text":  btn.Name,
				"emoji": "true",
			},
			"value":     btn.Value,
			"action_id": fmt.Sprintf("superplane_response_%s_%d", executionID, i),
		})
	}

	blocks := []interface{}{
		// Message text section
		map[string]interface{}{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": config.Message,
			},
		},
		// Actions block with buttons
		map[string]interface{}{
			"type":     "actions",
			"block_id": fmt.Sprintf("superplane_actions_%s", executionID),
			"elements": buttons,
		},
	}

	return blocks
}

func (c *SendAndWaitForResponse) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "check_timeout",
			Description:    "Check if the response has timed out",
			UserAccessible: false,
			Parameters: []configuration.Field{
				{
					Name:     "messageTs",
					Label:    "Message Timestamp",
					Type:     configuration.FieldTypeString,
					Required: true,
				},
			},
		},
		{
			Name:           "handle_response",
			Description:    "Handle a button click response from Slack",
			UserAccessible: false,
			Parameters: []configuration.Field{
				{
					Name:     "buttonName",
					Label:    "Button Name",
					Type:     configuration.FieldTypeString,
					Required: true,
				},
				{
					Name:     "buttonValue",
					Label:    "Button Value",
					Type:     configuration.FieldTypeString,
					Required: true,
				},
				{
					Name:     "userId",
					Label:    "User ID",
					Type:     configuration.FieldTypeString,
					Required: true,
				},
				{
					Name:     "username",
					Label:    "Username",
					Type:     configuration.FieldTypeString,
					Required: false,
				},
				{
					Name:     "userName",
					Label:    "User Name",
					Type:     configuration.FieldTypeString,
					Required: false,
				},
			},
		},
	}
}

func (c *SendAndWaitForResponse) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "check_timeout":
		return c.handleCheckTimeout(ctx)
	case "handle_response":
		return c.handleResponse(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *SendAndWaitForResponse) handleCheckTimeout(ctx core.ActionContext) error {
	var metadata SendAndWaitForResponseMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	// If already received a response, nothing to do
	if metadata.State != SendAndWaitStateWaiting {
		return nil
	}

	// Update metadata to timed out state
	metadata.State = SendAndWaitStateTimedOut
	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	// Emit timeout and finish execution
	return ctx.ExecutionState.Emit(
		SendAndWaitChannelTimeout,
		"slack.response.timeout",
		[]any{map[string]any{
			"timeoutAt": metadata.TimeoutAt,
		}},
	)
}

func (c *SendAndWaitForResponse) handleResponse(ctx core.ActionContext) error {
	var metadata SendAndWaitForResponseMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	// If not waiting, ignore (already received or timed out)
	if metadata.State != SendAndWaitStateWaiting {
		return nil
	}

	buttonName, _ := ctx.Parameters["buttonName"].(string)
	buttonValue, _ := ctx.Parameters["buttonValue"].(string)
	userID, _ := ctx.Parameters["userId"].(string)
	username, _ := ctx.Parameters["username"].(string)
	userName, _ := ctx.Parameters["userName"].(string)

	clickedAt := time.Now().Format(time.RFC3339)

	// Update metadata
	metadata.State = SendAndWaitStateReceived
	metadata.ClickedButton = &ClickedButton{
		Name:  buttonName,
		Value: buttonValue,
	}
	metadata.ClickedBy = &SlackUser{
		ID:       userID,
		Username: username,
		Name:     userName,
	}
	metadata.ClickedAt = &clickedAt

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	// Emit response and finish execution
	return ctx.ExecutionState.Emit(
		SendAndWaitChannelReceived,
		"slack.response.received",
		[]any{map[string]any{
			"button": map[string]string{
				"name":  buttonName,
				"value": buttonValue,
			},
			"user": map[string]string{
				"id":       userID,
				"username": username,
				"name":     userName,
			},
			"clickedAt": clickedAt,
		}},
	)
}

func (c *SendAndWaitForResponse) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *SendAndWaitForResponse) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SendAndWaitForResponse) Cleanup(ctx core.SetupContext) error {
	return nil
}

// OnIntegrationMessage implements core.IntegrationComponent interface
// This is called when an interaction event (button click) is received from Slack
func (c *SendAndWaitForResponse) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	message, ok := ctx.Message.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid message format")
	}

	messageType, ok := message["type"].(string)
	if !ok || messageType != "button_click" {
		return nil // Not a button click, ignore
	}

	// This method is for handling subscription messages
	// The actual execution handling is done via the HandleAction method
	// which is called by the component execution system

	return nil
}
