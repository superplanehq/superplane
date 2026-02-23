package circleci

import (
  "fmt"
  "net/http"

  "github.com/google/uuid"
  "github.com/mitchellh/mapstructure"
  "github.com/superplanehq/superplane/pkg/configuration"
  "github.com/superplanehq/superplane/pkg/core"
)

type GetLastWorkflow struct{}

type GetLastWorkflowConfiguration struct {
  ProjectSlug string `json:"projectSlug" mapstructure:"projectSlug"`
  Branch      string `json:"branch" mapstructure:"branch"`
  Status      string `json:"status" mapstructure:"status"`
}

func (c *GetLastWorkflow) Name() string { return "circleci.getLastWorkflow" }
func (c *GetLastWorkflow) Label() string { return "Get Last Workflow" }
func (c *GetLastWorkflow) Description() string { return "Get the most recent workflow for a project" }
func (c *GetLastWorkflow) Documentation() string {
  return `Retrieves the latest workflow for a CircleCI project, optionally filtered by branch or status.`
}
func (c *GetLastWorkflow) Icon() string { return "workflow" }
func (c *GetLastWorkflow) Color() string { return "gray" }
func (c *GetLastWorkflow) ExampleOutput() map[string]any { return map[string]any{} }
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
      Description: "CircleCI project slug (e.g. gh/org/repo)",
    },
    {
      Name:  "branch",
      Label: "Branch",
      Type:  configuration.FieldTypeString,
    },
    {
      Name:  "status",
      Label: "Status",
      Type:  configuration.FieldTypeString,
    },
  }
}
func (c *GetLastWorkflow) Setup(ctx core.SetupContext) error {
  var config GetLastWorkflowConfiguration
  if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
    return fmt.Errorf("failed to decode configuration: %w", err)
  }
  if config.ProjectSlug == "" {
    return fmt.Errorf("projectSlug is required")
  }
  return nil
}
func (c *GetLastWorkflow) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
  return ctx.DefaultProcessing()
}
func (c *GetLastWorkflow) Execute(ctx core.ExecutionContext) error {
  var config GetLastWorkflowConfiguration
  if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
    return fmt.Errorf("failed to decode configuration: %w", err)
  }
  client, err := NewClient(ctx.HTTP, ctx.Integration)
  if err != nil {
    return err
  }
  filters := map[string]string{}
  if config.Branch != "" { filters["branch"] = config.Branch }
  if config.Status != "" { filters["status"] = config.Status }

  result, err := client.GetLastWorkflow(config.ProjectSlug, filters)
  if err != nil {
    return fmt.Errorf("failed to get last workflow: %w", err)
  }
  return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "circleci.workflow", []any{result})
}
func (c *GetLastWorkflow) Actions() []core.Action { return []core.Action{} }
func (c *GetLastWorkflow) HandleAction(ctx core.ActionContext) error { return nil }
func (c *GetLastWorkflow) HandleWebhook(ctx core.WebhookRequestContext) (int, error) { return http.StatusOK, nil }
func (c *GetLastWorkflow) Cancel(ctx core.ExecutionContext) error { return nil }
func (c *GetLastWorkflow) Cleanup(ctx core.SetupContext) error { return nil }
