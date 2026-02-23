package circleci

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetRecentWorkflowRuns struct{}

type GetRecentWorkflowRunsSpec struct {
	ProjectSlug string `json:"projectSlug" mapstructure:"projectSlug"`
}

type GetRecentWorkflowRunsNodeMetadata struct {
	ProjectID   string `json:"projectId" mapstructure:"projectId"`
	ProjectSlug string `json:"projectSlug" mapstructure:"projectSlug"`
	ProjectName string `json:"projectName" mapstructure:"projectName"`
}

func (c *GetRecentWorkflowRuns) Name() string {
	return "circleci.getRecentWorkflowRuns"
}

func (c *GetRecentWorkflowRuns) Label() string {
	return "Get Recent Workflow Runs"
}

func (c *GetRecentWorkflowRuns) Description() string {
	return "Retrieve aggregated workflow run data including success rate, throughput, and duration metrics"
}

func (c *GetRecentWorkflowRuns) Documentation() string {
	return `The Get Recent Workflow Runs component retrieves aggregated workflow metrics from CircleCI Insights.

## Use Cases

- **CI/CD health monitoring**: Track success rates, throughput, and duration trends
- **Performance tracking**: Monitor workflow duration over time
- **Reliability insights**: Identify workflow reliability patterns
- **Reporting**: Collect CI/CD metrics for dashboards and reports

## Configuration

- **Project Slug**: CircleCI project slug (e.g., gh/org/repo)

## Output

Emits aggregated workflow run data including:
- Workflow names and their metrics
- Success rates and throughput
- Duration statistics (mean, median, p95)
- Time window for the aggregation period`
}

func (c *GetRecentWorkflowRuns) Icon() string {
	return "bar-chart"
}

func (c *GetRecentWorkflowRuns) Color() string {
	return "gray"
}

func (c *GetRecentWorkflowRuns) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetRecentWorkflowRuns) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectSlug",
			Label:       "Project slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "CircleCI project slug. Find in CircleCI project settings.",
		},
	}
}

func (c *GetRecentWorkflowRuns) Setup(ctx core.SetupContext) error {
	spec := GetRecentWorkflowRunsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.ProjectSlug) == "" {
		return fmt.Errorf("project slug is required")
	}

	metadata := GetRecentWorkflowRunsNodeMetadata{}
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

		err = ctx.Metadata.Set(GetRecentWorkflowRunsNodeMetadata{
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

func (c *GetRecentWorkflowRuns) Execute(ctx core.ExecutionContext) error {
	spec := GetRecentWorkflowRunsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	workflows, err := client.GetInsightsWorkflows(spec.ProjectSlug)
	if err != nil {
		return fmt.Errorf("failed to get insights workflows: %w", err)
	}

	output := map[string]any{
		"workflows": workflows,
		"total":     len(workflows),
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"circleci.insights.workflows",
		[]any{output},
	)
}

func (c *GetRecentWorkflowRuns) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetRecentWorkflowRuns) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetRecentWorkflowRuns) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetRecentWorkflowRuns) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetRecentWorkflowRuns) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetRecentWorkflowRuns) Cleanup(ctx core.SetupContext) error {
	return nil
}
