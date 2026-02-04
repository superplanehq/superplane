package semaphore

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListPipelines struct{}

type ListPipelinesSpec struct {
	Project string `json:"project"`
	Branch  string `json:"branch,omitempty"`
}

func (l *ListPipelines) Name() string {
	return "semaphore.listPipelines"
}

func (l *ListPipelines) Label() string {
	return "List Pipelines"
}

func (l *ListPipelines) Description() string {
	return "List Semaphore pipelines for a project"
}

func (l *ListPipelines) Documentation() string {
	return `The List Pipelines component fetches a list of pipelines for a specific Semaphore project.

## Configuration

- **Project**: Select the Semaphore project.
- **Branch Filter**: Optional branch name to filter the results.

## Output Channels

- **Done**: Emitted when the list is successfully retrieved, containing an array of pipeline objects.`
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
			Name:  "done",
			Label: "Done",
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
			Name:        "branch",
			Label:       "Branch Filter",
			Type:        configuration.FieldTypeString,
			Description: "Optional branch name to filter pipelines",
		},
	}
}

func (l *ListPipelines) Execute(ctx core.ExecutionContext) error {
	spec := ListPipelinesSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	project, err := client.GetProject(spec.Project)
	if err != nil {
		return fmt.Errorf("error finding project %s: %v", spec.Project, err)
	}

	pipelines, err := client.ListPipelines(project.Metadata.ProjectID)
	if err != nil {
		return fmt.Errorf("error listing pipelines: %v", err)
	}

	// Filter by branch if provided
	if spec.Branch != "" {
		filtered := make([]any, 0)
		for _, p := range pipelines {
			if pipelineMap, ok := p.(map[string]any); ok && pipelineMap["branch_name"] == spec.Branch {
				filtered = append(filtered, p)
			}
		}
		pipelines = filtered
	}

	return ctx.ExecutionState.Emit("done", "semaphore.pipelines.listed", pipelines)
}

func (l *ListPipelines) Setup(ctx core.SetupContext) error                          { return nil }
func (l *ListPipelines) Cancel(ctx core.ExecutionContext) error                     { return nil }
func (l *ListPipelines) Cleanup(ctx core.SetupContext) error                        { return nil }
func (l *ListPipelines) HandleWebhook(ctx core.WebhookRequestContext) (int, error)  { return (200, nil) }
func (l *ListPipelines) Actions() []core.Action                                     { return nil }
func (l *ListPipelines) HandleAction(ctx core.ActionContext) error                  { return nil }
func (l *ListPipelines) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
