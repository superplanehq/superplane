package render

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const RemoveCustomDomainPayloadType = "render.customDomain.removed"

type RemoveCustomDomain struct{}

type RemoveCustomDomainConfiguration struct {
	Service string `json:"service" mapstructure:"service"`
	Domain  string `json:"domain" mapstructure:"domain"`
}

func (c *RemoveCustomDomain) Name() string {
	return "render.service.removeCustomDomain"
}

func (c *RemoveCustomDomain) Label() string {
	return "Remove Custom Domain"
}

func (c *RemoveCustomDomain) Description() string {
	return "Remove a custom domain from a Render service"
}

func (c *RemoveCustomDomain) Documentation() string {
	return `The Remove Custom Domain component removes a custom domain from a Render service.

## Use Cases

- **Blue/green deployments**: Remove the live domain from the old (blue) service before adding it to the new one
- **Domain cleanup**: Automate domain removal as part of a decommission or rotation workflow

## Configuration

- **Service**: Render service to remove the domain from
- **Domain Name**: The custom domain name to remove (e.g., ` + "`app.example.com`" + `)

## Output

Emits a ` + "`render.customDomain.removed`" + ` payload with ` + "`name`" + ` and ` + "`serviceId`" + `.`
}

func (c *RemoveCustomDomain) Icon() string {
	return "globe"
}

func (c *RemoveCustomDomain) Color() string {
	return "gray"
}

func (c *RemoveCustomDomain) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RemoveCustomDomain) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "service",
			Label:    "Service",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "service",
				},
			},
			Description: "Render service to remove the domain from",
		},
		{
			Name:        "domain",
			Label:       "Domain Name",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The custom domain to remove",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "custom_domain",
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "service",
							ValueFrom: &configuration.ParameterValueFrom{Field: "service"},
						},
					},
				},
			},
		},
	}
}

func decodeRemoveCustomDomainConfiguration(cfg any) (RemoveCustomDomainConfiguration, error) {
	spec := RemoveCustomDomainConfiguration{}
	if err := mapstructure.Decode(cfg, &spec); err != nil {
		return RemoveCustomDomainConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Service = strings.TrimSpace(spec.Service)
	spec.Domain = strings.TrimSpace(spec.Domain)

	if spec.Service == "" {
		return RemoveCustomDomainConfiguration{}, fmt.Errorf("service is required")
	}
	if spec.Domain == "" {
		return RemoveCustomDomainConfiguration{}, fmt.Errorf("domain is required")
	}

	return spec, nil
}

func (c *RemoveCustomDomain) Setup(ctx core.SetupContext) error {
	spec, err := decodeRemoveCustomDomainConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	return setServiceNodeMetadata(ctx, spec.Service)
}

func (c *RemoveCustomDomain) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RemoveCustomDomain) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRemoveCustomDomainConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.RemoveCustomDomain(spec.Service, spec.Domain); err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		RemoveCustomDomainPayloadType,
		[]any{
			map[string]any{
				"name":      spec.Domain,
				"serviceId": spec.Service,
			},
		},
	)
}

func (c *RemoveCustomDomain) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *RemoveCustomDomain) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RemoveCustomDomain) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *RemoveCustomDomain) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *RemoveCustomDomain) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
