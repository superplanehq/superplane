package hetzner

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const DeleteLoadBalancerPayloadType = "hetzner.load_balancer.deleted"

type DeleteLoadBalancer struct{}

type DeleteLoadBalancerSpec struct {
	LoadBalancer string `json:"loadBalancer" mapstructure:"loadBalancer"`
}

type DeleteLoadBalancerExecutionMetadata struct {
	LoadBalancerID string `json:"loadBalancerId" mapstructure:"loadBalancerId"`
}

func (c *DeleteLoadBalancer) Name() string {
	return "hetzner.deleteLoadBalancer"
}

func (c *DeleteLoadBalancer) Label() string {
	return "Delete Load Balancer"
}

func (c *DeleteLoadBalancer) Description() string {
	return "Delete a Hetzner Cloud Load Balancer"
}

func (c *DeleteLoadBalancer) Documentation() string {
	return `The Delete Load Balancer component deletes a load balancer in Hetzner Cloud.

## How It Works

1. Deletes the selected load balancer via the Hetzner API
2. Emits on the default output when the load balancer is deleted. If deletion fails, the execution errors.
`
}

func (c *DeleteLoadBalancer) Icon() string {
	return "hetzner"
}

func (c *DeleteLoadBalancer) Color() string {
	return "gray"
}

func (c *DeleteLoadBalancer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteLoadBalancer) ExampleOutput() map[string]any {
	return map[string]any{
		"loadBalancerId": "12345",
	}
}

func (c *DeleteLoadBalancer) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "loadBalancer",
			Label:    "Load Balancer",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "load_balancer",
				},
			},
			Description: "Load Balancer to delete",
		},
	}
}

func (c *DeleteLoadBalancer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteLoadBalancer) Setup(ctx core.SetupContext) error {
	spec := DeleteLoadBalancerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(spec.LoadBalancer) == "" {
		return fmt.Errorf("loadBalancer is required")
	}
	return nil
}

func (c *DeleteLoadBalancer) Execute(ctx core.ExecutionContext) error {
	loadBalancerID, err := resolveLoadBalancerID(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := ctx.Metadata.Set(DeleteLoadBalancerExecutionMetadata{LoadBalancerID: loadBalancerID}); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.DeleteLoadBalancer(loadBalancerID); err != nil {
		return fmt.Errorf("delete load balancer: %w", err)
	}

	payload := map[string]any{
		"loadBalancerId": loadBalancerID,
	}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteLoadBalancerPayloadType, []any{payload})
}

func (c *DeleteLoadBalancer) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteLoadBalancer) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteLoadBalancer) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *DeleteLoadBalancer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteLoadBalancer) Cleanup(ctx core.SetupContext) error {
	return nil
}
