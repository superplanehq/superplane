package digitalocean

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteLoadBalancer struct{}

type DeleteLoadBalancerSpec struct {
	LoadBalancerID string `json:"loadBalancerId" mapstructure:"loadBalancerId"`
}

func (c *DeleteLoadBalancer) Name() string {
	return "digitalocean.deleteLoadBalancer"
}

func (c *DeleteLoadBalancer) Label() string {
	return "Delete Load Balancer"
}

func (c *DeleteLoadBalancer) Description() string {
	return "Delete a DigitalOcean Load Balancer"
}

func (c *DeleteLoadBalancer) Documentation() string {
	return `The Delete Load Balancer component deletes a DigitalOcean Load Balancer.

## How It Works

1. Deletes the specified load balancer via the DigitalOcean API
2. Emits on the default output when the load balancer is deleted. If deletion fails, the execution errors.

## Configuration

- **Load Balancer ID**: The ID of the load balancer to delete (required, supports expressions)

## Output

Returns confirmation of the deleted load balancer:
- **loadBalancerId**: The ID of the deleted load balancer`
}

func (c *DeleteLoadBalancer) Icon() string {
	return "server"
}

func (c *DeleteLoadBalancer) Color() string {
	return "gray"
}

func (c *DeleteLoadBalancer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteLoadBalancer) ExampleOutput() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"loadBalancerId": "4de7ac8b-495b-4884-9a69-1050c6793cd6",
		},
	}
}

func (c *DeleteLoadBalancer) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "loadBalancerId",
			Label:       "Load Balancer ID",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The ID of the load balancer to delete",
		},
	}
}

func (c *DeleteLoadBalancer) Setup(ctx core.SetupContext) error {
	spec := DeleteLoadBalancerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.LoadBalancerID == "" {
		return fmt.Errorf("loadBalancerId is required")
	}

	return nil
}

func (c *DeleteLoadBalancer) Execute(ctx core.ExecutionContext) error {
	loadBalancerID, err := resolveStringField(ctx.Configuration, "loadBalancerId")
	if err != nil {
		return err
	}

	if err := ctx.Metadata.Set(map[string]any{"loadBalancerId": loadBalancerID}); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.DeleteLoadBalancer(loadBalancerID); err != nil {
		return fmt.Errorf("failed to delete load balancer: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.load_balancer.deleted",
		[]any{map[string]any{"loadBalancerId": loadBalancerID}},
	)
}

func (c *DeleteLoadBalancer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteLoadBalancer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
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

func (c *DeleteLoadBalancer) Cleanup(ctx core.SetupContext) error {
	return nil
}
