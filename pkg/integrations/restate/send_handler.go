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

type SendHandler struct{}

type SendHandlerSpec struct {
	Service        string `json:"service"`
	Handler        string `json:"handler"`
	Payload        string `json:"payload"`
	IdempotencyKey string `json:"idempotencyKey"`
}

func (c *SendHandler) Name() string {
	return "restate.sendHandler"
}

func (c *SendHandler) Label() string {
	return "Send Handler"
}

func (c *SendHandler) Description() string {
	return "Send a fire-and-forget invocation to a Restate service handler"
}

func (c *SendHandler) Icon() string {
	return "repeat"
}

func (c *SendHandler) Color() string {
	return "gray"
}

func (c *SendHandler) Documentation() string {
	return `The Send Handler component sends a fire-and-forget invocation to a Restate handler without waiting for the response.

## Use Cases

- **Async processing**: Trigger a long-running handler without blocking the workflow
- **Event-driven workflows**: Send events to Restate handlers for background processing
- **Decoupled execution**: Start an invocation and track it separately via its invocation ID

## Outputs

The component emits an event containing:
- ` + "`invocation_id`" + `: The Restate invocation identifier (starts with inv_)
- ` + "`status`" + `: The acceptance status (e.g. "Accepted")
- ` + "`service`" + `: The target service name
- ` + "`handler`" + `: The target handler name
`
}

func (c *SendHandler) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *SendHandler) Configuration() []configuration.Field {
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

func (c *SendHandler) Setup(ctx core.SetupContext) error {
	spec := SendHandlerSpec{}
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

	return nil
}

func (c *SendHandler) Execute(ctx core.ExecutionContext) error {
	spec := SendHandlerSpec{}
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

	response, err := client.SendHandler(spec.Service, spec.Handler, payload, spec.IdempotencyKey)
	if err != nil {
		return fmt.Errorf("failed to send handler invocation: %v", err)
	}

	result := map[string]any{
		"invocation_id": response.InvocationID,
		"status":        response.Status,
		"service":       spec.Service,
		"handler":       spec.Handler,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"restate.invocation.sent",
		[]any{result},
	)
}

func (c *SendHandler) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SendHandler) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SendHandler) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *SendHandler) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *SendHandler) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *SendHandler) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
