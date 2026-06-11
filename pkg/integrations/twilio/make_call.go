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

type MakeCall struct{}

type MakeCallConfig struct {
	To      string `json:"to" mapstructure:"to"`
	Message string `json:"message" mapstructure:"message"`
	Timeout int    `json:"timeout" mapstructure:"timeout"`
}

func (c *MakeCall) Name() string  { return "twilio.makeCall" }
func (c *MakeCall) Label() string { return "Make Call" }

func (c *MakeCall) Description() string {
	return "Place an outbound phone call with a text-to-speech message"
}

func (c *MakeCall) Documentation() string {
	return `The **Make Call** component places an outbound phone call and speaks a text-to-speech message to the recipient.

## Use Cases

- **Incident alerting**: Call the on-call engineer when a critical alert fires
- **Escalation**: Call a backup contact if the primary doesn't answer
- **Notifications**: Voice notifications for time-sensitive events

## Configuration

- **To**: Destination phone number in E.164 format (e.g. +15551234567)
- **Message**: The text message to speak via TTS when the call is answered
- **Timeout**: How many seconds to ring before giving up (default: 30)

## Output Channels

- **answered**: The call was answered (status = completed)
- **no-answer**: The call was not answered (status = no-answer, busy, or failed)

## Output

Returns the call SID, final status, destination number, and call duration.`
}

func (c *MakeCall) Icon() string  { return "phone" }
func (c *MakeCall) Color() string { return "#F22F46" }

func (c *MakeCall) ExampleOutput() map[string]any {
	return getExampleOutput("make_call")
}

func (c *MakeCall) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: "answered", Label: "Answered"},
		{Name: "no-answer", Label: "No Answer"},
	}
}

func (c *MakeCall) Configuration() []configuration.Field {
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
			Name:        "message",
			Label:       "Message",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "Text-to-speech message to speak when the call is answered",
		},
		{
			Name:        "timeout",
			Label:       "Timeout",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     "30",
			Description: "How many seconds to ring before giving up (default: 30)",
		},
	}
}

func (c *MakeCall) Setup(ctx core.SetupContext) error {
	var config MakeCallConfig
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.To) == "" {
		return fmt.Errorf("to is required")
	}
	if !strings.HasPrefix(config.To, "+") {
		return fmt.Errorf("phone number must be in E.164 format (e.g. +15551234567)")
	}
	if strings.TrimSpace(config.Message) == "" {
		return fmt.Errorf("message is required")
	}

	return nil
}

func (c *MakeCall) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *MakeCall) Execute(ctx core.ExecutionContext) error {
	var config MakeCallConfig
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return err
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	resp, err := client.MakeCall(config.To, config.Message, timeout)
	if err != nil {
		return fmt.Errorf("failed to make call: %w", err)
	}

	output := map[string]any{
		"callSid":  resp.SID,
		"status":   resp.Status,
		"to":       resp.To,
		"from":     resp.From,
		"duration": resp.Duration,
	}

	channel := "answered"
	if resp.Status == "no-answer" || resp.Status == "busy" || resp.Status == "failed" || resp.Status == "canceled" {
		channel = "no-answer"
	}

	return ctx.ExecutionState.Emit(channel, "twilio.call.completed", []any{output})
}

func (c *MakeCall) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *MakeCall) Cancel(ctx core.ExecutionContext) error      { return nil }
func (c *MakeCall) Cleanup(ctx core.SetupContext) error         { return nil }
func (c *MakeCall) Hooks() []core.Hook                          { return []core.Hook{} }
func (c *MakeCall) HandleHook(ctx core.ActionHookContext) error { return nil }
