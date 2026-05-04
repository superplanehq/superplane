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

const TriggerDNSConfigurationPayloadType = "render.dnsConfiguration.verification.requested"

type TriggerDNSConfiguration struct{}

type TriggerDNSConfigurationConfiguration struct {
	Service    string `json:"service" mapstructure:"service"`
	DomainName string `json:"domainName" mapstructure:"domainName"`
}

func (c *TriggerDNSConfiguration) Name() string {
	return "render.triggerDNSConfiguration"
}

func (c *TriggerDNSConfiguration) Label() string {
	return "Trigger DNS Configuration"
}

func (c *TriggerDNSConfiguration) Description() string {
	return "Trigger DNS verification for a Render custom domain"
}

func (c *TriggerDNSConfiguration) Documentation() string {
	return `The Trigger DNS Configuration component asks Render to verify the DNS configuration for a custom domain.

## Use Cases

- **Custom domain provisioning**: Trigger Render verification after DNS records have been created or updated
- **Recovery workflows**: Retry verification for a domain that is still unverified after DNS propagation

## Configuration

- **Service**: Render service that owns the custom domain
- **Domain Name**: The custom domain name or ID to verify (e.g., ` + "`app.example.com`" + `)

## Output

Emits a ` + "`render.dnsConfiguration.verification.requested`" + ` payload with ` + "`name`" + `, ` + "`serviceId`" + `, and ` + "`status`" + `.`
}

func (c *TriggerDNSConfiguration) Icon() string {
	return "shield-check"
}

func (c *TriggerDNSConfiguration) Color() string {
	return "gray"
}

func (c *TriggerDNSConfiguration) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *TriggerDNSConfiguration) Configuration() []configuration.Field {
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
			Description: "Render service whose custom domain should be verified",
		},
		{
			Name:        "domainName",
			Label:       "Domain Name",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The custom domain to verify",
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

func decodeTriggerDNSConfigurationConfiguration(cfg any) (TriggerDNSConfigurationConfiguration, error) {
	spec := TriggerDNSConfigurationConfiguration{}
	if err := mapstructure.Decode(cfg, &spec); err != nil {
		return TriggerDNSConfigurationConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Service = strings.TrimSpace(spec.Service)
	spec.DomainName = strings.TrimSpace(spec.DomainName)

	if spec.Service == "" {
		return TriggerDNSConfigurationConfiguration{}, fmt.Errorf("service is required")
	}
	if spec.DomainName == "" {
		return TriggerDNSConfigurationConfiguration{}, fmt.Errorf("domainName is required")
	}

	return spec, nil
}

func (c *TriggerDNSConfiguration) Setup(ctx core.SetupContext) error {
	spec, err := decodeTriggerDNSConfigurationConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	return setServiceNodeMetadata(ctx, spec.Service)
}

func (c *TriggerDNSConfiguration) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *TriggerDNSConfiguration) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeTriggerDNSConfigurationConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if _, err := client.VerifyCustomDomain(spec.Service, spec.DomainName); err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		TriggerDNSConfigurationPayloadType,
		[]any{
			map[string]any{
				"name":      spec.DomainName,
				"serviceId": spec.Service,
				"status":    "accepted",
			},
		},
	)
}

func (c *TriggerDNSConfiguration) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *TriggerDNSConfiguration) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *TriggerDNSConfiguration) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *TriggerDNSConfiguration) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *TriggerDNSConfiguration) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
