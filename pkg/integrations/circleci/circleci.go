package circleci

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("circleci", &CircleCI{}, &CircleCIWebhookHandler{})
}

type CircleCI struct{}

type Configuration struct {
	APIToken string `json:"apiToken"`
}

type Metadata struct {
	Projects []string `json:"projects"`
}

func (c *CircleCI) Name() string {
	return "circleci"
}

func (c *CircleCI) Label() string {
	return "CircleCI"
}

func (c *CircleCI) Icon() string {
	return "workflow"
}

func (c *CircleCI) Description() string {
	return "Trigger and monitor CircleCI pipelines"
}

func (c *CircleCI) Instructions() string {
	return "Create a Personal API Token in CircleCI → User Settings → Personal API Tokens"
}

func (c *CircleCI) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "CircleCI Personal API Token",
			Placeholder: "Your CircleCI API token",
			Required:    true,
		},
	}
}

func (c *CircleCI) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (c *CircleCI) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	metadata := Metadata{}
	err = mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Verify the API token by getting current user info
	_, err = client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("error verifying API token: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (c *CircleCI) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (c *CircleCI) Actions() []core.Action {
	return []core.Action{}
}

func (c *CircleCI) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

const ResourceTypePipelineDefinition = "pipeline-definition"

func (c *CircleCI) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeProject:
		return ListProjectSlugs(ctx)
	case ResourceTypeWorkflow:
		return ListWorkflowNames(ctx)
	case ResourceTypePipelineDefinition:
		return c.listPipelineDefinitions(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func (c *CircleCI) listPipelineDefinitions(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	projectID := ctx.Parameters["project_id"]
	if projectID == "" {
		// Try to get project_id from projectSlug if provided
		projectSlug := ctx.Parameters["projectSlug"]
		if projectSlug != "" {
			client, err := NewClient(ctx.HTTP, ctx.Integration)
			if err != nil {
				return nil, fmt.Errorf("failed to create client: %v", err)
			}

			project, err := client.GetProject(projectSlug)
			if err != nil {
				return nil, fmt.Errorf("failed to get project: %v", err)
			}
			projectID = project.ID
		}
	}

	if projectID == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	definitions, err := client.GetPipelineDefinitions(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipeline definitions: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(definitions))
	for _, def := range definitions {
		name := def.Name
		if name == "" {
			name = def.ID
		}
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypePipelineDefinition,
			Name: name,
			ID:   def.ID,
		})
	}

	return resources, nil
}

func (c *CircleCI) Components() []core.Component {
	return []core.Component{
		&RunPipeline{},
		&GetWorkflow{},
		&GetLastWorkflow{},
		&GetRecentWorkflowRuns{},
		&GetTestMetrics{},
		&GetFlakyTests{},
	}
}

func (c *CircleCI) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnWorkflowCompleted{},
	}
}
