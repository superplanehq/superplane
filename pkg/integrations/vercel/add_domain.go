package vercel

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type AddDomain struct{}

type AddDomainConfiguration struct {
	Project string `json:"project" mapstructure:"project"`
	Domain  string `json:"domain" mapstructure:"domain"`
}

func (c *AddDomain) Name() string {
	return "vercel.addDomain"
}

func (c *AddDomain) Label() string {
	return "Add Domain"
}

func (c *AddDomain) Description() string {
	return "Add a domain to a Vercel project"
}

func (c *AddDomain) Documentation() string {
	return `The Add Domain component assigns a domain to a Vercel project.

## Use Cases

- **Tenant provisioning**: Attach a customer domain when onboarding a tenant
- **Environment automation**: Wire up domains as part of project setup

## Configuration

- **Project**: Required. The Vercel project.
- **Domain**: Required. The domain name (e.g. ` + "`www.example.com`" + `).

## Notes

- The domain must be added to the Vercel account or already owned there
- DNS records may need updating; check the verification state in the output`
}

func (c *AddDomain) Icon() string {
	return "globe"
}

func (c *AddDomain) Color() string {
	return "gray"
}

func (c *AddDomain) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddDomain) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			TypeOptions: &configuration.TypeOptions{Resource: &configuration.ResourceTypeOptions{Type: "project"}},
			Description: "Vercel project to add the domain to",
		},
		{
			Name:        "domain",
			Label:       "Domain",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "www.example.com",
			Description: "Domain name to add",
		},
	}
}

func decodeDomainConfiguration(configuration any) (AddDomainConfiguration, error) {
	spec := AddDomainConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return spec, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Project = strings.TrimSpace(spec.Project)
	spec.Domain = strings.TrimSpace(strings.ToLower(spec.Domain))

	if spec.Project == "" {
		return spec, fmt.Errorf("project is required")
	}
	if spec.Domain == "" {
		return spec, fmt.Errorf("domain is required")
	}

	return spec, nil
}

func (c *AddDomain) Setup(ctx core.SetupContext) error {
	_, err := decodeDomainConfiguration(ctx.Configuration)
	return err
}

func (c *AddDomain) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeDomainConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	domain, err := client.AddProjectDomain(spec.Project, spec.Domain)
	if err != nil {
		return fmt.Errorf("failed to add domain to Vercel project: %w", err)
	}

	data := map[string]any{
		"projectId": spec.Project,
		"name":      spec.Domain,
	}
	if domain != nil {
		data["verified"] = domain.Verified
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		DomainPayloadType,
		[]any{data},
	)
}

func (c *AddDomain) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *AddDomain) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddDomain) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AddDomain) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AddDomain) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
