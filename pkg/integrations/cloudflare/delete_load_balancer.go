package cloudflare

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteLoadBalancer struct{}

type DeleteLoadBalancerSpec struct {
	LoadBalancer string `json:"loadBalancer"`
}

func (c *DeleteLoadBalancer) Name() string {
	return "cloudflare.deleteLoadBalancer"
}

func (c *DeleteLoadBalancer) Label() string {
	return "Delete Load Balancer"
}

func (c *DeleteLoadBalancer) Description() string {
	return "Delete a Cloudflare Load Balancer"
}

func (c *DeleteLoadBalancer) Documentation() string {
	return `The Delete Load Balancer component permanently removes a Cloudflare Load Balancer from a zone.

## Use Cases

- **Teardown**: Remove a load balancer as part of environment cleanup
- **Blue/green decommission**: Delete the old load balancer after traffic has been fully migrated

## Configuration

- **Load Balancer**: The load balancer to delete

## Output

Emits a confirmation with the zone ID and load balancer ID of the deleted load balancer.

> **Warning**: This operation is irreversible. Deleting the load balancer will immediately stop routing traffic through it.`
}

func (c *DeleteLoadBalancer) Icon() string {
	return "cloud"
}

func (c *DeleteLoadBalancer) Color() string {
	return "orange"
}

func (c *DeleteLoadBalancer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteLoadBalancer) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "loadBalancer",
			Label:       "Load Balancer",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The load balancer to delete",
			Placeholder: "Select a load balancer",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "load_balancer",
				},
			},
		},
	}
}

func (c *DeleteLoadBalancer) Setup(ctx core.SetupContext) error {
	spec := DeleteLoadBalancerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.LoadBalancer == "" {
		return errors.New("loadBalancer is required")
	}

	if strings.Contains(spec.LoadBalancer, "{{") {
		return ctx.Metadata.Set(LoadBalancerNodeMetadata{LoadBalancerName: spec.LoadBalancer})
	}

	zoneID, lbID, err := splitLBID(spec.LoadBalancer)
	if err != nil {
		return err
	}

	return c.resolveMetadata(ctx, zoneID, lbID)
}

func (c *DeleteLoadBalancer) resolveMetadata(ctx core.SetupContext, zoneID, lbID string) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	lb, err := client.GetLoadBalancer(zoneID, lbID)
	if err != nil {
		return fmt.Errorf("failed to get load balancer: %w", err)
	}
	return ctx.Metadata.Set(LoadBalancerNodeMetadata{LoadBalancerName: lb.Name})
}

func (c *DeleteLoadBalancer) Execute(ctx core.ExecutionContext) error {
	spec := DeleteLoadBalancerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	zoneID, lbID, err := splitLBID(spec.LoadBalancer)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.DeleteLoadBalancer(zoneID, lbID); err != nil {
		return fmt.Errorf("failed to delete load balancer: %v", err)
	}

	result := map[string]any{
		"zoneId":         zoneID,
		"loadBalancerId": lbID,
		"deleted":        true,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.loadBalancer.deleted",
		[]any{result},
	)
}

func (c *DeleteLoadBalancer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteLoadBalancer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteLoadBalancer) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteLoadBalancer) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteLoadBalancer) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteLoadBalancer) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
