package render

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	AddCustomDomainPayloadType  = "render.customDomain.added"
	AddCustomDomainPollInterval = time.Minute
	addCustomDomainExecutionKey = "custom_domain_id"

	customDomainVerificationStatusVerified = "verified"
	customDomainVerificationStatusFailed   = "failed"
)

type AddCustomDomain struct{}

type AddCustomDomainConfiguration struct {
	Service             string `json:"service" mapstructure:"service"`
	Domain              string `json:"domain" mapstructure:"domain"`
	WaitForVerification bool   `json:"waitForVerification" mapstructure:"waitForVerification"`
}

type AddCustomDomainExecutionMetadata struct {
	CustomDomain *CustomDomainMetadata `json:"customDomain" mapstructure:"customDomain"`
}

type CustomDomainMetadata struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	ServiceID          string `json:"serviceId"`
	VerificationStatus string `json:"verificationStatus"`
}

func (c *AddCustomDomain) Name() string {
	return "render.service.addCustomDomain"
}

func (c *AddCustomDomain) Label() string {
	return "Add Custom Domain"
}

func (c *AddCustomDomain) Description() string {
	return "Add a custom domain to a Render service"
}

func (c *AddCustomDomain) Documentation() string {
	return `The Add Custom Domain component adds a custom domain to a Render service.

## Use Cases

- **Blue/green deployments**: Add the live domain to the new (green) service as part of a traffic switch
- **Domain management**: Automate custom domain provisioning as part of a deployment workflow

## How It Works

1. Adds the custom domain to the selected Render service
2. When **Wait For Verification** is enabled, triggers Render DNS verification and retrieves the latest custom domain status
3. Continues polling by re-triggering verification and checking ` + "`verificationStatus`" + ` until Render reports ` + "`verified`" + ` or ` + "`failed`" + `

## Configuration

- **Service**: Render service to add the domain to
- **Domain Name**: The custom domain name (e.g., ` + "`app.example.com`" + `)
- **Wait For Verification**: When enabled, waits for DNS verification before completing

## Output

Emits a ` + "`render.customDomain.added`" + ` payload with ` + "`id`" + `, ` + "`name`" + `, ` + "`serviceId`" + `, and ` + "`verificationStatus`" + `.`
}

func (c *AddCustomDomain) Icon() string {
	return "globe"
}

func (c *AddCustomDomain) Color() string {
	return "gray"
}

func (c *AddCustomDomain) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddCustomDomain) Configuration() []configuration.Field {
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
			Description: "Render service to add the domain to",
		},
		{
			Name:        "domain",
			Label:       "Domain Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., app.example.com",
			Description: "The custom domain name to add",
		},
		{
			Name:        "waitForVerification",
			Label:       "Wait For Verification",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Wait for DNS verification before completing",
		},
	}
}

func decodeAddCustomDomainConfiguration(cfg any) (AddCustomDomainConfiguration, error) {
	spec := AddCustomDomainConfiguration{}
	if err := mapstructure.Decode(cfg, &spec); err != nil {
		return AddCustomDomainConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Service = strings.TrimSpace(spec.Service)
	spec.Domain = strings.TrimSpace(spec.Domain)

	if spec.Service == "" {
		return AddCustomDomainConfiguration{}, fmt.Errorf("service is required")
	}
	if spec.Domain == "" {
		return AddCustomDomainConfiguration{}, fmt.Errorf("domain is required")
	}

	return spec, nil
}

func (c *AddCustomDomain) Setup(ctx core.SetupContext) error {
	spec, err := decodeAddCustomDomainConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	return setServiceNodeMetadata(ctx, spec.Service)
}

func (c *AddCustomDomain) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AddCustomDomain) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeAddCustomDomainConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	domain, err := client.AddCustomDomain(spec.Service, spec.Domain)
	if err != nil {
		return err
	}

	domainID := strings.TrimSpace(domain.ID)
	if domainID == "" {
		return fmt.Errorf("custom domain response missing id")
	}

	if err := ctx.Metadata.Set(AddCustomDomainExecutionMetadata{
		CustomDomain: &CustomDomainMetadata{
			ID:                 domainID,
			Name:               domain.Name,
			ServiceID:          spec.Service,
			VerificationStatus: domain.VerificationStatus,
		},
	}); err != nil {
		return err
	}

	if err := ctx.ExecutionState.SetKV(addCustomDomainExecutionKey, domainID); err != nil {
		return err
	}

	if !spec.WaitForVerification {
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			AddCustomDomainPayloadType,
			[]any{customDomainPayload(spec.Service, domain)},
		)
	}

	if domain.VerificationStatus == customDomainVerificationStatusVerified {
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			AddCustomDomainPayloadType,
			[]any{customDomainPayload(spec.Service, domain)},
		)
	}

	if domain.VerificationStatus == customDomainVerificationStatusFailed {
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			AddCustomDomainPayloadType,
			[]any{customDomainPayload(spec.Service, domain)},
		)
	}

	latestDomain, err := verifyAndFetchCustomDomain(client, spec.Service, domainID)
	if err != nil {
		return err
	}

	metadata := metadataFromDomain(spec.Service, latestDomain)
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	emitted, err := emitCustomDomainVerificationResult(ctx.ExecutionState, spec.Service, latestDomain)
	if emitted || err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, AddCustomDomainPollInterval)
}

func (c *AddCustomDomain) Hooks() []core.Hook {
	return []core.Hook{
		{
			Name: "poll",
			Type: core.HookTypeInternal,
		},
	}
}

func (c *AddCustomDomain) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case "poll":
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (c *AddCustomDomain) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	spec, err := decodeAddCustomDomainConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	metadata := AddCustomDomainExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.CustomDomain == nil || metadata.CustomDomain.ID == "" {
		return fmt.Errorf("custom domain metadata missing id")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	domain, err := verifyAndFetchCustomDomain(client, spec.Service, metadata.CustomDomain.ID)
	if err != nil {
		return err
	}

	metadata.CustomDomain.VerificationStatus = domain.VerificationStatus
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	if domain.VerificationStatus == customDomainVerificationStatusVerified ||
		domain.VerificationStatus == customDomainVerificationStatusFailed {
		_, err := emitCustomDomainVerificationResult(ctx.ExecutionState, spec.Service, domain)
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, AddCustomDomainPollInterval)
}

func (c *AddCustomDomain) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *AddCustomDomain) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddCustomDomain) Cleanup(ctx core.SetupContext) error {
	return nil
}

func metadataFromDomain(serviceID string, domain CustomDomainResponse) AddCustomDomainExecutionMetadata {
	return AddCustomDomainExecutionMetadata{
		CustomDomain: &CustomDomainMetadata{
			ID:                 domain.ID,
			Name:               domain.Name,
			ServiceID:          serviceID,
			VerificationStatus: domain.VerificationStatus,
		},
	}
}

func emitCustomDomainVerificationResult(
	executionState core.ExecutionStateContext,
	serviceID string,
	domain CustomDomainResponse,
) (bool, error) {
	switch domain.VerificationStatus {
	case customDomainVerificationStatusVerified, customDomainVerificationStatusFailed:
		return true, executionState.Emit(
			core.DefaultOutputChannel.Name,
			AddCustomDomainPayloadType,
			[]any{customDomainPayload(serviceID, domain)},
		)
	default:
		return false, nil
	}
}

func verifyAndFetchCustomDomain(client *Client, serviceID string, domainNameOrID string) (CustomDomainResponse, error) {
	if _, err := client.VerifyCustomDomain(serviceID, domainNameOrID); err != nil {
		return CustomDomainResponse{}, err
	}

	domain, err := client.GetCustomDomain(serviceID, domainNameOrID)
	if err != nil {
		return CustomDomainResponse{}, err
	}

	return domain, nil
}
