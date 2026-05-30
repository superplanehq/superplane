package railway

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithOptions("railway", &Railway{}, registry.IntegrationRegistrationOptions{
		WebhookHandler: &RailwayWebhookHandler{},
		SetupProvider:  &SetupProvider{},
	})
}

type Railway struct{}

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
	return "Deploy services and react to deployment events on Railway"
}

func (r *Railway) Instructions() string {
	return ""
}

func (r *Railway) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Railway Workspace API Token",
			Required:    true,
		},
	}
}

func (r *Railway) Actions() []core.Action {
	return []core.Action{
		&TriggerDeploy{},
	}
}

func (r *Railway) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnDeploymentEvent{},
	}
}

func (r *Railway) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (r *Railway) Sync(ctx core.SyncContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return fmt.Errorf("failed to verify Railway credentials: %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (r *Railway) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (r *Railway) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	switch resourceType {
	case "project":
		return r.listProjects(client)
	case "service":
		return r.listServices(client, ctx.Parameters["project"])
	case "environment":
		return r.listEnvironments(client, ctx.Parameters["project"])
	default:
		return []core.IntegrationResource{}, nil
	}
}

func (r *Railway) listProjects(client *Client) ([]core.IntegrationResource, error) {
	workspaces, err := client.ListWorkspaces()
	if err != nil {
		return nil, err
	}

	var resources []core.IntegrationResource
	for _, workspace := range workspaces {
		projects, err := client.ListProjects(workspace.ID)
		if err != nil {
			return nil, err
		}

		for _, p := range projects {
			resources = append(resources, core.IntegrationResource{
				Type: "project",
				ID:   p.ID,
				Name: p.Name,
			})
		}
	}

	return resources, nil
}

func (r *Railway) listServices(client *Client, projectID string) ([]core.IntegrationResource, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" || strings.Contains(projectID, "{{") {
		return []core.IntegrationResource{}, nil
	}

	project, err := client.GetProjectDetails(projectID)
	if err != nil {
		return nil, err
	}

	var resources []core.IntegrationResource
	for _, edge := range project.Services.Edges {
		resources = append(resources, core.IntegrationResource{
			Type: "service",
			ID:   edge.Node.ID,
			Name: edge.Node.Name,
		})
	}

	return resources, nil
}

func (r *Railway) listEnvironments(client *Client, projectID string) ([]core.IntegrationResource, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" || strings.Contains(projectID, "{{") {
		return []core.IntegrationResource{}, nil
	}

	project, err := client.GetProjectDetails(projectID)
	if err != nil {
		return nil, err
	}

	var resources []core.IntegrationResource
	for _, edge := range project.Environments.Edges {
		resources = append(resources, core.IntegrationResource{
			Type: "environment",
			ID:   edge.Node.ID,
			Name: edge.Node.Name,
		})
	}

	return resources, nil
}

func (r *Railway) Hooks() []core.Hook {
	return []core.Hook{}
}

func (r *Railway) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
