package semaphore

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListPipelines struct{}

type ListPipelinesSpec struct {
	Project string `json:"project"`
	Limit   int    `json:"limit"`
}

func (l *ListPipelines) Name() string {
	return "semaphore.listPipelines"
}

func (l *ListPipelines) Label() string {
	return "List Pipelines"
}

func (l *ListPipelines) Description() string {
	return "List pipelines for a project"
}

func (l *ListPipelines) Documentation() string {
	return `The List Pipelines component retrieves a list of pipelines for a specific Semaphore project.

## Use Cases

- **Pipeline Monitoring**: Retrieve a list of recent pipelines to check their status.
- **Reporting**: Generate reports on pipeline activity.
- **Workflow Automation**: Use the list of pipelines to trigger downstream actions based on the state of specific pipelines.

## Configuration

- **Project**: Select the Semaphore project to list pipelines for.
- **Limit**: Optional. The maximum number of pipelines to return (default is 10, max is 100).

## Outputs

- **Output**: Emits a list of pipelines, each containing details like ID, name, state, and creation time.`
}

func (l *ListPipelines) Icon() string {
	return "list"
}

func (l *ListPipelines) Color() string {
	return "gray"
}

func (l *ListPipelines) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  "output",
			Label: "Output",
		},
	}
}

func (l *ListPipelines) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "project",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "limit",
			Label:       "Limit",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     10,
			Description: "Maximum number of pipelines to return (max 100)",
		},
	}
}

func (l *ListPipelines) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (l *ListPipelines) Setup(ctx core.SetupContext) error {
	spec := ListPipelinesSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	if spec.Project == "" {
		return fmt.Errorf("project is required")
	}

	return nil
}

func (l *ListPipelines) ExampleOutput() map[string]any {
	return map[string]any{
		"output": []any{
			map[string]any{
				"id":   "123",
				"name": "master",
			},
		},
	}
}

func (l *ListPipelines) Execute(ctx core.ExecutionContext) error {
	spec := ListPipelinesSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	if spec.Limit <= 0 {
		spec.Limit = 10
	}
	if spec.Limit > 100 {
		spec.Limit = 100
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	project, err := client.GetProject(spec.Project)
	if err != nil {
		return fmt.Errorf("error finding project %s: %v", spec.Project, err)
	}
	if project == nil {
		return fmt.Errorf("project %s not found", spec.Project)
	}

	pipelines, err := client.ListPipelines(project.Metadata.ProjectID)
	if err != nil {
		return fmt.Errorf("error listing pipelines: %v", err)
	}

	// Apply limit manually if the API doesn't support it directly or to ensure correctness
	if len(pipelines) > spec.Limit {
		pipelines = pipelines[:spec.Limit]
	}

	return ctx.ExecutionState.Emit("output", "semaphore.pipelines.list", []any{pipelines})
}

func (l *ListPipelines) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (l *ListPipelines) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (l *ListPipelines) Actions() []core.Action {
	return []core.Action{}
}

func (l *ListPipelines) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (l *ListPipelines) Cleanup(ctx core.SetupContext) error {
	return nil
}
