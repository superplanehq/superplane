package launchdarkly

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("launchdarkly", &LaunchDarkly{}, &LaunchDarklyWebhookHandler{})
}

type LaunchDarkly struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
}

func (l *LaunchDarkly) Name() string {
	return "launchdarkly"
}

func (l *LaunchDarkly) Label() string {
	return "LaunchDarkly"
}

func (l *LaunchDarkly) Icon() string {
	return "launchdarkly"
}

func (l *LaunchDarkly) Description() string {
	return "Manage feature flags and react to flag changes in LaunchDarkly"
}

func (l *LaunchDarkly) Instructions() string {
	return `## API integration

1. In the [LaunchDarkly Account settings > Authorization](https://app.launchdarkly.com/settings/authorization), click **Create token**.
2. Give the token a name and select a role with at least **Reader** permissions for feature flags.
   - For the **Delete Feature Flag** action, the role must also include **Writer** permissions.
3. Create the token and **paste the API access token** in the Configuration section below.`
}

func (l *LaunchDarkly) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Access Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "API access token from LaunchDarkly. Create one in Account settings > Authorization with appropriate role permissions.",
		},
	}
}

func (l *LaunchDarkly) Components() []core.Component {
	return []core.Component{
		&GetFeatureFlag{},
		&DeleteFeatureFlag{},
	}
}

func (l *LaunchDarkly) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnFeatureFlagChange{},
	}
}

func (l *LaunchDarkly) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (l *LaunchDarkly) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	if strings.TrimSpace(config.APIKey) == "" {
		return fmt.Errorf("API access token is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	// Validate API key by listing projects
	_, err = client.ListProjects()
	if err != nil {
		return fmt.Errorf("error validating API access token (listing projects): %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (l *LaunchDarkly) HandleRequest(ctx core.HTTPRequestContext) {}

func (l *LaunchDarkly) Actions() []core.Action {
	return []core.Action{}
}

func (l *LaunchDarkly) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (l *LaunchDarkly) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "project":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		projects, err := client.ListProjects()
		if err != nil {
			return nil, fmt.Errorf("failed to list projects: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(projects))
		for _, p := range projects {
			resources = append(resources, core.IntegrationResource{
				Type: "project",
				Name: p.Name,
				ID:   p.Key,
			})
		}
		return resources, nil

	case "environment":
		projectKey := ctx.Parameters["projectKey"]
		if projectKey == "" {
			return []core.IntegrationResource{}, nil
		}

		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		environments, err := client.ListEnvironments(projectKey)
		if err != nil {
			return nil, fmt.Errorf("failed to list environments: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(environments))
		for _, e := range environments {
			resources = append(resources, core.IntegrationResource{
				Type: "environment",
				Name: e.Name,
				ID:   e.Key,
			})
		}
		return resources, nil

	case "flag":
		projectKey := ctx.Parameters["projectKey"]
		if projectKey == "" {
			return []core.IntegrationResource{}, nil
		}

		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		flags, err := client.ListFeatureFlags(projectKey)
		if err != nil {
			return nil, fmt.Errorf("failed to list feature flags: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(flags))
		for _, f := range flags {
			resources = append(resources, core.IntegrationResource{
				Type: "flag",
				Name: f.Name,
				ID:   f.Key,
			})
		}
		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}
