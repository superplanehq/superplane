package hetzner

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("hetzner", &Hetzner{})
}

type Hetzner struct{}

type Configuration struct {
	APIToken string `json:"apiToken" mapstructure:"apiToken"`
}

func (h *Hetzner) Name() string {
	return "hetzner"
}

func (h *Hetzner) Label() string {
	return "Hetzner Cloud"
}

func (h *Hetzner) Icon() string {
	return "server"
}

func (h *Hetzner) Description() string {
	return "Create and delete Hetzner Cloud servers"
}

func (h *Hetzner) Instructions() string {
	return `
1. **API Token:** Create a token in [Hetzner Cloud Console](https://console.hetzner.cloud/) → Project → Security → API Tokens. Use **Read & Write** scope.
2. **Auth:** SuperPlane sends requests to the [Hetzner Cloud API](https://docs.hetzner.cloud/) using ` + "`Authorization: Bearer <token>`" + `.
`
}

func (h *Hetzner) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Hetzner Cloud API token with Read & Write access",
		},
	}
}

func (h *Hetzner) Components() []core.Component {
	return []core.Component{
		&CreateServer{},
		&DeleteServer{},
	}
}

func (h *Hetzner) Triggers() []core.Trigger {
	return nil
}

func (h *Hetzner) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (h *Hetzner) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(config.APIToken) == "" {
		return fmt.Errorf("apiToken is required")
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	if err := client.Verify(); err != nil {
		return fmt.Errorf("failed to verify Hetzner credentials: %w", err)
	}
	ctx.Integration.Ready()
	return nil
}

func (h *Hetzner) HandleRequest(ctx core.HTTPRequestContext) {}

func (h *Hetzner) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "server" {
		return nil, nil
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}
	servers, err := client.ListServers()
	if err != nil {
		return nil, err
	}
	resources := make([]core.IntegrationResource, 0, len(servers))
	for _, s := range servers {
		id := fmt.Sprintf("%d", s.ID)
		name := s.Name
		if name == "" {
			name = id
		}
		resources = append(resources, core.IntegrationResource{Type: "server", Name: name, ID: id})
	}
	return resources, nil
}

func (h *Hetzner) Actions() []core.Action {
	return nil
}

func (h *Hetzner) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
