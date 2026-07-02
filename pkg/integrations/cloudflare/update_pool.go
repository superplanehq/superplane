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

type UpdatePool struct{}

type UpdatePoolSpec struct {
	AccountID            string            `json:"accountId"`
	Pool                 string            `json:"pool"`
	Name                 string            `json:"name"`
	Description          string            `json:"description"`
	Enabled              *bool             `json:"enabled"`
	MinimumOrigins       *int              `json:"minimumOrigins"`
	Monitor              string            `json:"monitor"`
	Origins              []OriginSpec      `json:"origins"`
	OriginSteeringPolicy string            `json:"originSteeringPolicy"`
	LoadShedding         *LoadSheddingSpec `json:"loadShedding"`
}

func (c *UpdatePool) Name() string {
	return "cloudflare.updatePool"
}

func (c *UpdatePool) Label() string {
	return "Update Pool"
}

func (c *UpdatePool) Description() string {
	return "Update a Cloudflare Load Balancer origin pool's configuration or origin weights"
}

func (c *UpdatePool) Documentation() string {
	return `The Update Pool component modifies an existing Cloudflare Load Balancer origin pool.

## Use Cases

- **Canary deployments**: Shift traffic weight from stable to canary origin
- **Blue/green deployments**: Disable blue origins and enable green origins
- **Scaling**: Add or remove origin servers dynamically
- **Health management**: Enable or disable individual origins without removing them

## Configuration

- **Pool ID**: The ID of the pool to update
- **Origins**: Full list of origin servers with updated weights or enabled status
- **Name**: Optional new name for the pool
- **Description**: Optional new description
- **Enabled**: Enable or disable the entire pool
- **Minimum Origins**: Minimum number of healthy origins before pool is marked unhealthy
- **Origin Steering Policy**: How requests are distributed across origins
- **Monitor**: Health monitor ID for origin health checks
- **Load Shedding**: Configure load shedding for the pool

## Output

Returns the updated origin pool configuration.`
}

func (c *UpdatePool) Icon() string {
	return "cloud"
}

func (c *UpdatePool) Color() string {
	return "orange"
}

func (c *UpdatePool) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdatePool) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "pool",
			Label:       "Pool",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The origin pool to update",
			Placeholder: "Select a pool",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "pool",
				},
			},
		},
		{
			Name:        "origins",
			Label:       "Origins",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Updated list of origin servers. When provided, replaces the current list.",
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
								Name:     "enabled",
								Label:    "Enabled",
								Type:     configuration.FieldTypeBool,
								Required: false,
								Default:  true,
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
						},
					},
				},
			},
		},
		{
			Name:        "name",
			Label:       "Pool Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "New name for the pool (optional)",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "New description for the pool (optional)",
		},
		{
			Name:        "enabled",
			Label:       "Enabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Togglable:   true,
			Default:     true,
			Description: "Enable or disable the pool",
		},
		{
			Name:        "minimumOrigins",
			Label:       "Minimum Origins",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Minimum number of healthy origins before the pool is marked as unhealthy",
		},
		{
			Name:        "originSteeringPolicy",
			Label:       "Origin Steering Policy",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
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
			Togglable:   true,
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
	}
}

func (c *UpdatePool) Setup(ctx core.SetupContext) error {
	spec := UpdatePoolSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)
	if accountID == "" {
		return errors.New("accountId is required")
	}

	if spec.Pool == "" {
		return errors.New("pool is required")
	}

	for i, o := range spec.Origins {
		if o.Name == "" {
			return fmt.Errorf("origins[%d].name is required", i)
		}
		if o.Address == "" {
			return fmt.Errorf("origins[%d].address is required", i)
		}
	}

	return c.resolvePoolMetadata(ctx, accountID, spec.Pool)
}

func (c *UpdatePool) resolvePoolMetadata(ctx core.SetupContext, accountID, poolID string) error {
	return resolvePoolMetadata(ctx, accountID, poolID)
}

func (c *UpdatePool) Execute(ctx core.ExecutionContext) error {
	spec := UpdatePoolSpec{}
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

	req := UpdatePoolRequest{
		Name:           spec.Name,
		Description:    spec.Description,
		Enabled:        spec.Enabled,
		MinimumOrigins: spec.MinimumOrigins,
		Monitor:        spec.Monitor,
		OriginSteering: originSteering,
		LoadShedding:   loadShedding,
	}

	if len(spec.Origins) > 0 {
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

		req.Origins = origins
	}

	pool, err := client.UpdatePool(accountID, spec.Pool, req)
	if err != nil {
		return fmt.Errorf("failed to update pool: %v", err)
	}

	result := map[string]any{
		"pool":      pool,
		"accountId": accountID,
		"poolId":    spec.Pool,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.pool.updated",
		[]any{result},
	)
}

func (c *UpdatePool) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdatePool) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdatePool) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdatePool) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdatePool) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdatePool) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
