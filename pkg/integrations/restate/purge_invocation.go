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

type PurgeInvocation struct{}

type PurgeInvocationSpec struct {
	InvocationID string `json:"invocationId"`
}

func (c *PurgeInvocation) Name() string {
	return "restate.purgeInvocation"
}

func (c *PurgeInvocation) Label() string {
	return "Purge Invocation"
}

func (c *PurgeInvocation) Description() string {
	return "Purge a completed Restate invocation and its associated data"
}

func (c *PurgeInvocation) Icon() string {
	return "repeat"
}

func (c *PurgeInvocation) Color() string {
	return "gray"
}

func (c *PurgeInvocation) Documentation() string {
	return `The Purge Invocation component purges a completed invocation and all its associated data from the Restate server.

## Use Cases

- **Data cleanup**: Remove invocation data after it's no longer needed
- **Post-migration cleanup**: Purge old invocations after a service migration
- **Storage management**: Free up storage by removing completed invocation journals

## Important Notes

- Only completed (succeeded, failed, or killed) invocations can be purged
- Purging removes the invocation journal and response data permanently
- This action is irreversible

## Outputs

The component emits an event containing:
- ` + "`invocation_id`" + `: The purged invocation ID
- ` + "`status`" + `: "purged"
`
}

func (c *PurgeInvocation) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PurgeInvocation) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "invocationId",
			Label:       "Invocation ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "ID of the completed invocation to purge (starts with inv_)",
			Placeholder: "inv_...",
		},
	}
}

func (c *PurgeInvocation) Setup(ctx core.SetupContext) error {
	spec := PurgeInvocationSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.InvocationID == "" {
		return errors.New("invocationId is required")
	}

	return nil
}

func (c *PurgeInvocation) Execute(ctx core.ExecutionContext) error {
	spec := PurgeInvocationSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.PurgeInvocation(spec.InvocationID)
	if err != nil {
		return fmt.Errorf("failed to purge invocation: %v", err)
	}

	result := map[string]any{
		"invocation_id": spec.InvocationID,
		"status":        "purged",
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"restate.invocation.purged",
		[]any{result},
	)
}

func (c *PurgeInvocation) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *PurgeInvocation) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PurgeInvocation) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *PurgeInvocation) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *PurgeInvocation) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *PurgeInvocation) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
