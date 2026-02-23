package circleci

import (
  "fmt"
  "net/http"

  "github.com/google/uuid"
  "github.com/mitchellh/mapstructure"
  "github.com/superplanehq/superplane/pkg/configuration"
  "github.com/superplanehq/superplane/pkg/core"
)

type GetWorkflow struct{}

type GetWorkflowConfiguration struct {
  WorkflowID string `json:"workflowId" mapstructure:"workflowId"`
}

func (c *GetWorkflow) Name() string { return "circleci.getWorkflow" }
func (c *GetWorkflow) Label() string { return "Get Workflow" }
func (c *GetWorkflow) Description() string { return "Get CircleCI workflow details and jobs" }
func (c *GetWorkflow) Documentation() string {
  return `Retrieves workflow details by ID and includes associated jobs.`
}
func (c *GetWorkflow) Icon() string { return "workflow" }
func (c *GetWorkflow) Color() string { return "gray" }
func (c *GetWorkflow) ExampleOutput() map[string]any { return map[string]any{} }
func (c *GetWorkflow) OutputChannels(configuration any) []core.OutputChannel {
  return []core.OutputChannel{core.DefaultOutputChannel}
}
func (c *GetWorkflow) Configuration() []configuration.Field {
  return []configuration.Field{
    {
      Name:        "workflowId",
      Label:       "Workflow ID",
      Type:        configuration.FieldTypeString,
      Required:    true,
      Description: "CircleCI workflow ID",
    },
  }
}
func (c *GetWorkflow) Setup(ctx core.SetupContext) error {
  var config GetWorkflowConfiguration
  if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
    return fmt.Errorf("failed to decode configuration: %w", err)
  }
  if config.WorkflowID == "" {
    return fmt.Errorf("workflowId is required")
  }
  return nil
}
func (c *GetWorkflow) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
  return ctx.DefaultProcessing()
}
func (c *GetWorkflow) Execute(ctx core.ExecutionContext) error {
  var config GetWorkflowConfiguration
  if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
    return fmt.Errorf("failed to decode configuration: %w", err)
  }
  client, err := NewClient(ctx.HTTP, ctx.Integration)
  if err != nil {
    return err
  }
  workflow, err := client.GetWorkflow(config.WorkflowID)
  if err != nil {
    return fmt.Errorf("failed to get workflow: %w", err)
  }
  jobs, err := client.GetWorkflowJobs(config.WorkflowID)
  if err != nil {
    return fmt.Errorf("failed to get workflow jobs: %w", err)
  }

  output := map[string]any{
    "workflow": workflow,
    "jobs":     jobs,
  }
  return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "circleci.workflow", []any{output})
}
func (c *GetWorkflow) Actions() []core.Action { return []core.Action{} }
func (c *GetWorkflow) HandleAction(ctx core.ActionContext) error { return nil }
func (c *GetWorkflow) HandleWebhook(ctx core.WebhookRequestContext) (int, error) { return http.StatusOK, nil }
func (c *GetWorkflow) Cancel(ctx core.ExecutionContext) error { return nil }
func (c *GetWorkflow) Cleanup(ctx core.SetupContext) error { return nil }
