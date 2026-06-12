package coolify

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ListServicesPayloadType = "coolify.services.listed"

type ListServices struct{}

func (c *ListServices) Name() string {
	return "coolify.listServices"
}

func (c *ListServices) Label() string {
	return "List Services"
}

func (c *ListServices) Description() string {
	return "List all services visible to the Coolify API token"
}

func (c *ListServices) Documentation() string {
	return `The List Services component fetches every Coolify service (one-click and custom Docker Compose stacks) available to the API token and emits the list on the default output channel.

## How It Works

1. Calls ` + "`GET /api/v1/services`" + ` on the configured Coolify instance
2. Emits the array of services, each with ` + "`uuid`" + `, ` + "`name`" + `, ` + "`status`" + `, ` + "`fqdn`" + `, ` + "`description`" + `, and ` + "`serverUuid`" + `

## Use Cases

- Iterate over every service with a downstream ` + "`forEach`" + ` to act on each
- Filter services by status before invoking lifecycle actions
- Inventory services across an environment
`
}

func (c *ListServices) Icon() string {
	return "coolify"
}

func (c *ListServices) Color() string {
	return "gray"
}

func (c *ListServices) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ListServices) Configuration() []configuration.Field {
	return nil
}

func (c *ListServices) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *ListServices) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ListServices) Execute(ctx core.ExecutionContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	services, err := client.ListServices()
	if err != nil {
		return err
	}

	payload := map[string]any{
		"services": servicesToPayloads(services),
		"count":    len(services),
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, ListServicesPayloadType, []any{payload})
}

func (c *ListServices) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ListServices) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *ListServices) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *ListServices) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ListServices) Cleanup(ctx core.SetupContext) error {
	return nil
}

func servicesToPayloads(services []Service) []map[string]any {
	out := make([]map[string]any, 0, len(services))
	for _, svc := range services {
		out = append(out, serviceToPayload(svc))
	}
	return out
}

func serviceToPayload(svc Service) map[string]any {
	payload := map[string]any{
		"uuid": svc.UUID,
		"name": svc.Name,
	}
	if svc.Status != "" {
		payload["status"] = svc.Status
	}
	if svc.FQDN != "" {
		payload["fqdn"] = svc.FQDN
	}
	if svc.Description != "" {
		payload["description"] = svc.Description
	}
	if svc.ServerUUID != "" {
		payload["serverUuid"] = svc.ServerUUID
	}
	return payload
}
