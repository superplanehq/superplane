package circleci

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetLastWorkflow struct{}

type GetLastWorkflowSpec struct {
	ProjectSlug string `json:"projectSlug" mapstructure:"projectSlug"`
	Branch      string `json:"branch" mapstructure:"branch"`
}

type GetLastWorkflowNodeMetadata struct {
	ProjectID   string `json:"projectId" mapstructure:"projectId"`
	ProjectSlug string `json:"projectSlug" mapstructure:"projectSlug"`
	ProjectName string `json:"projectName" mapstructure:"projectName"`
}

func (c *GetLastWorkflow) Name() string {
	return "circleci.getLastWorkflow"
}

func (c *GetLastWorkflow) Label() string {
	return "Get Last Workflow"
}

func (c *GetLastWorkflow) Description() string {
	return "Get the most recent workflow for a project, with optional branch filter"
}

func (c *GetLastWorkflow) Documentation() string {
	return `The Get Last Workflow component retrieves the most recent workflow for a CircleCI project.

## Use Cases

- **Latest build status**: Check if the latest workflow on a branch passed or failed
- **Deployment gates**: Verify the most recent CI run succeeded before deploying
- **Branch monitoring**: Monitor the latest workflow status on a specific branch
- **Workflow chaining**: Use the latest workflow result to make routing decisions

## How It Works

1. Fetches the most recent pipeline for the project (optionally filtered by branch)
2. Retrieves workflows from that pipeline
3. Returns the first (most recent) workflow with its details

## Configuration

- **Project Slug**: CircleCI project slug (e.g., gh/org/repo)
- **Branch**: Optional branch filter (leave empty for all branches)

## Output

Emits the most recent workflow including:
- Workflow ID, name, and status
- Pipeline information (ID, number, created at)
- Start and stop times`
}

func (c *GetLastWorkflow) Icon() string {
	return "workflow"
}

func (c *GetLastWorkflow) Color() string {
	return "gray"
}

func (c *GetLastWorkflow) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetLastWorkflow) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectSlug",
			Label:       "Project slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "CircleCI project slug. Find in CircleCI project settings.",
		},
		{
			Name:        "branch",
			Label:       "Branch",
			Type:        configuration.FieldTypeString,
			Description: "Optional branch filter. Leave empty for all branches.",
		},
	}
}

func (c *GetLastWorkflow) Setup(ctx core.SetupContext) error {
	spec := GetLastWorkflowSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.ProjectSlug) == "" {
		return fmt.Errorf("project slug is required")
	}

	metadata := GetLastWorkflowNodeMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	projectChanged := metadata.ProjectSlug != spec.ProjectSlug
	if projectChanged {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		project, err := client.GetProject(spec.ProjectSlug)
		if err != nil {
			return fmt.Errorf("project not found or inaccessible: %w", err)
		}

		err = ctx.Metadata.Set(GetLastWorkflowNodeMetadata{
			ProjectID:   project.ID,
			ProjectSlug: project.Slug,
			ProjectName: project.Name,
		})
		if err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}
	}

	return nil
}

func (c *GetLastWorkflow) Execute(ctx core.ExecutionContext) error {
	spec := GetLastWorkflowSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	pipelines, err := client.ListProjectPipelines(spec.ProjectSlug, strings.TrimSpace(spec.Branch))
	if err != nil {
		return fmt.Errorf("failed to list project pipelines: %w", err)
	}

	if len(pipelines) == 0 {
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"circleci.workflow.last",
			[]any{map[string]any{
				"workflow": nil,
				"pipeline": nil,
			}},
		)
	}

	// Sort pipelines by created_at descending to find the most recent one,
	// rather than assuming the API returns them in order.
	sort.Slice(pipelines, func(i, j int) bool {
		return pipelines[i].CreatedAt > pipelines[j].CreatedAt
	})

	latestPipeline := pipelines[0]
	workflows, err := client.GetPipelineWorkflows(latestPipeline.ID)
	if err != nil {
		return fmt.Errorf("failed to get pipeline workflows: %w", err)
	}

	// Sort workflows by created_at descending to find the most recent one.
	sort.Slice(workflows, func(i, j int) bool {
		return workflows[i].CreatedAt > workflows[j].CreatedAt
	})

	var latestWorkflow *WorkflowResponse
	if len(workflows) > 0 {
		latestWorkflow = &workflows[0]
	}

	output := map[string]any{
		"workflow": latestWorkflow,
		"pipeline": map[string]any{
			"id":         latestPipeline.ID,
			"number":     latestPipeline.Number,
			"state":      latestPipeline.State,
			"created_at": latestPipeline.CreatedAt,
		},
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"circleci.workflow.last",
		[]any{output},
	)
}

func (c *GetLastWorkflow) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetLastWorkflow) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetLastWorkflow) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetLastWorkflow) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetLastWorkflow) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetLastWorkflow) Cleanup(ctx core.SetupContext) error {
	return nil
}
