package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateStaticIP struct{}

type CreateStaticIPSpec struct {
	Name        string `mapstructure:"name"`
	Region      string `mapstructure:"region"`
	NetworkTier string `mapstructure:"networkTier"`
	Description string `mapstructure:"description"`
}

const (
	NetworkTierPremium  = "PREMIUM"
	NetworkTierStandard = "STANDARD"
)

func (c *CreateStaticIP) Name() string {
	return "gcp.compute.createStaticIP"
}

func (c *CreateStaticIP) Label() string {
	return "Compute • Create Static IP"
}

func (c *CreateStaticIP) Description() string {
	return "Reserve a regional external static IP address in a Google Cloud project"
}

func (c *CreateStaticIP) Documentation() string {
	return `The Create Static IP component reserves a regional external static (reserved) IP address in Compute Engine.

A static IP keeps the same address across VM restarts and re-creations, unlike an ephemeral IP. Once reserved it can be attached to a VM instance with the **Manage Static IP** component.

## Use Cases

- **Stable endpoints**: Give a service a fixed public address that survives VM replacement
- **Blue/green deployments**: Reserve the address ahead of time, then attach it to whichever VM is live
- **DNS**: Point an A record at a reserved address you control

## Configuration

- **Name**: The name for the new address resource (required, lowercase RFC1035 — e.g. ` + "`web-prod-ip`" + `)
- **Region**: The region to reserve the address in (required). Regional external IPs can only be attached to VMs in the same region.
- **Network Tier**: ` + "`PREMIUM`" + ` (default) or ` + "`STANDARD`" + `
- **Description**: Optional human-readable description

## Output

Returns the reserved address:
- **name**, **address** (the reserved IP), **region**, **status**, **addressType**, **networkTier**, **selfLink**

## Important Notes

- Reserving and holding a static IP that is not attached to a running resource incurs charges
- The component waits for the underlying regional operation to complete before reading the address back`
}

func (c *CreateStaticIP) Icon() string {
	return "globe"
}

func (c *CreateStaticIP) Color() string {
	return "blue"
}

func (c *CreateStaticIP) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateStaticIP) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name for the reserved address (lowercase letters, numbers and hyphens).",
			Placeholder: "e.g. web-prod-ip",
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The region to reserve the static IP in (e.g. us-central1).",
			Placeholder: "Select region",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeRegion,
				},
			},
		},
		{
			Name:        "networkTier",
			Label:       "Network Tier",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Network service tier for the address.",
			Default:     NetworkTierPremium,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Premium", Value: NetworkTierPremium},
						{Label: "Standard", Value: NetworkTierStandard},
					},
				},
			},
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional description for the address resource.",
			Placeholder: "e.g. Production web server IP",
		},
	}
}

func (c *CreateStaticIP) Setup(ctx core.SetupContext) error {
	spec := CreateStaticIPSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.Name) == "" {
		return errors.New("name is required")
	}

	if strings.TrimSpace(spec.Region) == "" {
		return errors.New("region is required")
	}

	if spec.NetworkTier != "" && spec.NetworkTier != NetworkTierPremium && spec.NetworkTier != NetworkTierStandard {
		return fmt.Errorf("invalid networkTier %q: must be %s or %s", spec.NetworkTier, NetworkTierPremium, NetworkTierStandard)
	}

	return nil
}

func (c *CreateStaticIP) Execute(ctx core.ExecutionContext) error {
	spec := CreateStaticIPSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return ctx.ExecutionState.Fail("error", "name is required")
	}

	region := lastSegment(strings.TrimSpace(spec.Region))
	if region == "" {
		return ctx.ExecutionState.Fail("error", "region is required")
	}

	client, err := getClient(ctx)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	project := client.ProjectID()
	callCtx := context.Background()

	body := map[string]any{
		"name":        name,
		"addressType": AddressTypeExternal,
	}
	if spec.NetworkTier != "" {
		body["networkTier"] = spec.NetworkTier
	}
	if strings.TrimSpace(spec.Description) != "" {
		body["description"] = strings.TrimSpace(spec.Description)
	}

	path := fmt.Sprintf("projects/%s/regions/%s/addresses", project, region)
	respBody, err := client.Post(callCtx, path, body)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to reserve static IP: %v", err))
	}

	var opResp struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(respBody, &opResp); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("parse create operation response: %v", err))
	}
	if opResp.Name == "" {
		return ctx.ExecutionState.Fail("error", "create operation response missing operation name; cannot confirm reservation")
	}

	if err := WaitForRegionOperation(callCtx, client, project, region, lastSegment(opResp.Name)); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("error waiting for create operation: %v", err))
	}

	addrBody, err := GetAddress(callCtx, client, project, region, name)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read reserved address: %v", err))
	}

	var addr addressGetResp
	if err := json.Unmarshal(addrBody, &addr); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("parse address response: %v", err))
	}

	payload := map[string]any{
		"name":        addr.Name,
		"address":     addr.Address,
		"region":      region,
		"status":      addr.Status,
		"addressType": addr.AddressType,
		"networkTier": addr.NetworkTier,
		"selfLink":    addr.SelfLink,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.compute.staticIP.created",
		[]any{payload},
	)
}

func (c *CreateStaticIP) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateStaticIP) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateStaticIP) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateStaticIP) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateStaticIP) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateStaticIP) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
