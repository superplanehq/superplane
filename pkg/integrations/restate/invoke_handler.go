package restate

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type InvokeHandler struct{}

type InvokeHandlerSpec struct {
	Service        string `json:"service"`
	Handler        string `json:"handler"`
	Payload        string `json:"payload"`
	IdempotencyKey string `json:"idempotencyKey"`
}

func (c *InvokeHandler) Name() string {
	return "restate.invokeHandler"
}

func (c *InvokeHandler) Label() string {
	return "Invoke Handler"
}

func (c *InvokeHandler) Description() string {
	return "Invoke a Restate service handler synchronously and wait for the response"
}

func (c *InvokeHandler) Icon() string {
	return "repeat"
}

func (c *InvokeHandler) Color() string {
	return "gray"
}

func (c *InvokeHandler) Documentation() string {
	return `The Invoke Handler component calls a Restate service handler and waits for the response.

## Use Cases

- **Workflow steps**: Execute a durable handler as part of a multi-step workflow
- **Data processing**: Invoke a handler to transform or validate data
- **Idempotent operations**: Use an idempotency key to ensure exactly-once execution

## Outputs

The component emits an event containing:
- ` + "`service`" + `: The service name
- ` + "`handler`" + `: The handler name
- ` + "`status_code`" + `: The HTTP status code of the response
- ` + "`response`" + `: The response body from the handler
- ` + "`idempotency_key`" + `: The idempotency key used (if any)
`
}

func (c *InvokeHandler) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *InvokeHandler) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "service",
			Label:       "Service",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name of the Restate service (or VirtualObject/Workflow, including key if needed, e.g. MyObject/myKey)",
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

func (c *InvokeHandler) Setup(ctx core.SetupContext) error {
	spec := InvokeHandlerSpec{}
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

func (c *InvokeHandler) Execute(ctx core.ExecutionContext) error {
	spec := InvokeHandlerSpec{}
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

	response, err := client.InvokeHandler(spec.Service, spec.Handler, payload, spec.IdempotencyKey)
	if err != nil {
		return fmt.Errorf("failed to invoke handler: %v", err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ctx.ExecutionState.Fail("invocation_error", fmt.Sprintf("handler returned status %d: %s", response.StatusCode, string(response.Body)))
	}

	var responseData any
	if len(response.Body) > 0 {
		if err := json.Unmarshal(response.Body, &responseData); err != nil {
			// If response is not JSON, use it as a string
			responseData = string(response.Body)
		}
	}

	result := map[string]any{
		"service":     spec.Service,
		"handler":     spec.Handler,
		"status_code": response.StatusCode,
		"response":    responseData,
	}

	if spec.IdempotencyKey != "" {
		result["idempotency_key"] = spec.IdempotencyKey
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"restate.invocation.response",
		[]any{result},
	)
}

func (c *InvokeHandler) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *InvokeHandler) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *InvokeHandler) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *InvokeHandler) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *InvokeHandler) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *InvokeHandler) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
