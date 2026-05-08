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

type CreatePool struct{}

type LoadSheddingSpec struct {
	DefaultPercent float64 `json:"defaultPercent"`
	DefaultPolicy  string  `json:"defaultPolicy"`
	SessionPercent float64 `json:"sessionPercent"`
	SessionPolicy  string  `json:"sessionPolicy"`
}

type CreatePoolSpec struct {
	AccountID            string            `json:"accountId"`
	Name                 string            `json:"name"`
	Description          string            `json:"description"`
	Enabled              *bool             `json:"enabled"`
	MinimumOrigins       *int              `json:"minimumOrigins"`
	Monitor              string            `json:"monitor"`
	Origins              []OriginSpec      `json:"origins"`
	OriginSteeringPolicy string            `json:"originSteeringPolicy"`
	LoadShedding         *LoadSheddingSpec `json:"loadShedding"`
}

type CoordinatesSpec struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type OriginSpec struct {
	Name        string           `json:"name"`
	Address     string           `json:"address"`
	Enabled     *bool            `json:"enabled"`
	Weight      *float64         `json:"weight"`
	Port        int              `json:"port"`
	Coordinates *CoordinatesSpec `json:"coordinates"`
}

func (c *CreatePool) Name() string {
	return "cloudflare.createPool"
}

func (c *CreatePool) Label() string {
	return "Create Pool"
}

func (c *CreatePool) Description() string {
	return "Create a Cloudflare Load Balancer origin pool"
}

func (c *CreatePool) Documentation() string {
	return `The Create Pool component creates a new Cloudflare Load Balancer origin pool.

## Use Cases

- **Canary deployments**: Provision a new origin pool for a canary release
- **Blue/green deployments**: Create the green pool before switching traffic
- **Multi-region**: Add origin servers in new regions

## Configuration

- **Name**: A unique, human-readable name for the pool
- **Description**: Optional description
- **Origins**: List of origin servers with name, address, enabled flag, weight, optional port, and optional coordinates
- **Enabled**: Whether the pool is active
- **Minimum Origins**: Minimum number of healthy origins before marking pool unhealthy
- **Origin Steering Policy**: How requests are distributed across origins
- **Monitor**: (Optional) Health monitor to use for origin health checks
- **Load Shedding**: (Optional) Configure load shedding for the pool

## Output

Returns the created origin pool with its assigned ID and full configuration.`
}

func (c *CreatePool) Icon() string {
	return "cloud"
}

func (c *CreatePool) Color() string {
	return "orange"
}

func (c *CreatePool) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreatePool) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Pool Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "A unique, human-readable name for the origin pool",
			Placeholder: "my-origin-pool",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional description of the origin pool",
		},
		{
			Name:        "origins",
			Label:       "Origins",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "List of origin servers in the pool",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Origin",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "name",
								Label:       "Name",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Human-readable name for the origin",
								Placeholder: "origin-1",
							},
							{
								Name:        "address",
								Label:       "Address",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "IPv4 address or hostname of the origin",
								Placeholder: "192.0.2.1",
							},
							{
								Name:        "port",
								Label:       "Port",
								Type:        configuration.FieldTypeNumber,
								Required:    false,
								Description: "Optional port to append to the address (e.g. 8080). Leave empty to use the default.",
							},
							{
								Name:        "weight",
								Label:       "Weight",
								Type:        configuration.FieldTypeNumber,
								Required:    false,
								Default:     1,
								Description: "Traffic weight for this origin (0.0–1.0)",
							},
							{
								Name:        "coordinates",
								Label:       "Coordinates",
								Type:        configuration.FieldTypeObject,
								Required:    false,
								Togglable:   true,
								Description: "Geographic coordinates for proximity steering",
								TypeOptions: &configuration.TypeOptions{
									Object: &configuration.ObjectTypeOptions{
										Schema: []configuration.Field{
											{
												Name:        "latitude",
												Label:       "Latitude",
												Type:        configuration.FieldTypeNumber,
												Required:    false,
												Description: "Geographic latitude for proximity steering (e.g. 51.5074)",
											},
											{
												Name:        "longitude",
												Label:       "Longitude",
												Type:        configuration.FieldTypeNumber,
												Required:    false,
												Description: "Geographic longitude for proximity steering (e.g. -0.1278)",
											},
										},
									},
								},
							},
							{
								Name:     "enabled",
								Label:    "Enabled",
								Type:     configuration.FieldTypeBool,
								Required: false,
								Default:  true,
							},
						},
					},
				},
			},
		},
		{
			Name:        "minimumOrigins",
			Label:       "Minimum Origins",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     1,
			Description: "Minimum number of healthy origins before the pool is marked as unhealthy",
		},
		{
			Name:        "originSteeringPolicy",
			Label:       "Origin Steering Policy",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Determines how requests are distributed across origins",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Random", Value: "random"},
						{Label: "Hash (URI)", Value: "hash"},
						{Label: "Least Outstanding Requests", Value: "least_outstanding_requests"},
						{Label: "Least Connections", Value: "least_connections"},
					},
				},
			},
		},
		{
			Name:        "monitor",
			Label:       "Monitor",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Health monitor to attach to this pool",
			Placeholder: "Select a monitor",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "monitor",
				},
			},
		},
		{
			Name:        "loadShedding",
			Label:       "Load Shedding",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Togglable:   true,
			Description: "Configure load shedding to drop a percentage of traffic to the pool",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:        "defaultPercent",
							Label:       "Default Percent",
							Type:        configuration.FieldTypeNumber,
							Required:    false,
							Default:     0,
							Description: "Percentage of traffic to shed from all sessions (0–100)",
						},
						{
							Name:        "defaultPolicy",
							Label:       "Default Policy",
							Type:        configuration.FieldTypeSelect,
							Required:    false,
							Description: "Policy for shedding default (non-session-affinity) traffic",
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "Random", Value: "random"},
										{Label: "Hash", Value: "hash"},
									},
								},
							},
						},
						{
							Name:        "sessionPercent",
							Label:       "Session Percent",
							Type:        configuration.FieldTypeNumber,
							Required:    false,
							Default:     0,
							Description: "Percentage of existing sessions to shed (0–100)",
						},
						{
							Name:        "sessionPolicy",
							Label:       "Session Policy",
							Type:        configuration.FieldTypeSelect,
							Required:    false,
							Description: "Policy for shedding session-affinity traffic",
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "Hash", Value: "hash"},
									},
								},
							},
						},
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
			Description: "Whether the pool is active and receives traffic",
		},
	}
}

func (c *CreatePool) Setup(ctx core.SetupContext) error {
	spec := CreatePoolSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)
	if accountID == "" {
		return errors.New("accountId is required")
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}

	if len(spec.Origins) == 0 {
		return errors.New("at least one origin is required")
	}

	for i, o := range spec.Origins {
		if o.Name == "" {
			return fmt.Errorf("origins[%d].name is required", i)
		}
		if o.Address == "" {
			return fmt.Errorf("origins[%d].address is required", i)
		}
	}

	return nil
}

func (c *CreatePool) Execute(ctx core.ExecutionContext) error {
	spec := CreatePoolSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)
	if accountID == "" {
		return errors.New("accountId is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	origins := make([]Origin, len(spec.Origins))
	for i, o := range spec.Origins {
		weight := 1.0
		if o.Weight != nil {
			weight = *o.Weight
		}

		address := o.Address

		enabled := true
		if o.Enabled != nil {
			enabled = *o.Enabled
		}

		var coords *Coordinates
		if o.Coordinates != nil {
			coords = &Coordinates{Latitude: o.Coordinates.Latitude, Longitude: o.Coordinates.Longitude}
		}

		origins[i] = Origin{
			Name:        o.Name,
			Address:     address,
			Enabled:     enabled,
			Weight:      weight,
			Port:        o.Port,
			Coordinates: coords,
		}
	}

	var originSteering *OriginSteering
	if spec.OriginSteeringPolicy != "" {
		originSteering = &OriginSteering{Policy: spec.OriginSteeringPolicy}
	}

	var loadShedding *LoadShedding
	if spec.LoadShedding != nil {
		loadShedding = &LoadShedding{
			DefaultPercent: spec.LoadShedding.DefaultPercent,
			DefaultPolicy:  spec.LoadShedding.DefaultPolicy,
			SessionPercent: spec.LoadShedding.SessionPercent,
			SessionPolicy:  spec.LoadShedding.SessionPolicy,
		}
	}

	enabled := true
	if spec.Enabled != nil {
		enabled = *spec.Enabled
	}

	req := CreatePoolRequest{
		Name:           spec.Name,
		Description:    spec.Description,
		Enabled:        enabled,
		MinimumOrigins: spec.MinimumOrigins,
		Monitor:        spec.Monitor,
		Origins:        origins,
		OriginSteering: originSteering,
		LoadShedding:   loadShedding,
	}

	pool, err := client.CreatePool(accountID, req)
	if err != nil {
		return fmt.Errorf("failed to create pool: %v", err)
	}

	result := map[string]any{
		"pool":      pool,
		"accountId": accountID,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.pool.created",
		[]any{result},
	)
}

func (c *CreatePool) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreatePool) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreatePool) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreatePool) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreatePool) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreatePool) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
