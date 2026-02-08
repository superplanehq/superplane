package dockerhub

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const installationInstructions = `
To connect Docker Hub to SuperPlane:

1. **Create a Docker Hub access token**: Docker Hub → Account Settings → Security → New Access Token
2. **Copy the token** and store it securely (you will not see it again)
3. **Enter your Docker Hub username** and access token below

**Scopes**:
- **Read**: Required to list repositories and read tags
`

func init() {
	registry.RegisterIntegration("dockerhub", &DockerHub{})
}

type DockerHub struct{}

type Configuration struct {
	Username    string `json:"username"`
	AccessToken string `json:"accessToken"`
}

func (d *DockerHub) Name() string {
	return "dockerhub"
}

func (d *DockerHub) Label() string {
	return "Docker Hub"
}

func (d *DockerHub) Icon() string {
	return "docker"
}

func (d *DockerHub) Description() string {
	return "Manage and react to Docker Hub repositories and tags"
}

func (d *DockerHub) Instructions() string {
	return installationInstructions
}

func (d *DockerHub) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "username",
			Label:       "Username",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Docker Hub username or organization name",
		},
		{
			Name:        "accessToken",
			Label:       "Access Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Docker Hub personal access token",
		},
	}
}

func (d *DockerHub) Components() []core.Component {
	return []core.Component{
		&GetImageTag{},
	}
}

func (d *DockerHub) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnImagePush{},
	}
}

func (d *DockerHub) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (d *DockerHub) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := refreshAccessToken(ctx.HTTP, ctx.Integration, config); err != nil {
		return fmt.Errorf("failed to refresh access token: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Docker Hub client: %w", err)
	}

	if err := client.ValidateCredentials(config.Username); err != nil {
		return fmt.Errorf("failed to validate Docker Hub credentials: %w", err)
	}

	if err := ctx.Integration.ScheduleActionCall("refreshAccessToken", map[string]any{}, accessTokenRefreshInterval); err != nil {
		return fmt.Errorf("failed to schedule token refresh: %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (d *DockerHub) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op - webhooks are handled by triggers
}

func (d *DockerHub) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return listDockerHubResources(resourceType, ctx)
}

func (d *DockerHub) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "refreshAccessToken",
			Description: "Refresh Docker Hub access token",
		},
	}
}

func (d *DockerHub) HandleAction(ctx core.IntegrationActionContext) error {
	switch ctx.Name {
	case "refreshAccessToken":
		config := Configuration{}
		if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
			return fmt.Errorf("failed to decode configuration: %w", err)
		}

		if err := refreshAccessToken(ctx.HTTP, ctx.Integration, config); err != nil {
			return fmt.Errorf("failed to refresh access token: %w", err)
		}

		return ctx.Integration.ScheduleActionCall("refreshAccessToken", map[string]any{}, accessTokenRefreshInterval)

	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}
