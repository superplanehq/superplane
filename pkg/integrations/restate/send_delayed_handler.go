package restate

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type SendDelayedHandler struct{}

type SendDelayedHandlerSpec struct {
	Service        string `json:"service"`
	Handler        string `json:"handler"`
	Delay          string `json:"delay"`
	Payload        string `json:"payload"`
	IdempotencyKey string `json:"idempotencyKey"`
}

func (c *SendDelayedHandler) Name() string {
	return "restate.sendDelayedHandler"
}

func (c *SendDelayedHandler) Label() string {
	return "Send Delayed Handler"
}

func (c *SendDelayedHandler) Description() string {
	return "Send a delayed fire-and-forget invocation to a Restate service handler"
}

func (c *SendDelayedHandler) Icon() string {
	return "repeat"
}

func (c *SendDelayedHandler) Color() string {
	return "gray"
}

func (c *SendDelayedHandler) Documentation() string {
	return `The Send Delayed Handler component schedules a fire-and-forget invocation to a Restate handler with a configurable delay.

## Use Cases

- **Scheduled processing**: Delay handler execution for a specified duration
- **Retry with backoff**: Schedule retries with increasing delays
- **Timed workflows**: Execute handler steps at specific intervals

## Delay Format

The delay can be specified in humantime format (e.g. ` + "`10s`" + `, ` + "`5m`" + `, ` + "`1h`" + `) or ISO8601 duration (e.g. ` + "`PT10S`" + `, ` + "`PT5M`" + `).

## Outputs

The component emits an event containing:
- ` + "`invocation_id`" + `: The Restate invocation identifier
- ` + "`status`" + `: The acceptance status
- ` + "`service`" + `: The target service name
- ` + "`handler`" + `: The target handler name
- ` + "`delay`" + `: The configured delay
`
}

func (c *SendDelayedHandler) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *SendDelayedHandler) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "service",
			Label:       "Service",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name of the Restate service (or VirtualObject/Workflow, including key if needed)",
		},
		{
			Name:        "handler",
			Label:       "Handler",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name of the handler to invoke",
		},
		{
			Name:        "delay",
			Label:       "Delay",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Delay before invocation (e.g. 10s, 5m, 1h, or ISO8601 like PT10S)",
			Placeholder: "30s",
		},
		{
			Name:        "payload",
			Label:       "Payload",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "JSON payload to send to the handler",
		},
		{
			Name:        "idempotencyKey",
			Label:       "Idempotency Key",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional idempotency key for exactly-once execution",
		},
	}
}

func (c *SendDelayedHandler) Setup(ctx core.SetupContext) error {
	spec := SendDelayedHandlerSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Service == "" {
		return errors.New("service is required")
	}

	if spec.Handler == "" {
		return errors.New("handler is required")
	}

	if spec.Delay == "" {
		return errors.New("delay is required")
	}

	return nil
}

func (c *SendDelayedHandler) Execute(ctx core.ExecutionContext) error {
	spec := SendDelayedHandlerSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	var payload []byte
	if spec.Payload != "" {
		payload = []byte(spec.Payload)
	}

	response, err := client.SendDelayedHandler(spec.Service, spec.Handler, payload, spec.Delay, spec.IdempotencyKey)
	if err != nil {
		return fmt.Errorf("failed to send delayed handler invocation: %v", err)
	}

	result := map[string]any{
		"invocation_id": response.InvocationID,
		"status":        response.Status,
		"service":       spec.Service,
		"handler":       spec.Handler,
		"delay":         spec.Delay,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"restate.invocation.delayed",
		[]any{result},
	)
}

func (c *SendDelayedHandler) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SendDelayedHandler) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SendDelayedHandler) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *SendDelayedHandler) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *SendDelayedHandler) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *SendDelayedHandler) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
