package octopus

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("octopus", &Octopus{}, &OctopusWebhookHandler{})
}

type Octopus struct{}

type Configuration struct {
	ServerURL string `json:"serverUrl" mapstructure:"serverUrl"`
	APIKey    string `json:"apiKey" mapstructure:"apiKey"`
	Space     string `json:"space" mapstructure:"space"`
}

type Metadata struct {
	Space *SpaceMetadata `json:"space,omitempty" mapstructure:"space"`
}

type SpaceMetadata struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
}

func (o *Octopus) Name() string {
	return "octopus"
}

func (o *Octopus) Label() string {
	return "Octopus Deploy"
}

func (o *Octopus) Icon() string {
	return "rocket"
}

func (o *Octopus) Description() string {
	return "Deploy releases and react to deployment events in Octopus Deploy"
}

func (o *Octopus) Instructions() string {
	return `
1. **Server URL:** Your Octopus Deploy instance URL (e.g. ` + "`https://my-company.octopus.app`" + `).
2. **API Key:** Create one in **Octopus Web Portal → Profile → My API Keys → New API Key**.
   - Enter a purpose (e.g., "SuperPlane Integration") and click **Generate New**.
   - **Copy the key immediately**—it cannot be viewed again.
3. **Space:** Select the Octopus Deploy space to use. Leave empty to use the default space.
4. **Auth:** SuperPlane sends requests using the ` + "`X-Octopus-ApiKey`" + ` header.
5. **Webhooks:** SuperPlane creates Octopus subscriptions automatically to receive deployment events. No manual setup is required.`
}

func (o *Octopus) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "serverUrl",
			Label:       "Server URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "https://my-company.octopus.app",
			Description: "Octopus Deploy instance URL",
		},
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Octopus Deploy API key",
		},
		{
			Name:        "space",
			Label:       "Space",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "Spaces-1",
			Description: "Octopus Deploy space name or ID (e.g. 'Spaces-1' or 'Default'). Leave empty to use the default space.",
		},
	}
}

func (o *Octopus) Components() []core.Component {
	return []core.Component{
		&DeployRelease{},
	}
}

func (o *Octopus) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnDeploymentEvent{},
	}
}

func (o *Octopus) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (o *Octopus) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ServerURL == "" {
		return fmt.Errorf("serverUrl is required")
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.ValidateCredentials(); err != nil {
		return fmt.Errorf("failed to verify Octopus Deploy credentials: %w", err)
	}

	space, err := resolveSpace(client, config.Space)
	if err != nil {
		return fmt.Errorf("failed to resolve space: %w", err)
	}

	ctx.Integration.SetMetadata(Metadata{
		Space: &SpaceMetadata{
			ID:   space.ID,
			Name: space.Name,
		},
	})
	ctx.Integration.Ready()
	return nil
}

func (o *Octopus) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (o *Octopus) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	switch resourceType {
	case "space":
		return listSpaceResources(client)
	case "project":
		return listProjectResources(client, ctx.Integration)
	case "environment":
		return listEnvironmentResources(client, ctx.Integration)
	case "release":
		return listReleaseResources(client, ctx.Integration, ctx.Parameters)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func (o *Octopus) Actions() []core.Action {
	return []core.Action{}
}

func (o *Octopus) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func listSpaceResources(client *Client) ([]core.IntegrationResource, error) {
	spaces, err := client.ListSpaces()
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(spaces))
	for _, space := range spaces {
		if space.ID == "" || space.Name == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{Type: "space", Name: space.Name, ID: space.ID})
	}

	return resources, nil
}

func listProjectResources(client *Client, integration core.IntegrationContext) ([]core.IntegrationResource, error) {
	spaceID, err := spaceIDForIntegration(client, integration)
	if err != nil {
		return nil, err
	}

	projects, err := client.ListProjects(spaceID)
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(projects))
	for _, project := range projects {
		if project.ID == "" || project.Name == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{Type: "project", Name: project.Name, ID: project.ID})
	}

	return resources, nil
}

func listEnvironmentResources(client *Client, integration core.IntegrationContext) ([]core.IntegrationResource, error) {
	spaceID, err := spaceIDForIntegration(client, integration)
	if err != nil {
		return nil, err
	}

	environments, err := client.ListEnvironments(spaceID)
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(environments))
	for _, env := range environments {
		if env.ID == "" || env.Name == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{Type: "environment", Name: env.Name, ID: env.ID})
	}

	return resources, nil
}

func listReleaseResources(client *Client, integration core.IntegrationContext, parameters map[string]string) ([]core.IntegrationResource, error) {
	spaceID, err := spaceIDForIntegration(client, integration)
	if err != nil {
		return nil, err
	}

	projectID := parameters["project"]
	if projectID == "" {
		return []core.IntegrationResource{}, nil
	}

	releases, err := client.ListReleasesForProject(spaceID, projectID)
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(releases))
	for _, release := range releases {
		if release.ID == "" || release.Version == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{Type: "release", Name: release.Version, ID: release.ID})
	}

	return resources, nil
}

func resolveSpace(client *Client, requestedSpace string) (Space, error) {
	spaces, err := client.ListSpaces()
	if err != nil {
		return Space{}, err
	}

	if len(spaces) == 0 {
		return Space{}, fmt.Errorf("no spaces available for this API key")
	}

	if requestedSpace == "" {
		// Return the default space if available, otherwise the first one
		for _, space := range spaces {
			if space.IsDefault {
				return space, nil
			}
		}
		return spaces[0], nil
	}

	for _, space := range spaces {
		if space.ID == requestedSpace || space.Name == requestedSpace {
			return space, nil
		}
	}

	return Space{}, fmt.Errorf("space %q is not accessible with this API key", requestedSpace)
}

func spaceIDForIntegration(client *Client, integration core.IntegrationContext) (string, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err == nil && metadata.Space != nil && metadata.Space.ID != "" {
		return metadata.Space.ID, nil
	}

	spaceValue := ""
	spaceConfig, err := integration.GetConfig("space")
	if err == nil {
		spaceValue = string(spaceConfig)
	}

	space, err := resolveSpace(client, spaceValue)
	if err != nil {
		return "", err
	}

	integration.SetMetadata(Metadata{
		Space: &SpaceMetadata{
			ID:   space.ID,
			Name: space.Name,
		},
	})

	return space.ID, nil
}
