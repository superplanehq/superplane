package cloudflare

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateLoadBalancer struct{}

type PoolWeightSpec struct {
	Pool   string  `json:"pool"`
	Weight float64 `json:"weight"`
}

type LBRuleOverridesSpec struct {
	SteeringPolicy     string           `json:"steeringPolicy"`
	FallbackPool       string           `json:"fallbackPool"`
	DefaultPools       []string         `json:"defaultPools"`
	SessionAffinity    string           `json:"sessionAffinity"`
	SessionAffinityTTL *int             `json:"sessionAffinityTtl"`
	PoolWeights        []PoolWeightSpec `json:"poolWeights"`
}

type LBRuleSpec struct {
	Name      string              `json:"name"`
	Condition string              `json:"condition"`
	Disabled  bool                `json:"disabled"`
	Priority  int                 `json:"priority"`
	Overrides LBRuleOverridesSpec `json:"overrides"`
}

type CreateLoadBalancerSpec struct {
	Zone               string           `json:"zone"`
	Name               string           `json:"name"`
	Description        string           `json:"description"`
	Enabled            *bool            `json:"enabled"`
	Proxied            bool             `json:"proxied"`
	TTL                int              `json:"ttl"`
	FallbackPool       string           `json:"fallbackPool"`
	DefaultPools       []string         `json:"defaultPools"`
	SteeringPolicy     string           `json:"steeringPolicy"`
	SessionAffinity    string           `json:"sessionAffinity"`
	SessionAffinityTTL *int             `json:"sessionAffinityTtl"`
	PoolWeights        []PoolWeightSpec `json:"poolWeights"`
	NetworkVisibility  string           `json:"networkVisibility"`
	Networks           []string         `json:"networks"`
	Monitor            string           `json:"monitor"`
	Rules              []LBRuleSpec     `json:"rules"`
}

func (c *CreateLoadBalancer) Name() string {
	return "cloudflare.createLoadBalancer"
}

func (c *CreateLoadBalancer) Label() string {
	return "Create Load Balancer"
}

func (c *CreateLoadBalancer) Description() string {
	return "Create a Cloudflare Load Balancer on a zone, with support for private/public visibility, monitor attachment, and custom rules"
}

func (c *CreateLoadBalancer) Documentation() string {
	return `The Create Load Balancer component creates a new Cloudflare Load Balancer attached to a zone.

## Use Cases

- **Traffic distribution**: Route traffic across multiple origin pools
- **Blue/green deployments**: Create a load balancer backed by your new (green) pool
- **Multi-region failover**: Define ordered pools with a fallback for automatic failover

## Configuration

- **Zone**: The zone to attach the load balancer to
- **Name**: The hostname of the load balancer (e.g. ` + "`lb.example.com`" + `)
- **Fallback Pool**: Pool to use when all default pools are unhealthy
- **Default Pools**: Ordered list of pools to route traffic to
- **Steering Policy**: How traffic is distributed across pools
- **Session Affinity**: How client sessions are pinned to a pool
- **Pool Weights**: Per-pool weights used when steering policy is set to Random

## Output

Returns the created load balancer with its assigned ID and full configuration.`
}

func (c *CreateLoadBalancer) Icon() string {
	return "cloud"
}

func (c *CreateLoadBalancer) Color() string {
	return "orange"
}

func (c *CreateLoadBalancer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateLoadBalancer) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "zone",
			Label:       "Zone",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The zone to attach the load balancer to",
			Placeholder: "Select a zone",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "zone",
				},
			},
		},
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Hostname of the load balancer (e.g. lb.example.com)",
			Placeholder: "lb.example.com",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional description for the load balancer",
		},
		{
			Name:        "defaultPools",
			Label:       "Default Pools",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Ordered list of pools to route traffic to",
			Placeholder: "Select pools",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "pool",
					Multi: true,
				},
			},
		},
		{
			Name:        "fallbackPool",
			Label:       "Fallback Pool",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Pool to use when all default pools are unhealthy",
			Placeholder: "Select a pool",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "pool",
				},
			},
		},
		{
			Name:        "monitor",
			Label:       "Monitor",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Health monitor to attach to this load balancer for active health checking across all pools",
			Placeholder: "Select a monitor",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "monitor",
				},
			},
		},
		{
			Name:        "steeringPolicy",
			Label:       "Steering Policy",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Determines how traffic is distributed across pools",
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
			Name:        "sessionAffinity",
			Label:       "Session Affinity",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Determines how client sessions are pinned to a specific pool",
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
			Name:        "poolWeights",
			Label:       "Pool Weights",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Per-pool weights used when steering policy is Random",
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
			Name:        "proxied",
			Label:       "Proxied",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Whether Cloudflare proxies traffic to this load balancer",
		},
		{
			Name:        "ttl",
			Label:       "TTL (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "DNS TTL in seconds. Only applicable when proxied is disabled.",
		},
		{
			Name:        "enabled",
			Label:       "Enabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Whether the load balancer is active and accepts traffic",
		},
		{
			Name:        "networkVisibility",
			Label:       "Network Visibility",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "public",
			Description: "Whether this load balancer is publicly accessible or restricted to private virtual networks",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Public", Value: "public"},
						{Label: "Private", Value: "private"},
					},
				},
			},
		},
		{
			Name:        "networks",
			Label:       "Virtual Networks",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "IDs of the Cloudflare virtual networks to attach this private load balancer to. Only used when Network Visibility is set to Private.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Virtual Network ID",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "rules",
			Label:       "Rules",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Conditional override rules. When a rule's condition matches, its overrides replace the load balancer's default settings for that request.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Rule",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "name",
								Label:       "Rule Name",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Descriptive name for the rule",
							},
							{
								Name:        "condition",
								Label:       "Condition",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Firewall Rules expression that triggers this rule (e.g. http.request.uri.path contains \"/api\")",
								Placeholder: "http.request.uri.path contains \"/api\"",
							},
							{
								Name:        "priority",
								Label:       "Priority",
								Type:        configuration.FieldTypeNumber,
								Required:    false,
								Default:     0,
								Description: "Lower value = higher priority. Rules are evaluated in ascending priority order.",
							},
							{
								Name:        "disabled",
								Label:       "Disabled",
								Type:        configuration.FieldTypeBool,
								Required:    false,
								Default:     false,
								Description: "When true, this rule is skipped during evaluation",
							},
							{
								Name:        "overrides",
								Label:       "Overrides",
								Type:        configuration.FieldTypeObject,
								Required:    true,
								Description: "Load balancer settings to apply when this rule matches",
								TypeOptions: &configuration.TypeOptions{
									Object: &configuration.ObjectTypeOptions{
										Schema: []configuration.Field{
											{
												Name:        "steeringPolicy",
												Label:       "Steering Policy",
												Type:        configuration.FieldTypeSelect,
												Required:    false,
												Description: "Override steering policy for matched requests",
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
												Name:        "fallbackPool",
												Label:       "Fallback Pool",
												Type:        configuration.FieldTypeIntegrationResource,
												Required:    false,
												Description: "Override fallback pool for matched requests",
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
												Description: "Override default pools for matched requests",
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
												Name:        "sessionAffinity",
												Label:       "Session Affinity",
												Type:        configuration.FieldTypeSelect,
												Required:    false,
												Description: "Override session affinity for matched requests",
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
												Label:       "Session Affinity TTL",
												Type:        configuration.FieldTypeNumber,
												Required:    false,
												Description: "Override session affinity TTL (seconds) for matched requests",
											},
											{
												Name:        "poolWeights",
												Label:       "Pool Weights",
												Type:        configuration.FieldTypeList,
												Required:    false,
												Description: "Override pool weights for matched requests",
												TypeOptions: &configuration.TypeOptions{
													List: &configuration.ListTypeOptions{
														ItemLabel: "Pool Weight",
														ItemDefinition: &configuration.ListItemDefinition{
															Type: configuration.FieldTypeObject,
															Schema: []configuration.Field{
																{
																	Name:        "pool",
																	Label:       "Pool ID",
																	Type:        configuration.FieldTypeString,
																	Required:    true,
																	Description: "Pool identifier",
																},
																{
																	Name:        "weight",
																	Label:       "Weight",
																	Type:        configuration.FieldTypeNumber,
																	Required:    true,
																	Description: "Weight for random steering (0.0–1.0)",
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (c *CreateLoadBalancer) Setup(ctx core.SetupContext) error {
	spec := CreateLoadBalancerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Zone == "" {
		return errors.New("zone is required")
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}

	if spec.FallbackPool == "" {
		return errors.New("fallbackPool is required")
	}

	if len(spec.DefaultPools) == 0 {
		return errors.New("at least one defaultPool is required")
	}

	return nil
}

func (c *CreateLoadBalancer) Execute(ctx core.ExecutionContext) error {
	spec := CreateLoadBalancerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
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

	var networks []string
	if spec.NetworkVisibility == "private" && len(spec.Networks) > 0 {
		networks = spec.Networks
	}

	var rules []LBRule
	for _, r := range spec.Rules {
		var ruleRandomSteering *RandomSteering
		if len(r.Overrides.PoolWeights) > 0 {
			weights := make(map[string]float64, len(r.Overrides.PoolWeights))
			for _, pw := range r.Overrides.PoolWeights {
				weights[pw.Pool] = pw.Weight
			}
			ruleRandomSteering = &RandomSteering{PoolWeights: weights}
		}
		rules = append(rules, LBRule{
			Name:      r.Name,
			Condition: r.Condition,
			Disabled:  r.Disabled,
			Priority:  r.Priority,
			Overrides: LBRuleOverrides{
				SteeringPolicy:     r.Overrides.SteeringPolicy,
				FallbackPool:       r.Overrides.FallbackPool,
				DefaultPools:       r.Overrides.DefaultPools,
				SessionAffinity:    r.Overrides.SessionAffinity,
				SessionAffinityTTL: r.Overrides.SessionAffinityTTL,
				RandomSteering:     ruleRandomSteering,
			},
		})
	}

	req := CreateLoadBalancerRequest{
		Name:               spec.Name,
		Description:        spec.Description,
		Enabled:            spec.Enabled,
		Proxied:            spec.Proxied,
		TTL:                spec.TTL,
		FallbackPool:       spec.FallbackPool,
		DefaultPools:       spec.DefaultPools,
		SteeringPolicy:     spec.SteeringPolicy,
		SessionAffinity:    spec.SessionAffinity,
		SessionAffinityTTL: spec.SessionAffinityTTL,
		RandomSteering:     randomSteering,
		Networks:           networks,
		Rules:              rules,
		Monitor:            spec.Monitor,
	}

	lb, err := client.CreateLoadBalancer(spec.Zone, req)
	if err != nil {
		return fmt.Errorf("failed to create load balancer: %v", err)
	}

	result := map[string]any{
		"loadBalancer": lb,
		"zoneId":       spec.Zone,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.loadBalancer.created",
		[]any{result},
	)
}

func (c *CreateLoadBalancer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateLoadBalancer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateLoadBalancer) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateLoadBalancer) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateLoadBalancer) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateLoadBalancer) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
