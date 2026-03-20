package digitalocean

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteLoadBalancer struct{}

type DeleteLoadBalancerSpec struct {
	LoadBalancerID string `json:"loadBalancerID" mapstructure:"loadBalancerID"`
}

func (d *DeleteLoadBalancer) Name() string {
	return "digitalocean.deleteLoadBalancer"
}

func (d *DeleteLoadBalancer) Label() string {
	return "Delete Load Balancer"
}

func (d *DeleteLoadBalancer) Description() string {
	return "Delete a DigitalOcean Load Balancer"
}

func (d *DeleteLoadBalancer) Documentation() string {
	return `The Delete Load Balancer component permanently deletes a load balancer from your DigitalOcean account.

## Use Cases

- **Cleanup**: Remove load balancers after decommissioning a service
- **Cost optimization**: Automatically tear down unused load balancers
- **Environment management**: Delete load balancers as part of environment teardown workflows

## Configuration

- **Load Balancer**: The load balancer to delete (required, supports expressions)

## Output

Returns information about the deleted load balancer:
- **loadBalancerID**: The UUID of the load balancer that was deleted

## Important Notes

- This operation is **permanent** and cannot be undone
- Deleting a load balancer does not delete the targeted droplets
- If the load balancer does not exist (404), the component emits success (idempotent)`
}

func (d *DeleteLoadBalancer) Icon() string {
	return "trash-2"
}

func (d *DeleteLoadBalancer) Color() string {
	return "red"
}

func (d *DeleteLoadBalancer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteLoadBalancer) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "loadBalancerID",
			Label:       "Load Balancer",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The load balancer to delete",
			Placeholder: "Select load balancer",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "load_balancer",
					UseNameAsValue: false,
				},
			},
		},
	}
}

func (d *DeleteLoadBalancer) Setup(ctx core.SetupContext) error {
	spec := DeleteLoadBalancerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.LoadBalancerID == "" {
		return errors.New("loadBalancerID is required")
	}

	return resolveLBMetadata(ctx, spec.LoadBalancerID)
}

func (d *DeleteLoadBalancer) Execute(ctx core.ExecutionContext) error {
	spec := DeleteLoadBalancerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.LoadBalancerID == "" {
		return errors.New("loadBalancerID is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.DeleteLoadBalancer(spec.LoadBalancerID)
	if err != nil {
		if doErr, ok := err.(*DOAPIError); ok && doErr.StatusCode == http.StatusNotFound {
			return ctx.ExecutionState.Emit(
				core.DefaultOutputChannel.Name,
				"digitalocean.loadbalancer.deleted",
				[]any{map[string]any{"loadBalancerID": spec.LoadBalancerID}},
			)
		}
		return fmt.Errorf("failed to delete load balancer: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.loadbalancer.deleted",
		[]any{map[string]any{"loadBalancerID": spec.LoadBalancerID}},
	)
}

func (d *DeleteLoadBalancer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteLoadBalancer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteLoadBalancer) Actions() []core.Action {
	return []core.Action{}
}

func (d *DeleteLoadBalancer) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (d *DeleteLoadBalancer) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteLoadBalancer) Cleanup(ctx core.SetupContext) error {
	return nil
}
