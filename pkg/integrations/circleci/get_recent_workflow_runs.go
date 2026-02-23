package circleci

import (
  "fmt"
  "net/http"

  "github.com/google/uuid"
  "github.com/mitchellh/mapstructure"
  "github.com/superplanehq/superplane/pkg/configuration"
  "github.com/superplanehq/superplane/pkg/core"
)

type GetRecentWorkflowRuns struct{}

type GetRecentWorkflowRunsConfiguration struct {
  ProjectSlug string `json:"projectSlug" mapstructure:"projectSlug"`
  Branch      string `json:"branch" mapstructure:"branch"`
}

func (c *GetRecentWorkflowRuns) Name() string { return "circleci.getRecentWorkflowRuns" }
func (c *GetRecentWorkflowRuns) Label() string { return "Get Recent Workflow Runs" }
func (c *GetRecentWorkflowRuns) Description() string { return "Get workflow run insights for a project" }
func (c *GetRecentWorkflowRuns) Documentation() string {
  return `Retrieves workflow insights for a CircleCI project, including success rate and duration metrics.`
}
func (c *GetRecentWorkflowRuns) Icon() string { return "workflow" }
func (c *GetRecentWorkflowRuns) Color() string { return "gray" }
func (c *GetRecentWorkflowRuns) ExampleOutput() map[string]any { return map[string]any{} }
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
      Description: "CircleCI project slug (e.g. gh/org/repo)",
    },
    {
      Name:  "branch",
      Label: "Branch",
      Type:  configuration.FieldTypeString,
    },
  }
}
func (c *GetRecentWorkflowRuns) Setup(ctx core.SetupContext) error {
  var config GetRecentWorkflowRunsConfiguration
  if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
    return fmt.Errorf("failed to decode configuration: %w", err)
  }
  if config.ProjectSlug == "" {
    return fmt.Errorf("projectSlug is required")
  }
  return nil
}
func (c *GetRecentWorkflowRuns) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
  return ctx.DefaultProcessing()
}
func (c *GetRecentWorkflowRuns) Execute(ctx core.ExecutionContext) error {
  var config GetRecentWorkflowRunsConfiguration
  if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
    return fmt.Errorf("failed to decode configuration: %w", err)
  }
  client, err := NewClient(ctx.HTTP, ctx.Integration)
  if err != nil {
    return err
  }
  filters := map[string]string{}
  if config.Branch != "" { filters["branch"] = config.Branch }

  result, err := client.GetWorkflowInsights(config.ProjectSlug, filters)
  if err != nil {
    return fmt.Errorf("failed to get workflow insights: %w", err)
  }
  return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "circleci.workflow_insights", []any{result})
}
func (c *GetRecentWorkflowRuns) Actions() []core.Action { return []core.Action{} }
func (c *GetRecentWorkflowRuns) HandleAction(ctx core.ActionContext) error { return nil }
func (c *GetRecentWorkflowRuns) HandleWebhook(ctx core.WebhookRequestContext) (int, error) { return http.StatusOK, nil }
func (c *GetRecentWorkflowRuns) Cancel(ctx core.ExecutionContext) error { return nil }
func (c *GetRecentWorkflowRuns) Cleanup(ctx core.SetupContext) error { return nil }
