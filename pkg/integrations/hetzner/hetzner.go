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
	return "hetzner"
}

func (h *Hetzner) Description() string {
	return "Create and delete Hetzner Cloud servers and load balancers"
}

func (h *Hetzner) Instructions() string {
	return `
**API Token:** Create a token in [Hetzner Cloud Console](https://console.hetzner.cloud/) → Project → Security → API Tokens. Use **Read & Write** scope.
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
		&CreateLoadBalancer{},
		&DeleteLoadBalancer{},
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
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	switch resourceType {
	case "server":
		servers, err := client.ListServers()
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(servers))
		for _, s := range servers {
			id := s.ID
			name := s.Name
			if name == "" {
				name = id
			}
			resources = append(resources, core.IntegrationResource{Type: "server", Name: name, ID: id})
		}
		return resources, nil
	case "load_balancer":
		loadBalancers, err := client.ListLoadBalancers()
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(loadBalancers))
		for _, lb := range loadBalancers {
			id := lb.ID
			name := lb.Name
			if name == "" {
				name = id
			}
			resources = append(resources, core.IntegrationResource{Type: "load_balancer", Name: name, ID: id})
		}
		return resources, nil
	case "server_type":
		types, err := client.ListServerTypes()
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(types))
		for _, t := range types {
			id := t.Name
			if id == "" {
				id = fmt.Sprintf("%d", t.ID)
			}
			displayName := t.ServerTypeDisplayName()
			if displayName == "" {
				displayName = id
			}
			resources = append(resources, core.IntegrationResource{Type: "server_type", Name: displayName, ID: id})
		}
		return resources, nil
	case "load_balancer_type":
		types, err := client.ListLoadBalancerTypes()
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(types))
		for _, t := range types {
			id := t.Name
			if id == "" {
				id = fmt.Sprintf("%d", t.Id)
			}
			resources = append(resources, core.IntegrationResource{Type: "load_balancer_type", Name: t.Name, ID: id})
		}

		return resources, nil
	case "image":
		images, err := client.ListImages()
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(images))
		for _, img := range images {
			id := img.Name
			if id == "" {
				id = fmt.Sprintf("%d", img.ID)
			}
			resources = append(resources, core.IntegrationResource{Type: "image", Name: img.Name, ID: id})
		}
		return resources, nil
	case "location":
		locations, err := client.ListLocations()
		if err != nil {
			return nil, err
		}
		if serverType := ctx.Parameters["serverType"]; serverType != "" {
			allowedNames, err := client.ServerTypeLocationNames(serverType)
			if err == nil && len(allowedNames) > 0 {
				allowed := make(map[string]bool)
				for _, n := range allowedNames {
					allowed[n] = true
				}
				filtered := locations[:0]
				for _, loc := range locations {
					if allowed[loc.Name] {
						filtered = append(filtered, loc)
					}
				}
				locations = filtered
			}
		}
		resources := make([]core.IntegrationResource, 0, len(locations))
		for _, loc := range locations {
			id := loc.Name
			if id == "" {
				id = fmt.Sprintf("%d", loc.ID)
			}
			displayName := loc.LocationDisplayName()
			if displayName == "" {
				displayName = id
			}
			resources = append(resources, core.IntegrationResource{Type: "location", Name: displayName, ID: id})
		}
		return resources, nil
	case "load_balancing_algorithm":
		return []core.IntegrationResource{
			{
				Type: "load_balancing_algorithm",
				Name: "Round Robin",
				ID:   "round_robin",
			},
			{
				Type: "load_balancing_algorithm",
				Name: "Least connections",
				ID:   "least_connections",
			},
		}, nil
	default:
		return nil, nil
	}
}

func (h *Hetzner) Actions() []core.Action {
	return nil
}

func (h *Hetzner) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
