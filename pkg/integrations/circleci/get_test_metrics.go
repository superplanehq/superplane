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

type GetTestMetrics struct{}

type GetTestMetricsSpec struct {
	ProjectSlug  string `json:"projectSlug" mapstructure:"projectSlug"`
	WorkflowName string `json:"workflowName" mapstructure:"workflowName"`
}

type GetTestMetricsNodeMetadata struct {
	ProjectID   string `json:"projectId" mapstructure:"projectId"`
	ProjectSlug string `json:"projectSlug" mapstructure:"projectSlug"`
	ProjectName string `json:"projectName" mapstructure:"projectName"`
}

func (c *GetTestMetrics) Name() string {
	return "circleci.getTestMetrics"
}

func (c *GetTestMetrics) Label() string {
	return "Get Test Metrics"
}

func (c *GetTestMetrics) Description() string {
	return "Retrieve test performance data including failure counts, durations, and slowest tests"
}

func (c *GetTestMetrics) Documentation() string {
	return `The Get Test Metrics component retrieves test performance data from CircleCI Insights for a specific workflow.

## Use Cases

- **Test health monitoring**: Track test failure rates and identify the most failing tests
- **Performance optimization**: Find the slowest tests that are slowing down your CI pipeline
- **Quality gates**: Check test metrics before proceeding with deployments
- **Reporting**: Collect test performance data for dashboards and trend analysis

## Configuration

- **Project Slug**: CircleCI project slug (e.g., gh/org/repo)
- **Workflow Name**: Name of the workflow to retrieve test metrics for

## Output

Emits test metrics including:
- Average test count per run
- Most failed tests with failure counts
- Slowest tests with duration data
- Total test runs`
}

func (c *GetTestMetrics) Icon() string {
	return "test-tubes"
}

func (c *GetTestMetrics) Color() string {
	return "gray"
}

func (c *GetTestMetrics) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetTestMetrics) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectSlug",
			Label:       "Project slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "CircleCI project slug. Find in CircleCI project settings.",
		},
		{
			Name:        "workflowName",
			Label:       "Workflow name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name of the CircleCI workflow to get test metrics for.",
		},
	}
}

func (c *GetTestMetrics) Setup(ctx core.SetupContext) error {
	spec := GetTestMetricsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.ProjectSlug) == "" {
		return fmt.Errorf("project slug is required")
	}

	if strings.TrimSpace(spec.WorkflowName) == "" {
		return fmt.Errorf("workflow name is required")
	}

	metadata := GetTestMetricsNodeMetadata{}
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

		err = ctx.Metadata.Set(GetTestMetricsNodeMetadata{
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

func (c *GetTestMetrics) Execute(ctx core.ExecutionContext) error {
	spec := GetTestMetricsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	metrics, err := client.GetInsightsTestMetrics(spec.ProjectSlug, strings.TrimSpace(spec.WorkflowName))
	if err != nil {
		return fmt.Errorf("failed to get test metrics: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"circleci.insights.test-metrics",
		[]any{metrics},
	)
}

func (c *GetTestMetrics) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetTestMetrics) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetTestMetrics) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetTestMetrics) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetTestMetrics) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetTestMetrics) Cleanup(ctx core.SetupContext) error {
	return nil
}
