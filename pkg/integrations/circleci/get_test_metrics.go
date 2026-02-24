package circleci

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetTestMetrics struct{}

type GetTestMetricsConfiguration struct {
	ProjectSlug  string `json:"projectSlug" mapstructure:"projectSlug"`
	WorkflowName string `json:"workflowName" mapstructure:"workflowName"`
	Branch       string `json:"branch" mapstructure:"branch"`
}

func (c *GetTestMetrics) Name() string        { return "circleci.getTestMetrics" }
func (c *GetTestMetrics) Label() string       { return "Get Test Metrics" }
func (c *GetTestMetrics) Description() string { return "Get test metrics for a workflow" }
func (c *GetTestMetrics) Documentation() string {
	return `Retrieves test metrics for a CircleCI workflow, including failure counts and durations.`
}
func (c *GetTestMetrics) Icon() string                  { return "workflow" }
func (c *GetTestMetrics) Color() string                 { return "gray" }
func (c *GetTestMetrics) ExampleOutput() map[string]any { return map[string]any{} }
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
			Description: "CircleCI project slug (e.g. gh/org/repo)",
		},
		{
			Name:        "workflowName",
			Label:       "Workflow name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Workflow name as shown in CircleCI",
		},
		{
			Name:  "branch",
			Label: "Branch",
			Type:  configuration.FieldTypeString,
		},
	}
}
func (c *GetTestMetrics) Setup(ctx core.SetupContext) error {
	var config GetTestMetricsConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if config.ProjectSlug == "" || config.WorkflowName == "" {
		return fmt.Errorf("projectSlug and workflowName are required")
	}
	return nil
}
func (c *GetTestMetrics) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *GetTestMetrics) Execute(ctx core.ExecutionContext) error {
	var config GetTestMetricsConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	filters := map[string]string{}
	if config.Branch != "" {
		filters["branch"] = config.Branch
	}

	result, err := client.GetWorkflowTestMetrics(config.ProjectSlug, config.WorkflowName, filters)
	if err != nil {
		return fmt.Errorf("failed to get test metrics: %w", err)
	}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "circleci.test_metrics", []any{result})
}
func (c *GetTestMetrics) Actions() []core.Action                    { return []core.Action{} }
func (c *GetTestMetrics) HandleAction(ctx core.ActionContext) error { return nil }
func (c *GetTestMetrics) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
func (c *GetTestMetrics) Cancel(ctx core.ExecutionContext) error { return nil }
func (c *GetTestMetrics) Cleanup(ctx core.SetupContext) error    { return nil }
