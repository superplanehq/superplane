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

const GetTunnelPayloadType = "cloudflare.tunnel.fetched"

type GetTunnel struct{}

type GetTunnelSpec struct {
	AccountID string `json:"accountId"`
	Tunnel    string `json:"tunnel"`
}

func (c *GetTunnel) Name() string {
	return "cloudflare.getTunnel"
}

func (c *GetTunnel) Label() string {
	return "Get Tunnel"
}

func (c *GetTunnel) Description() string {
	return "Retrieve configuration and status for a Cloudflare Tunnel by ID"
}

func (c *GetTunnel) Documentation() string {
	return `The Get Tunnel component fetches the current state of a Cloudflare Tunnel (cloudflared).

## Use Cases

- **Health and status checks**: Inspect tunnel status in a workflow
- **Validation**: Confirm a tunnel exists before updating routes or DNS

## Configuration

- **Tunnel**: The tunnel ID to retrieve

## Output

Returns the tunnel object from the Cloudflare API (ID, name, status, config source, timestamps, and related fields).`
}

func (c *GetTunnel) Icon() string {
	return "cloud"
}

func (c *GetTunnel) Color() string {
	return "orange"
}

func (c *GetTunnel) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetTunnel) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "tunnel",
			Label:       "Tunnel",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Cloudflare Tunnel to retrieve",
			Placeholder: "Select a tunnel",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "tunnel",
				},
			},
		},
	}
}

func (c *GetTunnel) Setup(ctx core.SetupContext) error {
	spec := GetTunnelSpec{}
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

func (c *GetTunnel) Execute(ctx core.ExecutionContext) error {
	spec := GetTunnelSpec{}
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

	tunnel, err := client.GetCFDTunnel(accountID, spec.Tunnel)
	if err != nil {
		return fmt.Errorf("failed to get tunnel: %v", err)
	}

	result := map[string]any{
		"tunnel":    tunnel,
		"accountId": accountID,
		"tunnelId":  spec.Tunnel,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetTunnelPayloadType,
		[]any{result},
	)
}

func (c *GetTunnel) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetTunnel) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetTunnel) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetTunnel) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetTunnel) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetTunnel) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
