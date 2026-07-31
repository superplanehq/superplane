package dokploy

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ListApplicationsPayloadType = "dokploy.applications.listed"

type ListApplications struct{}

func (d *ListApplications) Name() string {
	return "dokploy.listApplications"
}

func (d *ListApplications) Label() string {
	return "List Applications"
}

func (d *ListApplications) Description() string {
	return "List all applications visible to the Dokploy API key"
}

func (d *ListApplications) Documentation() string {
	return `The List Applications component fetches every application available to the Dokploy API key and emits the list on the default output channel.

## How It Works

1. Calls ` + "`GET /api/project.all`" + ` on the configured Dokploy instance
2. Flattens the returned projects, which nest applications under environments, into a single list
3. Emits the array of applications, each with ` + "`applicationId`" + `, ` + "`name`" + `, ` + "`status`" + `, and the ` + "`project`" + ` and ` + "`environment`" + ` it belongs to

Dokploy exposes no endpoint that lists applications directly, so they are read through their projects. Application names are only unique within an environment, which is why the project and environment are included on every entry.

## Use Cases

- Iterate over every application with a downstream ` + "`forEach`" + ` to act on each
- Filter applications by status before deploying
- Resolve an application's ID by name for use in later steps
`
}

func (d *ListApplications) Icon() string {
	return "dokploy"
}

func (d *ListApplications) Color() string {
	return "gray"
}

func (d *ListApplications) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *ListApplications) Configuration() []configuration.Field {
	return nil
}

func (d *ListApplications) Setup(ctx core.SetupContext) error {
	return nil
}

func (d *ListApplications) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *ListApplications) Execute(ctx core.ExecutionContext) error {
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

func (d *ListApplications) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *ListApplications) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (d *ListApplications) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (d *ListApplications) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *ListApplications) Cleanup(ctx core.SetupContext) error {
	return nil
}

func applicationsToPayloads(applications []ApplicationSummary) []map[string]any {
	out := make([]map[string]any, 0, len(applications))
	for _, application := range applications {
		out = append(out, applicationToPayload(application))
	}
	return out
}

func applicationToPayload(application ApplicationSummary) map[string]any {
	payload := map[string]any{
		"applicationId": application.ApplicationID,
		"name":          application.Name,
	}
	if application.Status != "" {
		payload["status"] = application.Status
	}
	if application.ProjectID != "" {
		payload["projectId"] = application.ProjectID
	}
	if application.ProjectName != "" {
		payload["project"] = application.ProjectName
	}
	if application.EnvironmentID != "" {
		payload["environmentId"] = application.EnvironmentID
	}
	if application.EnvironmentName != "" {
		payload["environment"] = application.EnvironmentName
	}
	return payload
}
