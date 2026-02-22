package hetzner

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const CreateLoadBalancerPayloadType = "hetzner.load_balancer.created"

type CreateLoadBalancer struct{}

type CreateLoadBalancerSpec struct {
	Name             string `json:"name" mapstructure:"name"`
	LoadBalancerType string `json:"load_balancer_type" mapstructure:"loadBalancerType"`
	Location         string `json:"location" mapstructure:"location"`
	Algorithm        string `json:"algorithm" mapstructure:"algorithm"`
}

type CreateLoadBalancerExecutionMetadata struct {
	ActionID     string          `json:"actionId" mapstructure:"actionId"`
	LoadBalancer *ServerResponse `json:"loadBalancer,omitempty" mapstructure:"loadBalancer"`
}

func (c *CreateLoadBalancer) Name() string {
	return "hetzner.createLoadBalancer"
}

func (c *CreateLoadBalancer) Label() string {
	return "Create Load Balancer"
}

func (c *CreateLoadBalancer) Description() string {
	return "Create a Hetzner Cloud Load Balancer."
}

func (c *CreateLoadBalancer) Documentation() string {
	return `The Create Load Balancer component creates a load balancer in Hetzner Cloud.

## How It Works

1. Creates a load balancer with the specified name, type, location, and algorithm via the Hetzner API
2. Emits the created load balancer details on the default output channel

## Configuration

- **Name**: The name for the new load balancer (supports expressions)
- **Type**: The load balancer type (e.g. lb11, lb21, lb31)
- **Location**: The location where the load balancer will be created
- **Algorithm**: The load balancing algorithm — Round Robin (default) or Least Connections
`
}

func (c *CreateLoadBalancer) Icon() string {
	return "hetzner"
}

func (c *CreateLoadBalancer) Color() string {
	return "gray"
}

func (c *CreateLoadBalancer) ExampleOutput() map[string]any {
	return map[string]any{
		"id":     "12345",
		"name":   "my-load-balancer",
		"status": "running",
	}
}

func (c *CreateLoadBalancer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateLoadBalancer) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "Name",
		},
		{
			Name:        "loadBalancerType",
			Label:       "Type",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select type",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "load_balancer_type",
				},
			},
			Description: "Type",
		},
		{
			Name:        "location",
			Label:       "Location",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select location",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "location",
				},
			},
			Description: "Location",
		},
		{
			Name:        "algorithm",
			Label:       "Load Balancing algorithm",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Placeholder: "Select load balancing algorithm",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "load_balancing_algorithm",
				},
			},
			Description: "Algorithm (optional, omit for Round Robin).",
		},
	}
}

func (c *CreateLoadBalancer) Setup(ctx core.SetupContext) error {
	spec := CreateLoadBalancerSpec{}

	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(spec.LoadBalancerType) == "" {
		return fmt.Errorf("loadBalancerType is required")
	}

	return nil
}

func (c *CreateLoadBalancer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateLoadBalancer) Execute(ctx core.ExecutionContext) error {
	spec := CreateLoadBalancerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	name := strings.TrimSpace(readStringFromAny(spec.Name))
	if name == "" {
		return fmt.Errorf("name is required")
	}
	loadBalancerType := strings.TrimSpace(readStringFromAny(spec.LoadBalancerType))
	if loadBalancerType == "" {
		return fmt.Errorf("load_balancer_type is required")
	}

	location := strings.TrimSpace(readStringFromAny(spec.Location))

	algorithm := strings.TrimSpace(readStringFromAny(spec.Algorithm))
	if algorithm == "" {
		algorithm = LoadBalancerAlgorithmTypeRoundRobin
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	loadBalancer, action, err := client.CreateLoadBalancer(name, loadBalancerType, location, algorithm)
	if err != nil {
		return fmt.Errorf("create load balancer: %w", err)
	}

	metadata := CreateLoadBalancerExecutionMetadata{
		ActionID:     action.ID,
		LoadBalancer: loadBalancer,
	}
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateLoadBalancerPayloadType, []any{loadBalancer})
}

func (c *CreateLoadBalancer) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateLoadBalancer) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateLoadBalancer) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *CreateLoadBalancer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateLoadBalancer) Cleanup(ctx core.SetupContext) error {
	return nil
}
