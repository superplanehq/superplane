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

type DeletePool struct{}

type DeletePoolSpec struct {
	AccountID string `json:"accountId"`
	Pool      string `json:"pool"`
}

func (c *DeletePool) Name() string {
	return "cloudflare.deletePool"
}

func (c *DeletePool) Label() string {
	return "Delete Pool"
}

func (c *DeletePool) Description() string {
	return "Delete a Cloudflare Load Balancer origin pool"
}

func (c *DeletePool) Documentation() string {
	return `The Delete Pool component permanently removes a Cloudflare Load Balancer origin pool.

## Use Cases

- **Blue/green deployments**: Clean up the old (blue) pool after traffic has shifted
- **Environment teardown**: Remove pools as part of infrastructure cleanup

## Configuration

- **Pool ID**: The origin pool to delete

## Output

Emits a confirmation with the account ID and pool ID of the deleted pool.

> **Warning**: This operation is irreversible. Ensure the pool is not attached to any load balancer before deleting.`
}

func (c *DeletePool) Icon() string {
	return "cloud"
}

func (c *DeletePool) Color() string {
	return "orange"
}

func (c *DeletePool) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeletePool) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "pool",
			Label:       "Pool",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The origin pool to delete",
			Placeholder: "Select a pool",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "pool",
				},
			},
		},
	}
}

func (c *DeletePool) Setup(ctx core.SetupContext) error {
	spec := DeletePoolSpec{}
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

	return c.resolvePoolMetadata(ctx, accountID, spec.Pool)
}

func (c *DeletePool) resolvePoolMetadata(ctx core.SetupContext, accountID, poolID string) error {
	return resolvePoolMetadata(ctx, accountID, poolID)
}

func (c *DeletePool) Execute(ctx core.ExecutionContext) error {
	spec := DeletePoolSpec{}
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

	if err := client.DeletePool(accountID, spec.Pool); err != nil {
		return fmt.Errorf("failed to delete pool: %v", err)
	}

	result := map[string]any{
		"accountId": accountID,
		"poolId":    spec.Pool,
		"deleted":   true,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.pool.deleted",
		[]any{result},
	)
}

func (c *DeletePool) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeletePool) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeletePool) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeletePool) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeletePool) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeletePool) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
