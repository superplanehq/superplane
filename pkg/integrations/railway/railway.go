package railway

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("railway", &Railway{}, &RailwayWebhookHandler{})
}

type Railway struct{}

type Configuration struct {
	APIToken string `json:"apiToken" mapstructure:"apiToken"`
}

type Metadata struct {
	Projects []ProjectResource `json:"projects,omitempty" mapstructure:"projects,omitempty"`
}

type ProjectResource struct {
	ID   string `json:"id"   mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
}

func (r *Railway) Name() string {
	return "railway"
}

func (r *Railway) Label() string {
	return "Railway"
}

func (r *Railway) Icon() string {
	return "railway"
}

func (r *Railway) Description() string {
	return "Deploy and monitor Railway applications"
}

func (r *Railway) Instructions() string {
	return `## Connect Railway

### Creating an API Token

1. Go to [Railway Account Settings → Tokens](https://railway.app/account/tokens)
2. Click **"Create Token"** to generate a new API token
3. Give your token a descriptive name (e.g., "SuperPlane Integration")


### Token Scoping

Railway offers different token scoping options:

| Scope | Access Level | Use Case |
|-------|--------------|----------|
| **No Workspace** | All workspaces and projects | Recommended for SuperPlane |
| **Project-specific** | Projects in a specific workspace only | Limited to one workspace |

**Recommendation:** Select **"No Workspace"** when creating your token. This allows SuperPlane to access all your projects across workspaces. If you scope the token to a specific workspace, only projects in that workspace will be available.

### Permissions

The API token allows SuperPlane to:
- List your projects, services, and environments
- Trigger deployments on your services
- Receive deployment status webhooks

### Webhook Configuration

For the **On Deployment Event** trigger, you'll need to manually configure webhooks in Railway:

1. Create the trigger in SuperPlane and save the canvas
2. Copy the generated webhook URL from the trigger settings
3. Go to your Railway project → Settings → Webhooks
4. Add the webhook URL and select "Deploy" events

### Troubleshooting

- **"Not Authorized" error**: Your token may be scoped to a workspace that doesn't include the project you're trying to access. Create a new token with "No Scoping" selected.
- **Projects not showing**: Try disconnecting and reconnecting the integration to refresh the project list.`
}

func (r *Railway) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Required:    true,
			Description: "Create a token at railway.app/account/tokens. Use 'No Workspace' to access all projects.",
			Placeholder: "YOUR RAILWAY API TOKEN",
		},
	}
}

func (r *Railway) Components() []core.Component {
	return []core.Component{
		&TriggerDeploy{},
	}
}

func (r *Railway) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnDeploymentEvent{},
	}
}

func (r *Railway) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.APIToken == "" {
		return fmt.Errorf("API token is required")
	}

	// Validate API token by making a test request
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	// Fetch and cache projects (the 'me' query requires additional permissions)
	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to validate API token: %w", err)
	}

	// Store projects in metadata for ListResources
	projectResources := make([]ProjectResource, 0, len(projects))
	for _, p := range projects {
		projectResources = append(projectResources, ProjectResource{
			ID:   p.ID,
			Name: p.Name,
		})
	}

	ctx.Integration.SetMetadata(Metadata{
		Projects: projectResources,
	})

	ctx.Integration.Ready()
	return nil
}

func (r *Railway) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (r *Railway) HandleRequest(ctx core.HTTPRequestContext) {
	// No OAuth or special HTTP handling needed
}

func (r *Railway) Actions() []core.Action {
	return []core.Action{}
}

func (r *Railway) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (r *Railway) ListResources(
	resourceType string,
	ctx core.ListResourcesContext,
) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "project":
		return r.listProjectsFromMetadata(ctx)
	case "service":
		projectID := ctx.Parameters["projectId"]
		if projectID == "" {
			return []core.IntegrationResource{}, nil
		}
		return r.listServices(ctx, projectID)
	case "environment":
		projectID := ctx.Parameters["projectId"]
		if projectID == "" {
			return []core.IntegrationResource{}, nil
		}
		return r.listEnvironments(ctx, projectID)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func (r *Railway) listProjectsFromMetadata(
	ctx core.ListResourcesContext,
) ([]core.IntegrationResource, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(metadata.Projects))
	for _, project := range metadata.Projects {
		resources = append(resources, core.IntegrationResource{
			Type: "project",
			ID:   project.ID,
			Name: project.Name,
		})
	}

	return resources, nil
}

func (r *Railway) listServices(
	ctx core.ListResourcesContext,
	projectID string,
) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	project, err := client.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(project.Services))
	for _, service := range project.Services {
		resources = append(resources, core.IntegrationResource{
			Type: "service",
			ID:   service.ID,
			Name: service.Name,
		})
	}

	return resources, nil
}

func (r *Railway) listEnvironments(
	ctx core.ListResourcesContext,
	projectID string,
) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	project, err := client.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(project.Environments))
	for _, env := range project.Environments {
		resources = append(resources, core.IntegrationResource{
			Type: "environment",
			ID:   env.ID,
			Name: env.Name,
		})
	}

	return resources, nil
}
