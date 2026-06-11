package twilio

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type SendSMS struct{}

type SendSMSConfig struct {
	To   string `json:"to" mapstructure:"to"`
	Body string `json:"body" mapstructure:"body"`
}

func (c *SendSMS) Name() string  { return "twilio.sendSMS" }
func (c *SendSMS) Label() string { return "Send SMS" }

func (c *SendSMS) Description() string {
	return "Send an outbound SMS message"
}

func (c *SendSMS) Documentation() string {
	return `The **Send SMS** component sends a text message to a phone number.

## Use Cases

- **Alert fallback**: Send an SMS when a phone call goes unanswered
- **Notifications**: Quick status updates via text
- **Confirmations**: Send confirmation codes or summaries

## Configuration

- **To**: Destination phone number in E.164 format (e.g. +15551234567)
- **Body**: The SMS message text (max 1600 characters)

## Output

Returns the message SID, delivery status, and message details.`
}

func (c *SendSMS) Icon() string  { return "message-square" }
func (c *SendSMS) Color() string { return "#F22F46" }

func (c *SendSMS) ExampleOutput() map[string]any {
	return getExampleOutput("send_sms")
}

func (c *SendSMS) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *SendSMS) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "to",
			Label:       "To",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "+15551234567",
			Description: "Destination phone number (E.164 format)",
		},
		{
			Name:        "body",
			Label:       "Message",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "SMS message text (max 1600 characters)",
		},
	}
}

func (c *SendSMS) Setup(ctx core.SetupContext) error {
	var config SendSMSConfig
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.To) == "" {
		return fmt.Errorf("to is required")
	}
	if !strings.HasPrefix(config.To, "+") {
		return fmt.Errorf("phone number must be in E.164 format (e.g. +15551234567)")
	}
	if strings.TrimSpace(config.Body) == "" {
		return fmt.Errorf("body is required")
	}
	if len(config.Body) > 1600 {
		return fmt.Errorf("message exceeds maximum length of 1600 characters")
	}

	return nil
}

func (c *SendSMS) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SendSMS) Execute(ctx core.ExecutionContext) error {
	var config SendSMSConfig
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return err
	}

	resp, err := client.SendSMS(config.To, config.Body)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	output := map[string]any{
		"messageSid": resp.SID,
		"status":     resp.Status,
		"to":         resp.To,
		"from":       resp.From,
		"body":       resp.Body,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"twilio.sms.sent",
		[]any{output},
	)
}

func (c *SendSMS) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *SendSMS) Cancel(ctx core.ExecutionContext) error      { return nil }
func (c *SendSMS) Cleanup(ctx core.SetupContext) error         { return nil }
func (c *SendSMS) Hooks() []core.Hook                          { return []core.Hook{} }
func (c *SendSMS) HandleHook(ctx core.ActionHookContext) error { return nil }
