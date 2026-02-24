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

type GetRecentWorkflowRunsConfiguration struct {
	ProjectSlug     string `json:"projectSlug" mapstructure:"projectSlug"`
	Branch          string `json:"branch" mapstructure:"branch"`
	ReportingWindow string `json:"reportingWindow" mapstructure:"reportingWindow"`
}

type GetRecentWorkflowRunsResult struct {
	ProjectSlug     string         `json:"project_slug"`
	Branch          string         `json:"branch,omitempty"`
	ReportingWindow string         `json:"reporting_window,omitempty"`
	Insights        map[string]any `json:"insights"`
}

func (c *GetRecentWorkflowRuns) Name() string {
	return "circleci.getRecentWorkflowRuns"
}

func (c *GetRecentWorkflowRuns) Label() string {
	return "Get Recent Workflow Runs"
}

func (c *GetRecentWorkflowRuns) Description() string {
	return "Get aggregated workflow run insights, including success rate, throughput, and duration metrics"
}

func (c *GetRecentWorkflowRuns) Documentation() string {
	return `The Get Recent Workflow Runs component retrieves aggregated workflow run insights from CircleCI.

## Use Cases

- **Reliability tracking**: Monitor success/failure trends across workflows
- **Performance monitoring**: Analyze duration and throughput metrics
- **Branch health checks**: Inspect metrics for a specific branch

## Configuration

- **Project Slug**: CircleCI project slug (e.g. ` + "`gh/org/repo`" + `)
- **Branch** (optional): Filter insights to a single branch
- **Reporting Window** (optional): CircleCI reporting window (for example ` + "`last-7-days`" + ` or ` + "`last-90-days`" + `)

## Output

Returns:
- ` + "`project_slug`" + `, ` + "`branch`" + `, ` + "`reporting_window`" + `: Request context
- ` + "`insights`" + `: Raw response from CircleCI insights workflows API`
}

func (c *GetRecentWorkflowRuns) Icon() string {
	return "workflow"
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
			Label:       "Project Slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "CircleCI project slug (e.g. gh/org/repo)",
		},
		{
			Name:        "branch",
			Label:       "Branch",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional branch filter",
			Placeholder: "e.g. main",
		},
		{
			Name:        "reportingWindow",
			Label:       "Reporting Window",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional reporting window (e.g. last-7-days, last-90-days)",
			Placeholder: "e.g. last-90-days",
		},
	}
}

func (c *GetRecentWorkflowRuns) Setup(ctx core.SetupContext) error {
	var config GetRecentWorkflowRunsConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.ProjectSlug) == "" {
		return fmt.Errorf("projectSlug is required")
	}

	return nil
}

func (c *GetRecentWorkflowRuns) Execute(ctx core.ExecutionContext) error {
	var config GetRecentWorkflowRunsConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	projectSlug := strings.TrimSpace(config.ProjectSlug)
	branch := strings.TrimSpace(config.Branch)
	reportingWindow := strings.TrimSpace(config.ReportingWindow)
	if projectSlug == "" {
		return fmt.Errorf("projectSlug is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	insights, err := client.GetInsightsWorkflows(projectSlug, branch, reportingWindow)
	if err != nil {
		return fmt.Errorf("failed to get workflow insights: %w", err)
	}

	result := GetRecentWorkflowRunsResult{
		ProjectSlug:     projectSlug,
		Branch:          branch,
		ReportingWindow: reportingWindow,
		Insights:        insights,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"circleci.recentWorkflowRuns",
		[]any{result},
	)
}

func (c *GetRecentWorkflowRuns) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetRecentWorkflowRuns) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetRecentWorkflowRuns) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetRecentWorkflowRuns) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetRecentWorkflowRuns) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetRecentWorkflowRuns) Cleanup(ctx core.SetupContext) error {
	return nil
}
