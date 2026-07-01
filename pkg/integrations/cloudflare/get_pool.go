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

type GetPool struct{}

type GetPoolSpec struct {
	AccountID string `json:"accountId"`
	Pool      string `json:"pool"`
}

func (c *GetPool) Name() string {
	return "cloudflare.getPool"
}

func (c *GetPool) Label() string {
	return "Get Pool"
}

func (c *GetPool) Description() string {
	return "Retrieve a Cloudflare Load Balancer origin pool by ID"
}

func (c *GetPool) Documentation() string {
	return `The Get Pool component fetches the current state of a Cloudflare Load Balancer origin pool.

## Use Cases

- **Health checks**: Inspect origin health and pool status in a workflow
- **Pre-flight validation**: Confirm a pool exists before updating it
- **Audit**: Capture a snapshot of pool configuration at a point in time

## Configuration

- **Pool ID**: The origin pool to retrieve

## Output

Returns the full pool configuration including its origins, enabled state, and health monitor.`
}

func (c *GetPool) Icon() string {
	return "cloud"
}

func (c *GetPool) Color() string {
	return "orange"
}

func (c *GetPool) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetPool) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "pool",
			Label:       "Pool",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The origin pool to retrieve",
			Placeholder: "Select a pool",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "pool",
				},
			},
		},
	}
}

func (c *GetPool) Setup(ctx core.SetupContext) error {
	spec := GetPoolSpec{}
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

func (c *GetPool) resolvePoolMetadata(ctx core.SetupContext, accountID, poolID string) error {
	return resolvePoolMetadata(ctx, accountID, poolID)
}

func (c *GetPool) Execute(ctx core.ExecutionContext) error {
	spec := GetPoolSpec{}
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

	pool, err := client.GetPool(accountID, spec.Pool)
	if err != nil {
		return fmt.Errorf("failed to get pool: %v", err)
	}

	result := map[string]any{
		"pool":      pool,
		"accountId": accountID,
		"poolId":    spec.Pool,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.pool.fetched",
		[]any{result},
	)
}

func (c *GetPool) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetPool) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetPool) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetPool) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetPool) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetPool) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
