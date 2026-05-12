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

type UpdateLoadBalancer struct{}

type UpdateLoadBalancerSpec struct {
	LoadBalancer       string           `json:"loadBalancer"`
	Name               string           `json:"name"`
	Description        string           `json:"description"`
	SteeringPolicy     string           `json:"steeringPolicy"`
	SessionAffinity    string           `json:"sessionAffinity"`
	SessionAffinityTTL *int             `json:"sessionAffinityTtl"`
	PoolWeights        []PoolWeightSpec `json:"poolWeights"`
	FallbackPool       string           `json:"fallbackPool"`
	DefaultPools       []string         `json:"defaultPools"`
	Enabled            *bool            `json:"enabled"`
}

func (c *UpdateLoadBalancer) Name() string {
	return "cloudflare.updateLoadBalancer"
}

func (c *UpdateLoadBalancer) Label() string {
	return "Update Load Balancer"
}

func (c *UpdateLoadBalancer) Description() string {
	return "Update a Cloudflare Load Balancer's steering policy, pool weights, or session affinity"
}

func (c *UpdateLoadBalancer) Documentation() string {
	return `The Update Load Balancer component modifies an existing Cloudflare Load Balancer.

## Use Cases

- **Traffic shifting**: Change pool weights to shift traffic between canary and stable pools
- **Blue/green cutover**: Switch default pools to point at the green pool
- **Session affinity changes**: Enable or disable sticky sessions for a release
- **Steering policy update**: Switch from random to geo or latency-based steering

## Configuration

- **Load Balancer**: The load balancer to update
- **Steering Policy**: (Optional) New distribution policy across pools
- **Pool Weights**: (Optional) Per-pool weights for random steering
- **Session Affinity**: (Optional) How client sessions are pinned to a specific pool
- **Session Affinity TTL**: (Optional) Duration of session affinity in seconds
- **Fallback Pool**: (Optional) New fallback pool
- **Default Pools**: (Optional) New ordered list of active pools

## Output

Returns the updated load balancer configuration.`
}

func (c *UpdateLoadBalancer) Icon() string {
	return "cloud"
}

func (c *UpdateLoadBalancer) Color() string {
	return "orange"
}

func (c *UpdateLoadBalancer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateLoadBalancer) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "loadBalancer",
			Label:       "Load Balancer",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The load balancer to update",
			Placeholder: "Select a load balancer",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "load_balancer",
				},
			},
		},
		{
			Name:        "name",
			Label:       "Hostname",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "New hostname for the load balancer (e.g. lb.example.com)",
			Placeholder: "lb.example.com",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "New description for the load balancer",
		},
		{
			Name:        "steeringPolicy",
			Label:       "Steering Policy",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Description: "New policy for distributing traffic across pools",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Off (use pool order)", Value: "off"},
						{Label: "Random", Value: "random"},
						{Label: "Geo", Value: "geo"},
						{Label: "Dynamic Latency", Value: "dynamic_latency"},
						{Label: "Proximity", Value: "proximity"},
						{Label: "Least Outstanding Requests", Value: "least_outstanding_requests"},
						{Label: "Least Connections", Value: "least_connections"},
					},
				},
			},
		},
		{
			Name:        "poolWeights",
			Label:       "Pool Weights",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Per-pool weights used when steering policy is Random. Replaces all existing weights.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Pool Weight",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "pool",
								Label:       "Pool",
								Type:        configuration.FieldTypeIntegrationResource,
								Required:    true,
								Description: "The pool to assign a weight to",
								Placeholder: "Select a pool",
								TypeOptions: &configuration.TypeOptions{
									Resource: &configuration.ResourceTypeOptions{
										Type: "pool",
									},
								},
							},
							{
								Name:        "weight",
								Label:       "Weight",
								Type:        configuration.FieldTypeNumber,
								Required:    true,
								Default:     1,
								Description: "Traffic weight for this pool (0.0–1.0)",
							},
						},
					},
				},
			},
		},
		{
			Name:        "sessionAffinity",
			Label:       "Session Affinity",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Description: "New session affinity mode",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "None", Value: "none"},
						{Label: "Cookie", Value: "cookie"},
						{Label: "IP Cookie", Value: "ip_cookie"},
					},
				},
			},
		},
		{
			Name:        "sessionAffinityTtl",
			Label:       "Session Affinity TTL (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Time-to-live for session affinity cookies in seconds",
		},
		{
			Name:        "fallbackPool",
			Label:       "Fallback Pool",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "New fallback pool to use when all default pools are unhealthy",
			Placeholder: "Select a pool",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "pool",
				},
			},
		},
		{
			Name:        "defaultPools",
			Label:       "Default Pools",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "New ordered list of pool IDs to route traffic to. When provided, replaces the current list.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Pool ID",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "enabled",
			Label:       "Enabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Enable or disable the load balancer",
		},
	}
}

func (c *UpdateLoadBalancer) Setup(ctx core.SetupContext) error {
	spec := UpdateLoadBalancerSpec{}
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

func (c *UpdateLoadBalancer) resolveMetadata(ctx core.SetupContext, zoneID, lbID string) error {
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

func (c *UpdateLoadBalancer) Execute(ctx core.ExecutionContext) error {
	spec := UpdateLoadBalancerSpec{}
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

	var randomSteering *RandomSteering
	if len(spec.PoolWeights) > 0 {
		weights := make(map[string]float64, len(spec.PoolWeights))
		for _, pw := range spec.PoolWeights {
			weights[pw.Pool] = pw.Weight
		}
		randomSteering = &RandomSteering{PoolWeights: weights}
	}

	req := UpdateLoadBalancerRequest{
		Name:               spec.Name,
		Description:        spec.Description,
		Enabled:            spec.Enabled,
		SteeringPolicy:     spec.SteeringPolicy,
		SessionAffinity:    spec.SessionAffinity,
		SessionAffinityTTL: spec.SessionAffinityTTL,
		RandomSteering:     randomSteering,
		FallbackPool:       spec.FallbackPool,
		DefaultPools:       spec.DefaultPools,
	}

	lb, err := client.UpdateLoadBalancer(zoneID, lbID, req)
	if err != nil {
		return fmt.Errorf("failed to update load balancer: %v", err)
	}

	result := map[string]any{
		"loadBalancer":   lb,
		"zoneId":         zoneID,
		"loadBalancerId": lbID,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.loadBalancer.updated",
		[]any{result},
	)
}

func (c *UpdateLoadBalancer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateLoadBalancer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateLoadBalancer) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateLoadBalancer) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateLoadBalancer) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateLoadBalancer) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
