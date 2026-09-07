package vercel

import (
	"fmt"
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type RemoveDomain struct{}

func (c *RemoveDomain) Name() string {
	return "vercel.removeDomain"
}

func (c *RemoveDomain) Label() string {
	return "Remove Domain"
}

func (c *RemoveDomain) Description() string {
	return "Remove a domain from a Vercel project"
}

func (c *RemoveDomain) Documentation() string {
	return `The Remove Domain component detaches a domain from a Vercel project.

## Use Cases

- **Tenant offboarding**: Detach customer domains when a tenant leaves
- **Environment cleanup**: Remove stale domains as part of teardown workflows

## Configuration

- **Project**: Required. The Vercel project.
- **Domain**: Required. The domain name to remove (e.g. ` + "`www.example.com`" + `).`
}

func (c *RemoveDomain) Icon() string {
	return "globe"
}

func (c *RemoveDomain) Color() string {
	return "gray"
}

func (c *RemoveDomain) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RemoveDomain) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			TypeOptions: &configuration.TypeOptions{Resource: &configuration.ResourceTypeOptions{Type: "project"}},
			Description: "Vercel project to remove the domain from",
		},
		{
			Name:        "domain",
			Label:       "Domain",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "www.example.com",
			Description: "Domain name to remove",
		},
	}
}

func (c *RemoveDomain) Setup(ctx core.SetupContext) error {
	if _, err := decodeDomainConfiguration(ctx.Configuration); err != nil {
		return err
	}
	return nil
}

func (c *RemoveDomain) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeDomainConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.RemoveProjectDomain(spec.Project, spec.Domain); err != nil {
		return fmt.Errorf("failed to remove domain from Vercel project: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		DomainPayloadType,
		[]any{map[string]any{
			"projectId": spec.Project,
			"name":      spec.Domain,
			"removed":   true,
		}},
	)
}

func (c *RemoveDomain) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *RemoveDomain) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RemoveDomain) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *RemoveDomain) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *RemoveDomain) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
