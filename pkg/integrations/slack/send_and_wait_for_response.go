package slack

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	sendAndWaitSubscriptionType = "button_click"

	sendAndWaitChannelReceived = "received"
	sendAndWaitChannelTimeout  = "timeout"

	sendAndWaitActionButtonClick = "buttonClick"
	sendAndWaitActionTimeout     = "timeout"
)

type SendAndWaitForResponse struct{}

type SendAndWaitForResponseConfiguration struct {
	Channel string                         `json:"channel" mapstructure:"channel"`
	Message string                         `json:"message" mapstructure:"message"`
	Timeout *int                           `json:"timeout,omitempty" mapstructure:"timeout,omitempty"`
	Buttons []SendAndWaitForResponseButton `json:"buttons" mapstructure:"buttons"`
}

type SendAndWaitForResponseButton struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type SendAndWaitForResponseMetadata struct {
	Channel        *ChannelMetadata `json:"channel,omitempty" mapstructure:"channel,omitempty"`
	MessageTS      *string          `json:"messageTS,omitempty" mapstructure:"messageTS,omitempty"`
	SubscriptionID *string          `json:"subscriptionID,omitempty" mapstructure:"subscriptionID,omitempty"`
}

func (c *SendAndWaitForResponse) Name() string {
	return "slack.sendAndWaitForResponse"
}

func (c *SendAndWaitForResponse) Label() string {
	return "Send and Wait for Response"
}

func (c *SendAndWaitForResponse) Description() string {
	return "Send a message with buttons and wait for a user response"
}

func (c *SendAndWaitForResponse) Documentation() string {
	return `The Send and Wait for Response component sends a Slack message with up to 4 buttons and pauses execution until one button is clicked or a timeout is reached.

## Use Cases

- **Manual approvals**: Ask someone to approve or reject before continuing
- **Human-in-the-loop flows**: Pause automation until a person confirms an option
- **Structured responses**: Gather a predefined selection instead of free-text input

## Configuration

- **Channel**: Slack channel or DM channel ID (required)
- **Message**: Message text to post (required)
- **Timeout**: Optional timeout in seconds
- **Buttons**: List of 1 to 4 buttons with name and value

## Output Channels

- **Received**: Emitted when a button is clicked; payload includes selected value
- **Timeout**: Emitted when timeout is reached before any click`
}

func (c *SendAndWaitForResponse) Icon() string {
	return "slack"
}

func (c *SendAndWaitForResponse) Color() string {
	return "gray"
}

func (c *SendAndWaitForResponse) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: sendAndWaitChannelReceived, Label: "Received", Description: "A button was clicked"},
		{Name: sendAndWaitChannelTimeout, Label: "Timeout", Description: "No button click received in time"},
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
			Description: "Maximum time to wait in seconds (leave empty to wait indefinitely)",
			Required:    false,
		},
		{
			Name:        "buttons",
			Label:       "Buttons",
			Description: "List of 1 to 4 buttons. Each item needs a name and value.",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Default:     `[{"name":"Approve","value":"approve"},{"name":"Reject","value":"reject"}]`,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Button",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "name",
								Label:    "Button Label",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "value",
								Label:    "Button Value",
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

func (c *SendAndWaitForResponse) Setup(ctx core.SetupContext) error {
	config, err := c.parseAndValidate(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Slack client: %w", err)
	}

	channelInfo, err := client.GetChannelInfo(config.Channel)
	if err != nil {
		return fmt.Errorf("channel validation failed: %w", err)
	}

	return ctx.Metadata.Set(SendAndWaitForResponseMetadata{
		Channel: &ChannelMetadata{
			ID:   channelInfo.ID,
			Name: channelInfo.Name,
		},
	})
}

func (c *SendAndWaitForResponse) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SendAndWaitForResponse) Execute(ctx core.ExecutionContext) error {
	config, err := c.parseAndValidate(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Slack client: %w", err)
	}

	actions := make([]map[string]any, 0, len(config.Buttons))
	for i, button := range config.Buttons {
		actions = append(actions, map[string]any{
			"type": "button",
			"text": map[string]string{
				"type": "plain_text",
				"text": button.Name,
			},
			"value":     button.Value,
			"action_id": fmt.Sprintf("button_%d", i+1),
		})
	}

	blocks := []interface{}{
		map[string]any{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": config.Message,
			},
		},
		map[string]any{
			"type":     "actions",
			"elements": actions,
		},
	}

	response, err := client.PostMessage(ChatPostMessageRequest{
		Channel: config.Channel,
		Text:    config.Message,
		Blocks:  blocks,
	})
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	subscriptionID, err := ctx.Integration.Subscribe(map[string]any{
		"type":         sendAndWaitSubscriptionType,
		"message_ts":   response.TS,
		"execution_id": ctx.ID.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe for button interactions: %w", err)
	}

	metadata := SendAndWaitForResponseMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	ts := response.TS
	subID := subscriptionID.String()
	metadata.MessageTS = &ts
	metadata.SubscriptionID = &subID

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	if config.Timeout != nil {
		timeout := time.Duration(*config.Timeout) * time.Second
		if err := ctx.Requests.ScheduleActionCall(sendAndWaitActionTimeout, map[string]any{}, timeout); err != nil {
			return fmt.Errorf("failed to schedule timeout: %w", err)
		}
	}

	return nil
}

func (c *SendAndWaitForResponse) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *SendAndWaitForResponse) Actions() []core.Action {
	return []core.Action{
		{Name: sendAndWaitActionButtonClick},
		{Name: sendAndWaitActionTimeout},
	}
}

func (c *SendAndWaitForResponse) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case sendAndWaitActionButtonClick:
		return c.handleButtonClick(ctx)
	case sendAndWaitActionTimeout:
		return c.handleTimeout(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *SendAndWaitForResponse) handleButtonClick(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	buttonValue, ok := ctx.Parameters["value"].(string)
	if !ok || buttonValue == "" {
		return errors.New("button value is required")
	}

	config, err := c.parseAndValidate(ctx.Configuration)
	if err != nil {
		return err
	}

	isAllowedValue := slices.ContainsFunc(config.Buttons, func(button SendAndWaitForResponseButton) bool {
		return button.Value == buttonValue
	})
	if !isAllowedValue {
		return fmt.Errorf("button value not allowed: %s", buttonValue)
	}

	c.cleanupSubscription(ctx)

	return ctx.ExecutionState.Emit(
		sendAndWaitChannelReceived,
		"slack.button.clicked",
		[]any{
			map[string]any{
				"value":      buttonValue,
				"clicked_at": time.Now().Format(time.RFC3339),
			},
		},
	)
}

func (c *SendAndWaitForResponse) handleTimeout(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	c.cleanupSubscription(ctx)

	return ctx.ExecutionState.Emit(
		sendAndWaitChannelTimeout,
		"slack.button.timeout",
		[]any{
			map[string]any{
				"timeout_at": time.Now().Format(time.RFC3339),
			},
		},
	)
}

func (c *SendAndWaitForResponse) cleanupSubscription(ctx core.ActionContext) {
	if ctx.Integration == nil {
		return
	}

	metadata := SendAndWaitForResponseMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("failed to decode metadata for subscription cleanup: %v", err)
		}
		return
	}

	if metadata.SubscriptionID == nil || *metadata.SubscriptionID == "" {
		return
	}

	subscriptionID, err := uuid.Parse(*metadata.SubscriptionID)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("invalid subscription ID in metadata: %v", err)
		}
		return
	}

	if err := ctx.Integration.Unsubscribe(subscriptionID); err != nil && ctx.Logger != nil {
		ctx.Logger.Warnf("failed to cleanup Slack subscription %s: %v", subscriptionID.String(), err)
	}
}

func (c *SendAndWaitForResponse) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SendAndWaitForResponse) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *SendAndWaitForResponse) parseAndValidate(configuration any) (*SendAndWaitForResponseConfiguration, error) {
	config := SendAndWaitForResponseConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Channel == "" {
		return nil, errors.New("channel is required")
	}

	if config.Message == "" {
		return nil, errors.New("message is required")
	}

	if len(config.Buttons) == 0 {
		return nil, errors.New("at least one button is required")
	}

	if len(config.Buttons) > 4 {
		return nil, errors.New("maximum of 4 buttons allowed")
	}

	for i, button := range config.Buttons {
		if button.Name == "" {
			return nil, fmt.Errorf("button %d: name is required", i+1)
		}
		if button.Value == "" {
			return nil, fmt.Errorf("button %d: value is required", i+1)
		}
	}

	if config.Timeout != nil && *config.Timeout <= 0 {
		return nil, errors.New("timeout must be a positive number")
	}

	return &config, nil
}
