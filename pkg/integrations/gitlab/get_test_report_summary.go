package gitlab

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_get_test_report_summary.json
var exampleOutputGetTestReportSummary []byte

type GetTestReportSummary struct{}

type GetTestReportSummaryConfiguration struct {
	Project  string `json:"project" mapstructure:"project"`
	Pipeline string `json:"pipeline" mapstructure:"pipeline"`
}

func (c *GetTestReportSummary) Name() string {
	return "gitlab.getTestReportSummary"
}

func (c *GetTestReportSummary) Label() string {
	return "Get Test Report Summary"
}

func (c *GetTestReportSummary) Description() string {
	return "Get GitLab pipeline test report summary"
}

func (c *GetTestReportSummary) Documentation() string {
	return `The Get Test Report Summary component fetches the test report summary for a GitLab pipeline.

## Configuration

- **Project** (required): The GitLab project containing the pipeline
- **Pipeline** (required): Select a pipeline from the selected project`
}

func (c *GetTestReportSummary) Icon() string {
	return "gitlab"
}

func (c *GetTestReportSummary) Color() string {
	return "orange"
}

func (c *GetTestReportSummary) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputGetTestReportSummary, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *GetTestReportSummary) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetTestReportSummary) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
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
			Name:        "pipeline",
			Label:       "Pipeline",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. 1234567890",
			Description: "The ID of the pipeline",
		},
	}
}

func (c *GetTestReportSummary) Setup(ctx core.SetupContext) error {
	var config GetTestReportSummaryConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Pipeline == "" {
		return fmt.Errorf("pipeline is required")
	}

	return ensureProjectInMetadata(ctx.Metadata, ctx.Integration, config.Project)
}

func (c *GetTestReportSummary) Execute(ctx core.ExecutionContext) error {
	var config GetTestReportSummaryConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	p, err := strconv.ParseFloat(config.Pipeline, 64)
	if err != nil {
		return fmt.Errorf("pipeline ID must be a number: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	summary, err := client.GetPipelineTestReportSummary(config.Project, int(p))
	if err != nil {
		return fmt.Errorf("failed to get test report summary: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "gitlab.testReportSummary", []any{summary})
}

func (c *GetTestReportSummary) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetTestReportSummary) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetTestReportSummary) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetTestReportSummary) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetTestReportSummary) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetTestReportSummary) Cleanup(ctx core.SetupContext) error {
	return nil
}
