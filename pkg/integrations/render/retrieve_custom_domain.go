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

const RetrieveCustomDomainPayloadType = "render.customDomain"

type RetrieveCustomDomain struct{}

type RetrieveCustomDomainConfiguration struct {
	Service    string `json:"service" mapstructure:"service"`
	DomainName string `json:"domainName" mapstructure:"domainName"`
}

func (c *RetrieveCustomDomain) Name() string {
	return "render.retrieveCustomDomain"
}

func (c *RetrieveCustomDomain) Label() string {
	return "Retrieve Custom Domain"
}

func (c *RetrieveCustomDomain) Description() string {
	return "Retrieve a Render custom domain and its verification status"
}

func (c *RetrieveCustomDomain) Documentation() string {
	return `The Retrieve Custom Domain component fetches a custom domain for a Render service.

## Use Cases

- **DNS verification checks**: Retrieve the latest verification status after triggering Render DNS verification
- **Workflow context**: Use custom domain fields to drive branching decisions in later steps

## Configuration

- **Service**: Render service that owns the custom domain
- **Domain Name**: The custom domain name or ID to retrieve (e.g., ` + "`app.example.com`" + `)

## Output

Emits a ` + "`render.customDomain`" + ` payload with ` + "`id`" + `, ` + "`name`" + `, ` + "`serviceId`" + `, and ` + "`verificationStatus`" + `.`
}

func (c *RetrieveCustomDomain) Icon() string {
	return "globe"
}

func (c *RetrieveCustomDomain) Color() string {
	return "gray"
}

func (c *RetrieveCustomDomain) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RetrieveCustomDomain) Configuration() []configuration.Field {
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
			Description: "Render service that owns the custom domain",
		},
		{
			Name:        "domainName",
			Label:       "Domain Name",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The custom domain to retrieve",
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

func decodeRetrieveCustomDomainConfiguration(cfg any) (RetrieveCustomDomainConfiguration, error) {
	spec := RetrieveCustomDomainConfiguration{}
	if err := mapstructure.Decode(cfg, &spec); err != nil {
		return RetrieveCustomDomainConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Service = strings.TrimSpace(spec.Service)
	spec.DomainName = strings.TrimSpace(spec.DomainName)

	if spec.Service == "" {
		return RetrieveCustomDomainConfiguration{}, fmt.Errorf("service is required")
	}
	if spec.DomainName == "" {
		return RetrieveCustomDomainConfiguration{}, fmt.Errorf("domainName is required")
	}

	return spec, nil
}

func (c *RetrieveCustomDomain) Setup(ctx core.SetupContext) error {
	spec, err := decodeRetrieveCustomDomainConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	return setServiceNodeMetadata(ctx, spec.Service)
}

func (c *RetrieveCustomDomain) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RetrieveCustomDomain) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRetrieveCustomDomainConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	domain, err := client.GetCustomDomain(spec.Service, spec.DomainName)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		RetrieveCustomDomainPayloadType,
		[]any{customDomainPayload(spec.Service, domain)},
	)
}

func (c *RetrieveCustomDomain) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *RetrieveCustomDomain) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RetrieveCustomDomain) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *RetrieveCustomDomain) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *RetrieveCustomDomain) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
