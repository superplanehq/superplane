package coolify

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ListApplicationsPayloadType = "coolify.applications.listed"

type ListApplications struct{}

func (c *ListApplications) Name() string {
	return "coolify.listApplications"
}

func (c *ListApplications) Label() string {
	return "List Applications"
}

func (c *ListApplications) Description() string {
	return "List all applications visible to the Coolify API token"
}

func (c *ListApplications) Documentation() string {
	return `The List Applications component fetches every application available to the Coolify API token and emits the list on the default output channel.

## How It Works

1. Calls ` + "`GET /api/v1/applications`" + ` on the configured Coolify instance
2. Emits the array of applications, each with ` + "`uuid`" + `, ` + "`name`" + `, ` + "`status`" + `, ` + "`fqdn`" + `, ` + "`description`" + `, ` + "`gitRepository`" + `, and ` + "`gitBranch`" + `

## Use Cases

- Iterate over every application with a downstream ` + "`forEach`" + ` to act on each (e.g. restart all)
- Filter applications by status before invoking lifecycle actions
- Surface application details into other workflows
`
}

func (c *ListApplications) Icon() string {
	return "coolify"
}

func (c *ListApplications) Color() string {
	return "gray"
}

func (c *ListApplications) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ListApplications) Configuration() []configuration.Field {
	return nil
}

func (c *ListApplications) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *ListApplications) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ListApplications) Execute(ctx core.ExecutionContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	applications, err := client.ListApplications()
	if err != nil {
		return err
	}

	payload := map[string]any{
		"applications": applicationsToPayloads(applications),
		"count":        len(applications),
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, ListApplicationsPayloadType, []any{payload})
}

func (c *ListApplications) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ListApplications) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *ListApplications) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *ListApplications) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ListApplications) Cleanup(ctx core.SetupContext) error {
	return nil
}

func applicationsToPayloads(applications []Application) []map[string]any {
	out := make([]map[string]any, 0, len(applications))
	for _, app := range applications {
		out = append(out, applicationToPayload(app))
	}
	return out
}

func applicationToPayload(app Application) map[string]any {
	payload := map[string]any{
		"uuid": app.UUID,
		"name": app.Name,
	}
	if app.Status != "" {
		payload["status"] = app.Status
	}
	if app.FQDN != "" {
		payload["fqdn"] = app.FQDN
	}
	if app.Description != "" {
		payload["description"] = app.Description
	}
	if app.GitRepo != "" {
		payload["gitRepository"] = app.GitRepo
	}
	if app.GitBranch != "" {
		payload["gitBranch"] = app.GitBranch
	}
	return payload
}
