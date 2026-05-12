package restate

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListServices struct{}

func (c *ListServices) Name() string {
	return "restate.listServices"
}

func (c *ListServices) Label() string {
	return "List Services"
}

func (c *ListServices) Description() string {
	return "List all services registered with the Restate server"
}

func (c *ListServices) Icon() string {
	return "repeat"
}

func (c *ListServices) Color() string {
	return "gray"
}

func (c *ListServices) Documentation() string {
	return `The List Services component retrieves all services currently registered with the Restate server.

## Use Cases

- **Inventory checks**: Verify which services are deployed before running a workflow
- **Health monitoring**: List all registered services as part of a status check
- **Post-deploy validation**: Confirm a newly deployed service appears in the registry

## Outputs

The component emits an event containing:
- ` + "`services`" + `: Array of service objects, each containing name, revision, type, deployment ID, and handlers
- ` + "`count`" + `: Total number of registered services
`
}

func (c *ListServices) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ListServices) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (c *ListServices) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *ListServices) Execute(ctx core.ExecutionContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	services, err := client.ListAllServices()
	if err != nil {
		return fmt.Errorf("failed to list services: %v", err)
	}

	result := map[string]any{
		"services": services,
		"count":    len(services),
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"restate.services",
		[]any{result},
	)
}

func (c *ListServices) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ListServices) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ListServices) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *ListServices) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ListServices) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ListServices) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
