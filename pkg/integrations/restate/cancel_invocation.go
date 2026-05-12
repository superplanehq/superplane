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

type CancelInvocation struct{}

type CancelInvocationSpec struct {
	InvocationID string `json:"invocationId"`
}

func (c *CancelInvocation) Name() string {
	return "restate.cancelInvocation"
}

func (c *CancelInvocation) Label() string {
	return "Cancel Invocation"
}

func (c *CancelInvocation) Description() string {
	return "Cancel a running Restate invocation gracefully"
}

func (c *CancelInvocation) Icon() string {
	return "repeat"
}

func (c *CancelInvocation) Color() string {
	return "gray"
}

func (c *CancelInvocation) Documentation() string {
	return `The Cancel Invocation component cancels a running invocation in Restate gracefully.

Cancellation allows the handler to run its compensation logic (sagas) before terminating,
ensuring consistent state. If you need to force-stop without compensation, use Kill Invocation instead.

## Use Cases

- **Graceful rollback**: Cancel in-flight invocations during a rollback, allowing compensation to run
- **Timeout handling**: Cancel invocations that have exceeded an expected duration
- **Incident response**: Cancel active invocations for a misbehaving service

## Important Notes

- Cancellation is **non-blocking** — the invocation may not be fully cancelled when this component completes
- Handlers need compensation logic for cancellation to roll back state correctly
- If cancellation doesn't work (e.g. endpoint unreachable), use Kill Invocation as a fallback

## Outputs

The component emits an event containing:
- ` + "`invocation_id`" + `: The cancelled invocation ID
- ` + "`status`" + `: "cancelled"
`
}

func (c *CancelInvocation) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CancelInvocation) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "invocationId",
			Label:       "Invocation ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "ID of the invocation to cancel (starts with inv_)",
			Placeholder: "inv_...",
		},
	}
}

func (c *CancelInvocation) Setup(ctx core.SetupContext) error {
	spec := CancelInvocationSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.InvocationID == "" {
		return errors.New("invocationId is required")
	}

	return nil
}

func (c *CancelInvocation) Execute(ctx core.ExecutionContext) error {
	spec := CancelInvocationSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.CancelInvocation(spec.InvocationID)
	if err != nil {
		return fmt.Errorf("failed to cancel invocation: %v", err)
	}

	result := map[string]any{
		"invocation_id": spec.InvocationID,
		"status":        "cancelled",
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"restate.invocation.cancelled",
		[]any{result},
	)
}

func (c *CancelInvocation) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CancelInvocation) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CancelInvocation) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CancelInvocation) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CancelInvocation) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CancelInvocation) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
