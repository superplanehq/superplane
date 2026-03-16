package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const lbPollInterval = 15 * time.Second

type CreateLoadBalancer struct{}

var lbNameRegexp = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

type ForwardingRuleSpec struct {
	EntryProtocol  string `json:"entryProtocol" mapstructure:"entryProtocol"`
	EntryPort      int    `json:"entryPort" mapstructure:"entryPort"`
	TargetProtocol string `json:"targetProtocol" mapstructure:"targetProtocol"`
	TargetPort     int    `json:"targetPort" mapstructure:"targetPort"`
	TLSPassthrough bool   `json:"tlsPassthrough" mapstructure:"tlsPassthrough"`
}

type CreateLoadBalancerSpec struct {
	Name            string               `json:"name" mapstructure:"name"`
	Region          string               `json:"region" mapstructure:"region"`
	ForwardingRules []ForwardingRuleSpec `json:"forwardingRules" mapstructure:"forwardingRules"`
	Droplets        []string             `json:"droplets" mapstructure:"droplets"`
	Tag             string               `json:"tag" mapstructure:"tag"`
}

func (c *CreateLoadBalancer) Name() string {
	return "digitalocean.createLoadBalancer"
}

func (c *CreateLoadBalancer) Label() string {
	return "Create Load Balancer"
}

func (c *CreateLoadBalancer) Description() string {
	return "Create a DigitalOcean Load Balancer with forwarding rules and targets"
}

func (c *CreateLoadBalancer) Documentation() string {
	return `The Create Load Balancer component creates a new load balancer in DigitalOcean and waits until it is active.

## Use Cases

- **Traffic distribution**: Distribute incoming requests across multiple droplets
- **High availability**: Ensure zero-downtime deployments by routing traffic across instances
- **Scalable infrastructure**: Provision load balancers as part of automated environment setup

## Configuration

- **Name**: The name of the load balancer (required, only letters, numbers, and hyphens)
- **Region**: Region where the load balancer will be created (required)
- **Forwarding Rules**: One or more forwarding rules specifying entry/target protocol, port, and optional TLS passthrough (required)
- **Droplets**: The droplets to add as targets — must be in the same region as the load balancer (optional, mutually exclusive with Tag)
- **Tag**: Tag used to dynamically target droplets (optional, mutually exclusive with Droplets)

## Output

Returns the created load balancer object including:
- **id**: Load balancer ID (UUID)
- **name**: Load balancer name
- **ip**: Assigned public IP address
- **status**: Current status (active)
- **region**: Region information
- **forwarding_rules**: Configured forwarding rules
- **droplet_ids**: Targeted droplet IDs

## Important Notes

- The component polls until the load balancer status becomes **active**
- Specify either **Droplet IDs** or **Tag** to define targets, not both
- The load balancer name must contain only letters, numbers, and hyphens
- All specified droplets must be in the same region as the load balancer`
}

func (c *CreateLoadBalancer) Icon() string {
	return "network"
}

func (c *CreateLoadBalancer) Color() string {
	return "blue"
}

func (c *CreateLoadBalancer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateLoadBalancer) Configuration() []configuration.Field {
	protocols := []configuration.FieldOption{
		{Label: "HTTP", Value: "http"},
		{Label: "HTTPS", Value: "https"},
		{Label: "TCP", Value: "tcp"},
		{Label: "UDP", Value: "udp"},
	}

	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The name of the load balancer",
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The region where the load balancer will be created",
			Placeholder: "Select a region",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "region",
				},
			},
		},
		{
			Name:        "forwardingRules",
			Label:       "Forwarding Rules",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "Rules that define how traffic is forwarded to droplets",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Rule",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "entryProtocol",
								Label:    "Entry Protocol",
								Type:     configuration.FieldTypeSelect,
								Required: true,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: protocols,
									},
								},
							},
							{
								Name:     "entryPort",
								Label:    "Entry Port",
								Type:     configuration.FieldTypeNumber,
								Required: true,
								TypeOptions: &configuration.TypeOptions{
									Number: &configuration.NumberTypeOptions{
										Min: intPtr(1),
										Max: intPtr(65535),
									},
								},
							},
							{
								Name:     "targetProtocol",
								Label:    "Target Protocol",
								Type:     configuration.FieldTypeSelect,
								Required: true,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: protocols,
									},
								},
							},
							{
								Name:     "targetPort",
								Label:    "Target Port",
								Type:     configuration.FieldTypeNumber,
								Required: true,
								TypeOptions: &configuration.TypeOptions{
									Number: &configuration.NumberTypeOptions{
										Min: intPtr(1),
										Max: intPtr(65535),
									},
								},
							},
							{
								Name:        "tlsPassthrough",
								Label:       "TLS Passthrough",
								Type:        configuration.FieldTypeBool,
								Required:    false,
								Description: "Pass TLS traffic through to the target without terminating the connection",
							},
						},
					},
				},
			},
		},
		{
			Name:        "droplets",
			Label:       "Droplets",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Droplets to add as load balancer targets. Must be in the same region as the load balancer (mutually exclusive with Tag)",
			Placeholder: "Select droplets",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "droplet",
					Multi: true,
				},
			},
		},
		{
			Name:        "tag",
			Label:       "Tag",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Tag used to dynamically target droplets (mutually exclusive with Droplet IDs)",
		},
	}
}

func intPtr(i int) *int {
	return &i
}

func (c *CreateLoadBalancer) Setup(ctx core.SetupContext) error {
	spec := CreateLoadBalancerSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}

	if !lbNameRegexp.MatchString(spec.Name) {
		return errors.New("name must contain only letters, numbers, and hyphens")
	}

	if spec.Region == "" {
		return errors.New("region is required")
	}

	if len(spec.ForwardingRules) == 0 {
		return errors.New("at least one forwarding rule is required")
	}

	for i, rule := range spec.ForwardingRules {
		if rule.EntryProtocol == "" {
			return fmt.Errorf("forwarding rule %d: entryProtocol is required", i+1)
		}
		if rule.EntryPort == 0 {
			return fmt.Errorf("forwarding rule %d: entryPort is required", i+1)
		}
		if rule.TargetProtocol == "" {
			return fmt.Errorf("forwarding rule %d: targetProtocol is required", i+1)
		}
		if rule.TargetPort == 0 {
			return fmt.Errorf("forwarding rule %d: targetPort is required", i+1)
		}
	}

	return nil
}

func (c *CreateLoadBalancer) Execute(ctx core.ExecutionContext) error {
	spec := CreateLoadBalancerSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	rules := make([]ForwardingRule, 0, len(spec.ForwardingRules))
	for _, r := range spec.ForwardingRules {
		rules = append(rules, ForwardingRule{
			EntryProtocol:  r.EntryProtocol,
			EntryPort:      r.EntryPort,
			TargetProtocol: r.TargetProtocol,
			TargetPort:     r.TargetPort,
			TLSPassthrough: r.TLSPassthrough,
		})
	}

	req := CreateLoadBalancerRequest{
		Name:            spec.Name,
		Region:          spec.Region,
		ForwardingRules: rules,
		Tag:             spec.Tag,
	}

	if len(spec.Droplets) > 0 {
		dropletIDs := make([]int, 0, len(spec.Droplets))
		for _, idStr := range spec.Droplets {
			var id int
			if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
				return fmt.Errorf("invalid droplet ID %q: must be a number", idStr)
			}
			dropletIDs = append(dropletIDs, id)
		}
		req.DropletIDs = dropletIDs
	}

	lb, err := client.CreateLoadBalancer(req)
	if err != nil {
		return fmt.Errorf("failed to create load balancer: %v", err)
	}

	if err := ctx.Metadata.Set(map[string]any{"lbID": lb.ID}); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, lbPollInterval)
}

func (c *CreateLoadBalancer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateLoadBalancer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateLoadBalancer) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (c *CreateLoadBalancer) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata struct {
		LBID string `mapstructure:"lbID"`
	}

	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	lb, err := client.GetLoadBalancer(metadata.LBID)
	if err != nil {
		return fmt.Errorf("failed to get load balancer: %v", err)
	}

	switch lb.Status {
	case "active":
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"digitalocean.loadbalancer.created",
			[]any{lb},
		)
	case "errored":
		return fmt.Errorf("load balancer reached error status")
	default:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, lbPollInterval)
	}
}

func (c *CreateLoadBalancer) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateLoadBalancer) Cleanup(ctx core.SetupContext) error {
	return nil
}
