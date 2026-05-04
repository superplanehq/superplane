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
	AccountID         string       `json:"accountId"`
	PoolID            string       `json:"poolId"`
	Name              string       `json:"name"`
	Description       string       `json:"description"`
	Enabled           *bool        `json:"enabled"`
	MinimumOrigins    *int         `json:"minimumOrigins"`
	Monitor           string       `json:"monitor"`
	NotificationEmail string       `json:"notificationEmail"`
	Origins           []OriginSpec `json:"origins"`
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

- **Account ID**: The Cloudflare account ID that owns the pool
- **Pool ID**: The ID of the pool to update
- **Origins**: Full list of origin servers with updated weights or enabled status
- **Name**: Optional new name for the pool
- **Description**: Optional new description
- **Enabled**: Enable or disable the entire pool
- **Minimum Origins**: Minimum number of healthy origins before pool is marked unhealthy
- **Monitor**: Health monitor ID for origin health checks
- **Notification Email**: Email to notify on health changes

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
			Name:        "accountId",
			Label:       "Account ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The Cloudflare account ID that owns the pool",
		},
		{
			Name:        "poolId",
			Label:       "Pool ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the origin pool to update",
		},
		{
			Name:        "origins",
			Label:       "Origins",
			Type:        configuration.FieldTypeList,
			Required:    false,
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
								Description: "IP address or hostname of the origin",
								Placeholder: "192.0.2.1",
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
			Description: "New name for the pool (optional)",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "New description for the pool (optional)",
		},
		{
			Name:        "enabled",
			Label:       "Enabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Enable or disable the pool",
		},
		{
			Name:        "minimumOrigins",
			Label:       "Minimum Origins",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Minimum number of healthy origins before the pool is marked as unhealthy",
		},
		{
			Name:        "monitor",
			Label:       "Monitor ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Health monitor ID to attach to this pool",
		},
		{
			Name:        "notificationEmail",
			Label:       "Notification Email",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Email address to notify when pool health changes",
		},
	}
}

func (c *UpdatePool) Setup(ctx core.SetupContext) error {
	spec := UpdatePoolSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.AccountID == "" {
		return errors.New("accountId is required")
	}

	if spec.PoolID == "" {
		return errors.New("poolId is required")
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

func (c *UpdatePool) Execute(ctx core.ExecutionContext) error {
	spec := UpdatePoolSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	req := UpdatePoolRequest{
		Name:              spec.Name,
		Description:       spec.Description,
		Enabled:           spec.Enabled,
		MinimumOrigins:    spec.MinimumOrigins,
		Monitor:           spec.Monitor,
		NotificationEmail: spec.NotificationEmail,
	}

	if len(spec.Origins) > 0 {
		origins := make([]Origin, len(spec.Origins))
		for i, o := range spec.Origins {
			weight := o.Weight
			if weight == 0 {
				weight = 1.0
			}

			origins[i] = Origin{
				Name:    o.Name,
				Address: o.Address,
				Enabled: o.Enabled,
				Weight:  weight,
			}
		}

		req.Origins = origins
	}

	pool, err := client.UpdatePool(spec.AccountID, spec.PoolID, req)
	if err != nil {
		return fmt.Errorf("failed to update pool: %v", err)
	}

	result := map[string]any{
		"pool":      pool,
		"accountId": spec.AccountID,
		"poolId":    spec.PoolID,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.pool",
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
