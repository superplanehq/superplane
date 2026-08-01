package dokploy

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const resourceTypeApplication = "application"

func init() {
	registry.RegisterIntegration("dokploy", &Dokploy{})
}

type Dokploy struct{}

type Configuration struct {
	BaseURL  string `json:"baseUrl" mapstructure:"baseUrl"`
	APIToken string `json:"apiToken" mapstructure:"apiToken"`
}

func (d *Dokploy) Name() string {
	return "dokploy"
}

func (d *Dokploy) Label() string {
	return "Dokploy"
}

func (d *Dokploy) Icon() string {
	return "dokploy"
}

func (d *Dokploy) Description() string {
	return "List Dokploy applications and trigger deployments"
}

func (d *Dokploy) Instructions() string {
	return `
**Setup steps:**

1. **Base URL:** Use your Dokploy instance URL (for example ` + "`https://dokploy.example.com`" + `). Dokploy is self-hosted, so this is the address of your own server.
2. **API Key:** In Dokploy, go to **Settings → API/CLI** and generate a new API key, then copy it.
3. The API key inherits the permissions of the user that created it, so SuperPlane only sees the projects and applications that user can access.
`
}

func (d *Dokploy) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "baseUrl",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Dokploy instance URL (e.g. https://dokploy.example.com)",
		},
		{
			Name:        "apiToken",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Dokploy API key (Settings → API/CLI)",
		},
	}
}

func (d *Dokploy) Actions() []core.Action {
	return []core.Action{
		&ListApplications{},
		&DeployApplication{},
	}
}

// Triggers returns nil deliberately. Dokploy's only outbound mechanism is its
// generic webhook notification, whose payload is limited to a title, a
// human-readable message and a timestamp. It carries no event type, no
// resource identifier and no signature, so a trigger built on it could not
// reliably report which application changed or whether it succeeded.
func (d *Dokploy) Triggers() []core.Trigger {
	return nil
}

func (d *Dokploy) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (d *Dokploy) Sync(ctx core.SyncContext) error {
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
		return fmt.Errorf("failed to verify Dokploy credentials: %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (d *Dokploy) HandleRequest(ctx core.HTTPRequestContext) {}

func (d *Dokploy) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != resourceTypeApplication {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	applications, err := client.ListApplications()
	if err != nil {
		return nil, err
	}
	return resourcesFromApplications(applications), nil
}

func (d *Dokploy) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *Dokploy) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}

func resourcesFromApplications(applications []ApplicationSummary) []core.IntegrationResource {
	resources := make([]core.IntegrationResource, 0, len(applications))
	for _, application := range applications {
		id := strings.TrimSpace(application.ApplicationID)
		if id == "" {
			continue
		}
		resources = append(resources, core.IntegrationResource{
			Type: resourceTypeApplication,
			Name: displayName(application),
			ID:   id,
		})
	}
	return resources
}

// displayName qualifies an application with the project and environment it
// lives in. Dokploy only requires application names to be unique within an
// environment, so the bare name is ambiguous in a picker.
func displayName(application ApplicationSummary) string {
	parts := []string{}
	for _, part := range []string{application.ProjectName, application.EnvironmentName, application.Name} {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	if len(parts) == 0 {
		return application.ApplicationID
	}
	return strings.Join(parts, " / ")
}
