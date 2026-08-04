package posthog

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("posthog", &PostHog{}, &PostHogWebhookHandler{})
}

type PostHog struct{}

type Configuration struct {
	Host   string `json:"host" mapstructure:"host"`
	APIKey string `json:"apiKey" mapstructure:"apiKey"`
}

const installationInstructions = `
SuperPlane connects to PostHog with a personal API key.

1. In PostHog, open [Settings > Personal API keys](https://us.posthog.com/settings/user-api-keys) and click **Create personal API key**.
2. Give the key a name and scope it to the projects SuperPlane should access.
3. Grant these scopes:
   - **Project**: Read — to list the projects you can pick from.
   - **Event Definition**: Read — to suggest event names on the On Event trigger.
   - **Hog Function**: Write — to create and remove the webhook destination used by the On Event trigger.
   - **Query**: Read — to run the Run Query action.
4. Copy the key (it is shown only once) and paste it below.
5. Set **Host** to the region your project lives in: ` + "`https://us.posthog.com`" + ` or ` + "`https://eu.posthog.com`" + `, or your own address if you self-host.
`

func (p *PostHog) Name() string {
	return "posthog"
}

func (p *PostHog) Label() string {
	return "PostHog"
}

func (p *PostHog) Icon() string {
	return "posthog"
}

func (p *PostHog) Description() string {
	return "React to product analytics events and query data in PostHog"
}

func (p *PostHog) Instructions() string {
	return installationInstructions
}

func (p *PostHog) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "host",
			Label:       "Host",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     DefaultHost,
			Description: "PostHog instance URL. Use https://us.posthog.com or https://eu.posthog.com for PostHog Cloud, or your own address if you self-host.",
		},
		{
			Name:        "apiKey",
			Label:       "Personal API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Personal API key from PostHog. Create one in Settings > Personal API keys with the Project, Event Definition, Hog Function, and Query scopes.",
		},
	}
}

func (p *PostHog) Actions() []core.Action {
	return []core.Action{
		&RunQuery{},
	}
}

func (p *PostHog) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnEvent{},
	}
}

func (p *PostHog) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	if strings.TrimSpace(config.APIKey) == "" {
		return fmt.Errorf("personal API key is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	//
	// Listing projects both validates the key and confirms it carries the
	// project scope every trigger and action depends on.
	//
	if _, err := client.ListProjects(); err != nil {
		return fmt.Errorf("error validating personal API key (listing projects): %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (p *PostHog) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (p *PostHog) Hooks() []core.Hook {
	return []core.Hook{}
}

func (p *PostHog) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}

func (p *PostHog) HandleRequest(ctx core.HTTPRequestContext) {}

func (p *PostHog) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	switch resourceType {
	case "project":
		projects, err := client.ListProjects()
		if err != nil {
			return nil, fmt.Errorf("failed to list projects: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(projects))
		for _, project := range projects {
			resources = append(resources, core.IntegrationResource{
				Type: "project",
				Name: project.Name,
				ID:   strconv.Itoa(project.ID),
			})
		}

		return resources, nil

	case "event":
		//
		// Event names are scoped to a project, so without one there is nothing
		// to suggest yet - the user picks the project first.
		//
		projectID := ctx.Parameters["projectId"]
		if projectID == "" {
			return []core.IntegrationResource{}, nil
		}

		definitions, err := client.ListEventDefinitions(projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to list event definitions: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(definitions))
		for _, definition := range definitions {
			resources = append(resources, core.IntegrationResource{
				Type: "event",
				Name: definition.Name,
				ID:   definition.Name,
			})
		}

		return resources, nil

	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
}
