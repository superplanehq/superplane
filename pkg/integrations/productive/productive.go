package productive

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const installationInstructions = `
To configure Productive.io to work with SuperPlane:

1. **Get an API token**: In Productive.io, open your profile, go to **API integrations**, and create a personal access token.
2. **Get the organization id**: The organization id is the numeric id in your Productive.io URL, e.g. ` + "`app.productive.io/<organization-id>-name`" + `.
3. **Enter credentials**: Provide the API token and organization id in the integration configuration.
`

func init() {
	registry.RegisterIntegrationWithWebhookHandler("productive", &Productive{}, &ProductiveWebhookHandler{})
}

type Productive struct{}

func (p *Productive) Name() string {
	return "productive"
}

func (p *Productive) Label() string {
	return "Productive"
}

func (p *Productive) Icon() string {
	return "productive"
}

func (p *Productive) Description() string {
	return "Create backlog items from tasks in Productive.io"
}

func (p *Productive) Instructions() string {
	return installationInstructions
}

func (p *Productive) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Personal access token from Productive.io > API integrations",
		},
		{
			Name:        "organizationId",
			Label:       "Organization ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The numeric organization id in your Productive.io URL",
		},
		{
			Name:        "region",
			Label:       "API Base URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: fmt.Sprintf("Override the API base URL. Defaults to %s", BaseURL),
		},
	}
}

func (p *Productive) Actions() []core.Action {
	return []core.Action{}
}

func (p *Productive) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnTask{},
	}
}

func (p *Productive) Sync(ctx core.SyncContext) error {
	config := struct {
		APIToken       string `mapstructure:"apiToken"`
		OrganizationID string `mapstructure:"organizationId"`
	}{}

	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if strings.TrimSpace(config.APIToken) == "" {
		return fmt.Errorf("apiToken is required")
	}

	if strings.TrimSpace(config.OrganizationID) == "" {
		return fmt.Errorf("organizationId is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.ValidateCredentials(); err != nil {
		return fmt.Errorf("invalid credentials: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (p *Productive) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (p *Productive) Hooks() []core.Hook {
	return []core.Hook{}
}

func (p *Productive) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}

func (p *Productive) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op - webhooks are handled by the onTask trigger.
}

func (p *Productive) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeProject:
		return listProjectResources(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func listProjectResources(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	projects, err := client.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(projects))
	for _, project := range projects {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeProject,
			Name: project.Name,
			ID:   project.ID,
		})
	}

	return resources, nil
}
