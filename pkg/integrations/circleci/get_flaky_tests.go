package circleci

import (
  "fmt"
  "net/http"

  "github.com/google/uuid"
  "github.com/mitchellh/mapstructure"
  "github.com/superplanehq/superplane/pkg/configuration"
  "github.com/superplanehq/superplane/pkg/core"
)

type GetFlakyTests struct{}

type GetFlakyTestsConfiguration struct {
  ProjectSlug string `json:"projectSlug" mapstructure:"projectSlug"`
  Branch      string `json:"branch" mapstructure:"branch"`
}

func (c *GetFlakyTests) Name() string { return "circleci.getFlakyTests" }
func (c *GetFlakyTests) Label() string { return "Get Flaky Tests" }
func (c *GetFlakyTests) Description() string { return "Get flaky tests for a project" }
func (c *GetFlakyTests) Documentation() string {
  return `Retrieves flaky test data for a CircleCI project.`
}
func (c *GetFlakyTests) Icon() string { return "workflow" }
func (c *GetFlakyTests) Color() string { return "gray" }
func (c *GetFlakyTests) ExampleOutput() map[string]any { return map[string]any{} }
func (c *GetFlakyTests) OutputChannels(configuration any) []core.OutputChannel {
  return []core.OutputChannel{core.DefaultOutputChannel}
}
func (c *GetFlakyTests) Configuration() []configuration.Field {
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
func (c *GetFlakyTests) Setup(ctx core.SetupContext) error {
  var config GetFlakyTestsConfiguration
  if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
    return fmt.Errorf("failed to decode configuration: %w", err)
  }
  if config.ProjectSlug == "" {
    return fmt.Errorf("projectSlug is required")
  }
  return nil
}
func (c *GetFlakyTests) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
  return ctx.DefaultProcessing()
}
func (c *GetFlakyTests) Execute(ctx core.ExecutionContext) error {
  var config GetFlakyTestsConfiguration
  if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
    return fmt.Errorf("failed to decode configuration: %w", err)
  }
  client, err := NewClient(ctx.HTTP, ctx.Integration)
  if err != nil {
    return err
  }
  filters := map[string]string{}
  if config.Branch != "" { filters["branch"] = config.Branch }

  result, err := client.GetFlakyTests(config.ProjectSlug, filters)
  if err != nil {
    return fmt.Errorf("failed to get flaky tests: %w", err)
  }
  return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "circleci.flaky_tests", []any{result})
}
func (c *GetFlakyTests) Actions() []core.Action { return []core.Action{} }
func (c *GetFlakyTests) HandleAction(ctx core.ActionContext) error { return nil }
func (c *GetFlakyTests) HandleWebhook(ctx core.WebhookRequestContext) (int, error) { return http.StatusOK, nil }
func (c *GetFlakyTests) Cancel(ctx core.ExecutionContext) error { return nil }
func (c *GetFlakyTests) Cleanup(ctx core.SetupContext) error { return nil }
