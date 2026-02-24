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

type GetTestMetricsConfiguration struct {
	ProjectSlug     string `json:"projectSlug" mapstructure:"projectSlug"`
	WorkflowName    string `json:"workflowName" mapstructure:"workflowName"`
	Branch          string `json:"branch" mapstructure:"branch"`
	ReportingWindow string `json:"reportingWindow" mapstructure:"reportingWindow"`
}

type GetTestMetricsResult struct {
	ProjectSlug     string         `json:"project_slug"`
	WorkflowName    string         `json:"workflow_name"`
	Branch          string         `json:"branch,omitempty"`
	ReportingWindow string         `json:"reporting_window,omitempty"`
	Metrics         map[string]any `json:"metrics"`
}

func (c *GetTestMetrics) Name() string {
	return "circleci.getTestMetrics"
}

func (c *GetTestMetrics) Label() string {
	return "Get Test Metrics"
}

func (c *GetTestMetrics) Description() string {
	return "Get CircleCI test metrics for a workflow, including failures, durations, and slow tests"
}

func (c *GetTestMetrics) Documentation() string {
	return `The Get Test Metrics component retrieves workflow test insights from CircleCI.

## Use Cases

- **Test health monitoring**: Track failing test counts and reliability trends
- **Duration analysis**: Inspect total and slow test durations to find bottlenecks
- **Regression detection**: Detect changes in test performance over time

## Configuration

- **Project Slug**: CircleCI project slug (e.g. ` + "`gh/org/repo`" + `)
- **Workflow Name**: Workflow name in CircleCI insights
- **Branch** (optional): Filter metrics to a single branch
- **Reporting Window** (optional): CircleCI reporting window (for example ` + "`last-7-days`" + ` or ` + "`last-90-days`" + `)

## Output

Returns:
- ` + "`project_slug`" + `, ` + "`workflow_name`" + `, ` + "`branch`" + `, ` + "`reporting_window`" + `: Request context
- ` + "`metrics`" + `: Raw response from CircleCI workflow test metrics API`
}

func (c *GetTestMetrics) Icon() string {
	return "workflow"
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
			Label:       "Project Slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "CircleCI project slug (e.g. gh/org/repo)",
		},
		{
			Name:        "workflowName",
			Label:       "Workflow Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Workflow name used by CircleCI insights",
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

func (c *GetTestMetrics) Setup(ctx core.SetupContext) error {
	var config GetTestMetricsConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.ProjectSlug) == "" {
		return fmt.Errorf("projectSlug is required")
	}
	if strings.TrimSpace(config.WorkflowName) == "" {
		return fmt.Errorf("workflowName is required")
	}

	return nil
}

func (c *GetTestMetrics) Execute(ctx core.ExecutionContext) error {
	var config GetTestMetricsConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	projectSlug := strings.TrimSpace(config.ProjectSlug)
	workflowName := strings.TrimSpace(config.WorkflowName)
	branch := strings.TrimSpace(config.Branch)
	reportingWindow := strings.TrimSpace(config.ReportingWindow)

	if projectSlug == "" {
		return fmt.Errorf("projectSlug is required")
	}
	if workflowName == "" {
		return fmt.Errorf("workflowName is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	metrics, err := client.GetTestMetrics(projectSlug, workflowName, branch, reportingWindow)
	if err != nil {
		return fmt.Errorf("failed to get test metrics: %w", err)
	}

	result := GetTestMetricsResult{
		ProjectSlug:     projectSlug,
		WorkflowName:    workflowName,
		Branch:          branch,
		ReportingWindow: reportingWindow,
		Metrics:         metrics,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"circleci.testMetrics",
		[]any{result},
	)
}

func (c *GetTestMetrics) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetTestMetrics) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetTestMetrics) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetTestMetrics) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetTestMetrics) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetTestMetrics) Cleanup(ctx core.SetupContext) error {
	return nil
}
