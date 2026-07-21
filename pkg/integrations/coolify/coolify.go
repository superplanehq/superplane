package coolify

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	resourceTypeApplication = "application"
	resourceTypeService     = "service"
)

func init() {
	registry.RegisterIntegration("coolify", &Coolify{})
}

type Coolify struct{}

type Configuration struct {
	BaseURL  string `json:"baseUrl" mapstructure:"baseUrl"`
	APIToken string `json:"apiToken" mapstructure:"apiToken"`
}

func (c *Coolify) Name() string {
	return "coolify"
}

func (c *Coolify) Label() string {
	return "Coolify"
}

func (c *Coolify) Icon() string {
	return "coolify"
}

func (c *Coolify) CustomTools() []core.CustomIntegrationTool {
	return []core.CustomIntegrationTool{}
}

func (c *Coolify) Description() string {
	return "List and control Coolify applications and services, and trigger deployments"
}

func (c *Coolify) Instructions() string {
	return `
**Setup steps:**

1. **Base URL:** Use your Coolify instance URL (for example ` + "`https://coolify.example.com`" + ` for self-hosted, or ` + "`https://app.coolify.io`" + ` for Coolify Cloud).
2. **API Token:** In Coolify, go to **Keys & Tokens → API tokens**, click **Create new token**, give it a name, select the required permissions (read + write), and copy the generated token.
3. The API token is **team-scoped** — SuperPlane will see all applications and services available to that team.
`
}

func (c *Coolify) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "baseUrl",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Coolify instance URL (e.g. https://coolify.example.com or https://app.coolify.io)",
		},
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Coolify API token (Keys & Tokens → API tokens) with read and write permissions",
		},
	}
}

func (c *Coolify) Actions() []core.Action {
	return []core.Action{
		&ListApplications{},
		&ListServices{},
		&ControlApplication{},
		&ControlService{},
		&DeployApplication{},
	}
}

func (c *Coolify) Triggers() []core.Trigger {
	return nil
}

func (c *Coolify) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (c *Coolify) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.BaseURL) == "" {
		return fmt.Errorf("baseUrl is required")
	}
	if strings.TrimSpace(config.APIToken) == "" {
		return fmt.Errorf("apiToken is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return fmt.Errorf("failed to verify Coolify credentials: %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (c *Coolify) HandleRequest(ctx core.HTTPRequestContext) {}

func (c *Coolify) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	switch resourceType {
	case resourceTypeApplication:
		applications, err := client.ListApplications()
		if err != nil {
			return nil, err
		}
		return resourcesFromApplications(applications), nil
	case resourceTypeService:
		services, err := client.ListServices()
		if err != nil {
			return nil, err
		}
		return resourcesFromServices(services), nil
	default:
		return []core.IntegrationResource{}, nil
	}
}

func (c *Coolify) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *Coolify) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}

func resourcesFromApplications(applications []Application) []core.IntegrationResource {
	resources := make([]core.IntegrationResource, 0, len(applications))
	for _, app := range applications {
		id := strings.TrimSpace(app.UUID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(app.Name)
		if name == "" {
			name = id
		}
		resources = append(resources, core.IntegrationResource{
			Type: resourceTypeApplication,
			Name: name,
			ID:   id,
		})
	}
	return resources
}

func resourcesFromServices(services []Service) []core.IntegrationResource {
	resources := make([]core.IntegrationResource, 0, len(services))
	for _, svc := range services {
		id := strings.TrimSpace(svc.UUID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(svc.Name)
		if name == "" {
			name = id
		}
		resources = append(resources, core.IntegrationResource{
			Type: resourceTypeService,
			Name: name,
			ID:   id,
		})
	}
	return resources
}
