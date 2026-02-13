package gitlab

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetTestReportSummary struct{}

type GetTestReportSummaryConfiguration struct {
	Project    string `json:"project" mapstructure:"project"`
	PipelineID string `json:"pipelineId" mapstructure:"pipelineId"`
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
- **Pipeline ID** (required): Numeric pipeline ID to inspect

## Output

Returns aggregate test statistics and per-suite summary data for the pipeline.`
}

func (c *GetTestReportSummary) Icon() string {
	return "gitlab"
}

func (c *GetTestReportSummary) Color() string {
	return "orange"
}

func (c *GetTestReportSummary) ExampleOutput() map[string]any {
	return map[string]any{
		"total": map[string]any{
			"time":    12.34,
			"count":   40,
			"success": 39,
			"failed":  1,
			"skipped": 0,
			"error":   0,
		},
		"test_suites": []map[string]any{
			{
				"name":          "rspec",
				"total_time":    12.34,
				"total_count":   40,
				"success_count": 39,
				"failed_count":  1,
			},
		},
	}
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
			Name:     "pipelineId",
			Label:    "Pipeline ID",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
	}
}

func (c *GetTestReportSummary) Setup(ctx core.SetupContext) error {
	var config GetTestReportSummaryConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.PipelineID == "" {
		return fmt.Errorf("pipeline ID is required")
	}

	if _, err := strconv.Atoi(config.PipelineID); err != nil {
		return fmt.Errorf("pipeline ID must be a number")
	}

	return ensureProjectInMetadata(ctx.Metadata, ctx.Integration, config.Project)
}

func (c *GetTestReportSummary) Execute(ctx core.ExecutionContext) error {
	var config GetTestReportSummaryConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	pipelineID, err := strconv.Atoi(config.PipelineID)
	if err != nil {
		return fmt.Errorf("pipeline ID must be a number")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	summary, err := client.GetPipelineTestReportSummary(config.Project, pipelineID)
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
