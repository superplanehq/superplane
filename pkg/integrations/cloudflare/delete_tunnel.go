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

const DeleteTunnelPayloadType = "cloudflare.tunnel.deleted"

type DeleteTunnel struct{}

type DeleteTunnelSpec struct {
	AccountID string `json:"accountId"`
	Tunnel    string `json:"tunnel"`
}

func (c *DeleteTunnel) Name() string {
	return "cloudflare.deleteTunnel"
}

func (c *DeleteTunnel) Label() string {
	return "Delete Tunnel"
}

func (c *DeleteTunnel) Description() string {
	return "Delete a Cloudflare Tunnel and clean up its connector credentials"
}

func (c *DeleteTunnel) Documentation() string {
	return `The Delete Tunnel component permanently removes a Cloudflare Tunnel.

## Use Cases

- **Teardown**: Remove tunnels when an environment is decommissioned
- **Rotation**: Delete an old tunnel after migrating to a new one

## Configuration

- **Tunnel**: The tunnel ID to delete

## Output

Emits confirmation with the account ID and tunnel ID.

> **Warning**: This operation is irreversible. Active traffic using the tunnel will fail until reconfigured.`
}

func (c *DeleteTunnel) Icon() string {
	return "cloud"
}

func (c *DeleteTunnel) Color() string {
	return "orange"
}

func (c *DeleteTunnel) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteTunnel) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "tunnel",
			Label:       "Tunnel",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Cloudflare Tunnel to delete",
			Placeholder: "Select a tunnel",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "tunnel",
				},
			},
		},
	}
}

func (c *DeleteTunnel) Setup(ctx core.SetupContext) error {
	spec := DeleteTunnelSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)
	if accountID == "" {
		return errors.New("accountId is required")
	}

	if spec.Tunnel == "" {
		return errors.New("tunnel is required")
	}

	return resolveTunnelMetadata(ctx, accountID, spec.Tunnel)
}

func (c *DeleteTunnel) Execute(ctx core.ExecutionContext) error {
	spec := DeleteTunnelSpec{}
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

	if err := client.DeleteCFDTunnel(accountID, spec.Tunnel); err != nil {
		return fmt.Errorf("failed to delete tunnel: %w", err)
	}

	result := map[string]any{
		"accountId": accountID,
		"tunnelId":  spec.Tunnel,
		"deleted":   true,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		DeleteTunnelPayloadType,
		[]any{result},
	)
}

func (c *DeleteTunnel) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteTunnel) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteTunnel) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteTunnel) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteTunnel) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteTunnel) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
