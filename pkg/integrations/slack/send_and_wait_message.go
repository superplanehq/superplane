package slack

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type SendAndWaitMessage struct{}

type SendAndWaitMessageMetadata struct {
	Channel *ChannelMetadata `json:"channel" mapstructure:"channel"`
}

type SendAndWaitMessageConfiguration struct {
	Channel string   `json:"channel" mapstructure:"channel"`
	Message string   `json:"message" mapstructure:"message"`
	Timeout int      `json:"timeout" mapstructure:"timeout"`
	Buttons []Button `json:"buttons" mapstructure:"buttons"`
}

type Button struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

func (c *SendAndWaitMessage) Name() string {
	return "slack.sendAndWaitMessage"
}

func (c *SendAndWaitMessage) Label() string {
	return "Send Message and Wait"
}

func (c *SendAndWaitMessage) Description() string {
	return "Send a message with buttons and wait for a response"
}

func (c *SendAndWaitMessage) Documentation() string {
	return `Send a message to a Slack channel or DM and wait for the user to click one of the configured buttons.

## Use Cases

- **Request approval**: Request approval or input from a user in Slack before applying or deploying.
- **Pause workflow**: Pause a workflow until a human selects an option.
- **Structured reply**: Implement flows that need a structured reply via buttons.

## Configuration

- **Channel**: Slack channel or DM channel name to post to.
- **Message**: Message text (supports Slack formatting).
- **Timeout**: Maximum time to wait in seconds (optional).
- **Buttons**: Set of 1–4 items, each with name (label) and value.

## Outputs

- **Received**: Emits when the user clicks a button; payload includes the selected button's value.
- **Timeout**: Emits when no button click is received within the configured timeout.
`
}

func (c *SendAndWaitMessage) Icon() string {
	return "slack"
}

func (c *SendAndWaitMessage) Color() string {
	return "gray"
}

func (c *SendAndWaitMessage) ExampleOutput() map[string]any {
	return map[string]any{
		"value": "approved",
	}
}

func (c *SendAndWaitMessage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  "received",
			Label: "Received",
		},
		{
			Name:  "timeout",
			Label: "Timeout",
		},
	}
}

func (c *SendAndWaitMessage) Setup(ctx core.SetupContext) error {
	var config SendAndWaitMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	if len(config.Buttons) == 0 {
		return fmt.Errorf("at least one button must be configured")
	}

	if len(config.Buttons) > 4 {
		return fmt.Errorf("a maximum of 4 buttons can be configured")
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Slack client: %w", err)
	}

	channelInfo, err := client.GetChannelInfo(config.Channel)
	if err != nil {
		return fmt.Errorf("channel validation failed: %w", err)
	}

	metadata := SendAndWaitMessageMetadata{
		Channel: &ChannelMetadata{
			ID:   channelInfo.ID,
			Name: channelInfo.Name,
		},
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	_, err = ctx.Integration.Subscribe(SubscriptionConfiguration{
		EventTypes: []string{"interaction"},
	})
	return err
}

func (c *SendAndWaitMessage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SendAndWaitMessage) Execute(ctx core.ExecutionContext) error {
	var config SendAndWaitMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Slack client: %w", err)
	}

	buttons := []interface{}{}
	for i, b := range config.Buttons {
		if i >= 4 {
			break
		}
		buttons = append(buttons, map[string]any{
			"type": "button",
			"text": map[string]any{
				"type": "plain_text",
				"text": b.Name,
			},
			"action_id": fmt.Sprintf("button_%d", i),
			"value":     fmt.Sprintf("sp_exec:%s:%s", ctx.ID.String(), b.Value),
		})
	}

	blocks := []interface{}{
		map[string]any{
			"type": "section",
			"text": map[string]any{
				"type": "mrkdwn",
				"text": config.Message,
			},
		},
		map[string]any{
			"type":     "actions",
			"elements": buttons,
		},
	}

	_, err = client.PostMessage(ChatPostMessageRequest{
		Channel: config.Channel,
		Text:    config.Message,
		Blocks:  blocks,
	})

	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Store execution ID in KV for lookup when interaction arrives
	err = ctx.ExecutionState.SetKV("execution_id", ctx.ID.String())
	if err != nil {
		return fmt.Errorf("failed to set execution ID in KV: %w", err)
	}

	if config.Timeout > 0 {
		err := ctx.Requests.ScheduleActionCall("timeout", nil, time.Duration(config.Timeout)*time.Second)
		if err != nil {
			return fmt.Errorf("failed to schedule timeout: %w", err)
		}
	}

	return nil
}

func (c *SendAndWaitMessage) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "timeout",
			UserAccessible: false,
		},
	}
}

func (c *SendAndWaitMessage) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "timeout" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	return ctx.ExecutionState.Emit("timeout", "slack.message.timeout", []any{map[string]any{"status": "timeout"}})
}

func (c *SendAndWaitMessage) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	payload, ok := ctx.Message.(map[string]any)
	if !ok {
		return nil
	}

	actions, ok := payload["actions"].([]any)
	if !ok || len(actions) == 0 {
		return nil
	}

	for _, actionAny := range actions {
		action, ok := actionAny.(map[string]any)
		if !ok {
			continue
		}

		value, ok := action["value"].(string)
		if !ok || !strings.HasPrefix(value, "sp_exec:") {
			continue
		}

		parts := strings.SplitN(value, ":", 3)
		if len(parts) < 3 {
			continue
		}

		execID := parts[1]
		buttonValue := parts[2]

		executionCtx, err := ctx.FindExecutionByKV("execution_id", execID)
		if err != nil || executionCtx == nil {
			continue
		}

		if executionCtx.ExecutionState.IsFinished() {
			continue
		}

		return executionCtx.ExecutionState.Emit("received", "slack.button.clicked", []any{map[string]any{"value": buttonValue}})
	}

	return nil
}

func (c *SendAndWaitMessage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *SendAndWaitMessage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SendAndWaitMessage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *SendAndWaitMessage) Configuration() []configuration.Field {
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
			Name:     "timeout",
			Label:    "Timeout (seconds)",
			Type:     configuration.FieldTypeNumber,
			Required: false,
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
								Label:    "Name",
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
