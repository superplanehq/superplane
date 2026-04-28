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

type KillInvocation struct{}

type KillInvocationSpec struct {
	InvocationID string `json:"invocationId"`
}

func (c *KillInvocation) Name() string {
	return "restate.killInvocation"
}

func (c *KillInvocation) Label() string {
	return "Kill Invocation"
}

func (c *KillInvocation) Description() string {
	return "Force-kill a running Restate invocation without running compensation logic"
}

func (c *KillInvocation) Icon() string {
	return "repeat"
}

func (c *KillInvocation) Color() string {
	return "gray"
}

func (c *KillInvocation) Documentation() string {
	return `The Kill Invocation component force-kills a running invocation in Restate **without** running compensation logic.

Use this as a last resort when Cancel Invocation doesn't work — for example, when the service
endpoint is permanently unavailable and Restate cannot invoke the handler to run cleanup.

## Use Cases

- **Stuck invocations**: Kill invocations that are stuck in retry loops with unreachable endpoints
- **Emergency cleanup**: Force-terminate invocations during incidents when graceful cancellation fails
- **Resource reclamation**: Free up resources held by invocations that cannot complete

## Important Notes

- Kill does **not** run compensation logic — state may be left inconsistent
- Use Cancel Invocation first; only kill if cancellation fails
- Killing is non-blocking — the invocation may not be fully terminated when this component completes

## Outputs

The component emits an event containing:
- ` + "`invocation_id`" + `: The killed invocation ID
- ` + "`status`" + `: "killed"
`
}

func (c *KillInvocation) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *KillInvocation) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "invocationId",
			Label:       "Invocation ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "ID of the invocation to kill (starts with inv_)",
			Placeholder: "inv_...",
		},
	}
}

func (c *KillInvocation) Setup(ctx core.SetupContext) error {
	spec := KillInvocationSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.InvocationID == "" {
		return errors.New("invocationId is required")
	}

	return nil
}

func (c *KillInvocation) Execute(ctx core.ExecutionContext) error {
	spec := KillInvocationSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.KillInvocation(spec.InvocationID)
	if err != nil {
		return fmt.Errorf("failed to kill invocation: %v", err)
	}

	result := map[string]any{
		"invocation_id": spec.InvocationID,
		"status":        "killed",
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"restate.invocation.killed",
		[]any{result},
	)
}

func (c *KillInvocation) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *KillInvocation) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *KillInvocation) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *KillInvocation) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *KillInvocation) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *KillInvocation) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
