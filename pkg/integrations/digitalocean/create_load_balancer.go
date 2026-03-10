package digitalocean

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateLoadBalancer struct{}

type CreateLoadBalancerSpec struct {
	Name           string `json:"name" mapstructure:"name"`
	Region         string `json:"region" mapstructure:"region"`
	Algorithm      string `json:"algorithm" mapstructure:"algorithm"`
	EntryProtocol  string `json:"entryProtocol" mapstructure:"entryProtocol"`
	EntryPort      int    `json:"entryPort" mapstructure:"entryPort"`
	TargetProtocol string `json:"targetProtocol" mapstructure:"targetProtocol"`
	TargetPort     int    `json:"targetPort" mapstructure:"targetPort"`
}

func (c *CreateLoadBalancer) Name() string {
	return "digitalocean.createLoadBalancer"
}

func (c *CreateLoadBalancer) Label() string {
	return "Create Load Balancer"
}

func (c *CreateLoadBalancer) Description() string {
	return "Create a DigitalOcean Load Balancer"
}

func (c *CreateLoadBalancer) Documentation() string {
	return `The Create Load Balancer component creates a new DigitalOcean Load Balancer.

## Use Cases

- **Traffic distribution**: Distribute traffic across multiple droplets
- **High availability**: Set up load balancing for production environments
- **Auto-scaling**: Create load balancers as part of scaling workflows

## Configuration

- **Name**: The name for the load balancer (required, supports expressions)
- **Region**: Region where the load balancer will be created (required)
- **Algorithm**: Load balancing algorithm (optional, defaults to round_robin)
- **Entry Protocol**: Protocol for incoming traffic (required)
- **Entry Port**: Port for incoming traffic (required)
- **Target Protocol**: Protocol for backend traffic (required)
- **Target Port**: Port for backend traffic (required)

## Output

Returns the created load balancer object including:
- **id**: Load balancer ID
- **name**: Load balancer name
- **ip**: Load balancer IP address
- **status**: Current status
- **region**: Region information`
}

func (c *CreateLoadBalancer) Icon() string {
	return "server"
}

func (c *CreateLoadBalancer) Color() string {
	return "gray"
}

func (c *CreateLoadBalancer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateLoadBalancer) ExampleOutput() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"id":     "4de7ac8b-495b-4884-9a69-1050c6793cd6",
			"name":   "my-load-balancer",
			"ip":     "",
			"status": "new",
			"region": map[string]any{"name": "New York 3", "slug": "nyc3"},
		},
	}
}

func (c *CreateLoadBalancer) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The name for the load balancer",
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a region",
			Description: "Region where the load balancer will be created",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "region",
				},
			},
		},
		{
			Name:        "algorithm",
			Label:       "Algorithm",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Description: "Load balancing algorithm (defaults to round_robin)",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Round Robin", Value: "round_robin"},
						{Label: "Least Connections", Value: "least_connections"},
					},
				},
			},
		},
		{
			Name:        "entryProtocol",
			Label:       "Entry Protocol",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Protocol for incoming traffic",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "HTTP", Value: "http"},
						{Label: "HTTPS", Value: "https"},
						{Label: "TCP", Value: "tcp"},
						{Label: "UDP", Value: "udp"},
					},
				},
			},
		},
		{
			Name:        "entryPort",
			Label:       "Entry Port",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Description: "Port for incoming traffic",
		},
		{
			Name:        "targetProtocol",
			Label:       "Target Protocol",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Protocol for backend traffic",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "HTTP", Value: "http"},
						{Label: "HTTPS", Value: "https"},
						{Label: "TCP", Value: "tcp"},
						{Label: "UDP", Value: "udp"},
					},
				},
			},
		},
		{
			Name:        "targetPort",
			Label:       "Target Port",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Description: "Port for backend traffic",
		},
	}
}

func (c *CreateLoadBalancer) Setup(ctx core.SetupContext) error {
	spec := CreateLoadBalancerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Region == "" {
		return fmt.Errorf("region is required")
	}

	if spec.EntryProtocol == "" {
		return fmt.Errorf("entryProtocol is required")
	}

	if spec.EntryPort == 0 {
		return fmt.Errorf("entryPort is required")
	}

	if spec.TargetProtocol == "" {
		return fmt.Errorf("targetProtocol is required")
	}

	if spec.TargetPort == 0 {
		return fmt.Errorf("targetPort is required")
	}

	return nil
}

func (c *CreateLoadBalancer) Execute(ctx core.ExecutionContext) error {
	spec := CreateLoadBalancerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	name := readStringFromAny(spec.Name)
	if name == "" {
		return fmt.Errorf("name is required")
	}

	algorithm := spec.Algorithm
	if algorithm == "" {
		algorithm = "round_robin"
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	lb, err := client.CreateLoadBalancer(CreateLoadBalancerRequest{
		Name:      name,
		Region:    spec.Region,
		Algorithm: algorithm,
		ForwardingRules: []ForwardingRule{
			{
				EntryProtocol:  spec.EntryProtocol,
				EntryPort:      spec.EntryPort,
				TargetProtocol: spec.TargetProtocol,
				TargetPort:     spec.TargetPort,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create load balancer: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.load_balancer.created",
		[]any{lb},
	)
}

func (c *CreateLoadBalancer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateLoadBalancer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
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

func (c *CreateLoadBalancer) Cleanup(ctx core.SetupContext) error {
	return nil
}
