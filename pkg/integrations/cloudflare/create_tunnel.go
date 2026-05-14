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

const CreateTunnelPayloadType = "cloudflare.tunnel.created"

type CreateTunnel struct{}

type CreateTunnelSpec struct {
	AccountID string `json:"accountId"`
	Name      string `json:"name"`
	ConfigSrc string `json:"configSrc"`
}

func (c *CreateTunnel) Name() string {
	return "cloudflare.createTunnel"
}

func (c *CreateTunnel) Label() string {
	return "Create Tunnel"
}

func (c *CreateTunnel) Description() string {
	return "Create a new Cloudflare Tunnel (cloudflared) for secure outbound-only connectivity to Cloudflare"
}

func (c *CreateTunnel) Documentation() string {
	return `The Create Tunnel component provisions a Cloudflare Tunnel under your account.

## Use Cases

- **Expose internal services**: Pair with ingress rules so private origins are reachable through Cloudflare without opening inbound firewall ports
- **Automation**: Create tunnels as part of environment provisioning

## Configuration

- **Name**: A unique name for the tunnel
- **Config source**: ` + "`cloudflare`" + ` (managed in Zero Trust) or ` + "`local`" + ` (managed via local cloudflared configuration)

## Output

Returns the tunnel resource from the Cloudflare API, including its ID. When Cloudflare returns a connector token, it is included in the output (treat it as a secret).

> **Note**: Deleting a tunnel does not automatically remove all public hostname routes; use Cloudflare Zero Trust or DNS workflows as needed.`
}

func (c *CreateTunnel) Icon() string {
	return "cloud"
}

func (c *CreateTunnel) Color() string {
	return "orange"
}

func (c *CreateTunnel) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateTunnel) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Tunnel Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "A unique name for the Cloudflare Tunnel",
			Placeholder: "my-app-tunnel",
		},
		{
			Name:        "configSrc",
			Label:       "Config Source",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "cloudflare",
			Description: "Whether tunnel configuration is managed in Cloudflare (remote) or locally in cloudflared",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Cloudflare (remote)", Value: "cloudflare"},
						{Label: "Local (cloudflared)", Value: "local"},
					},
				},
			},
		},
	}
}

func (c *CreateTunnel) Setup(ctx core.SetupContext) error {
	spec := CreateTunnelSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)
	if accountID == "" {
		return errors.New("accountId is required")
	}

	if strings.TrimSpace(spec.Name) == "" {
		return errors.New("name is required")
	}

	if normalizeTunnelConfigSrc(spec.ConfigSrc) == "" {
		return fmt.Errorf("configSrc must be cloudflare or local")
	}

	return nil
}

func (c *CreateTunnel) Execute(ctx core.ExecutionContext) error {
	spec := CreateTunnelSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)
	if accountID == "" {
		return errors.New("accountId is required")
	}

	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return errors.New("name is required")
	}

	configSrc := normalizeTunnelConfigSrc(spec.ConfigSrc)
	if configSrc == "" {
		return fmt.Errorf("configSrc must be cloudflare or local")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	tunnel, err := client.CreateCFDTunnel(accountID, CreateCFDTunnelRequest{
		Name:      name,
		ConfigSrc: configSrc,
	})
	if err != nil {
		return fmt.Errorf("failed to create tunnel: %w", err)
	}

	result := map[string]any{
		"tunnel":    tunnel,
		"accountId": accountID,
		"tunnelId":  tunnel.ID,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CreateTunnelPayloadType,
		[]any{result},
	)
}

func (c *CreateTunnel) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateTunnel) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateTunnel) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateTunnel) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateTunnel) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateTunnel) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func normalizeTunnelConfigSrc(value string) string {
	v := strings.TrimSpace(strings.ToLower(value))
	switch v {
	case "", "cloudflare":
		return "cloudflare"
	case "local":
		return "local"
	default:
		return ""
	}
}
