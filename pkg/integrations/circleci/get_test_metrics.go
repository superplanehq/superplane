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

const GetTestMetricsPayloadType = "circleci.testMetrics"

type GetTestMetrics struct{}

type GetTestMetricsConfiguration struct {
	ProjectSlug  string `json:"projectSlug" mapstructure:"projectSlug"`
	WorkflowName string `json:"workflowName" mapstructure:"workflowName"`
}

func (c *GetTestMetrics) Name() string {
	return "circleci.getTestMetrics"
}

func (c *GetTestMetrics) Label() string {
	return "Get Test Metrics"
}

func (c *GetTestMetrics) Description() string {
	return "Get test performance data for a CircleCI workflow"
}

func (c *GetTestMetrics) Documentation() string {
	return `The Get Test Metrics component fetches test performance data from the CircleCI Insights API.

## Use Cases

- **Test health monitoring**: Track most failed and slowest tests
- **CI optimization**: Identify tests that slow down your pipeline
- **Quality tracking**: Monitor test success rates across runs

## Configuration

- **Project Slug**: CircleCI project slug (e.g., gh/username/repo)
- **Workflow Name**: Name of the workflow to get test metrics for

## Output

Emits a ` + "`circleci.testMetrics`" + ` payload with most failed tests, slowest tests, and test run summaries.`
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
			Name:     "projectSlug",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
		{
			Name:     "workflowName",
			Label:    "Workflow",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeWorkflow,
					Parameters: []configuration.ParameterRef{
						{
							Name: "projectSlug",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "projectSlug",
							},
						},
					},
				},
			},
		},
	}
}

func decodeGetTestMetricsConfiguration(config any) (GetTestMetricsConfiguration, error) {
	spec := GetTestMetricsConfiguration{}
	if err := mapstructure.Decode(config, &spec); err != nil {
		return GetTestMetricsConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.ProjectSlug = strings.TrimSpace(spec.ProjectSlug)
	if spec.ProjectSlug == "" {
		return GetTestMetricsConfiguration{}, fmt.Errorf("project slug is required")
	}

	spec.WorkflowName = strings.TrimSpace(spec.WorkflowName)
	if spec.WorkflowName == "" {
		return GetTestMetricsConfiguration{}, fmt.Errorf("workflow name is required")
	}

	return spec, nil
}

func (c *GetTestMetrics) Setup(ctx core.SetupContext) error {
	_, err := decodeGetTestMetricsConfiguration(ctx.Configuration)
	return err
}

func (c *GetTestMetrics) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetTestMetrics) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetTestMetricsConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	metrics, err := client.GetTestMetrics(spec.ProjectSlug, spec.WorkflowName)
	if err != nil {
		return fmt.Errorf("failed to get test metrics: %w", err)
	}

	mostFailed := make([]map[string]any, 0, len(metrics.MostFailedTests))
	for _, t := range metrics.MostFailedTests {
		mostFailed = append(mostFailed, map[string]any{
			"testName":   t.TestName,
			"classname":  t.Classname,
			"failedRuns": t.FailedRuns,
			"totalRuns":  t.TotalRuns,
			"flaky":      t.Flaky,
		})
	}

	slowest := make([]map[string]any, 0, len(metrics.SlowestTests))
	for _, t := range metrics.SlowestTests {
		slowest = append(slowest, map[string]any{
			"testName":        t.TestName,
			"classname":       t.Classname,
			"failedRuns":      t.FailedRuns,
			"totalRuns":       t.TotalRuns,
			"flaky":           t.Flaky,
			"p50DurationSecs": t.P50Secs,
		})
	}

	testRuns := make([]map[string]any, 0, len(metrics.TestRuns))
	for _, r := range metrics.TestRuns {
		testRuns = append(testRuns, map[string]any{
			"pipelineNumber": r.PipelineNumber,
			"workflowId":     r.WorkflowID,
			"successRate":    r.SuccessRate,
			"testCounts":     r.TestCounts,
		})
	}

	payload := map[string]any{
		"mostFailedTests": mostFailed,
		"slowestTests":    slowest,
		"totalTestRuns":   metrics.TotalTestRuns,
		"testRuns":        testRuns,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetTestMetricsPayloadType,
		[]any{payload},
	)
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
