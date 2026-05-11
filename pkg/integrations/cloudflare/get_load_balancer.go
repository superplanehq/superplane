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

type GetLoadBalancer struct{}

type GetLoadBalancerSpec struct {
	LoadBalancer string `json:"loadBalancer"` // composite "zoneId/lbId"
}

type LoadBalancerNodeMetadata struct {
	LoadBalancerName string `json:"loadBalancerName"`
}

func (c *GetLoadBalancer) Name() string {
	return "cloudflare.getLoadBalancer"
}

func (c *GetLoadBalancer) Label() string {
	return "Get Load Balancer"
}

func (c *GetLoadBalancer) Description() string {
	return "Retrieve a Cloudflare Load Balancer by ID"
}

func (c *GetLoadBalancer) Documentation() string {
	return `The Get Load Balancer component fetches the current state of a Cloudflare Load Balancer.

## Use Cases

- **Pre-flight validation**: Confirm a load balancer exists before updating it
- **Audit**: Capture a snapshot of the load balancer configuration at a point in time
- **Conditional logic**: Branch a workflow based on the current steering policy or pool set

## Configuration

- **Load Balancer**: The load balancer to retrieve

## Output

Returns the full load balancer configuration including pools, steering policy, and session affinity settings.`
}

func (c *GetLoadBalancer) Icon() string {
	return "cloud"
}

func (c *GetLoadBalancer) Color() string {
	return "orange"
}

func (c *GetLoadBalancer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetLoadBalancer) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "loadBalancer",
			Label:       "Load Balancer",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The load balancer to retrieve",
			Placeholder: "Select a load balancer",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "load_balancer",
				},
			},
		},
	}
}

func (c *GetLoadBalancer) Setup(ctx core.SetupContext) error {
	spec := GetLoadBalancerSpec{}
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

func (c *GetLoadBalancer) resolveMetadata(ctx core.SetupContext, zoneID, lbID string) error {
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

func (c *GetLoadBalancer) Execute(ctx core.ExecutionContext) error {
	spec := GetLoadBalancerSpec{}
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

	lb, err := client.GetLoadBalancer(zoneID, lbID)
	if err != nil {
		return fmt.Errorf("failed to get load balancer: %v", err)
	}

	result := map[string]any{
		"loadBalancer":   lb,
		"zoneId":         zoneID,
		"loadBalancerId": lbID,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.loadBalancer.fetched",
		[]any{result},
	)
}

func (c *GetLoadBalancer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetLoadBalancer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetLoadBalancer) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetLoadBalancer) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetLoadBalancer) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetLoadBalancer) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
