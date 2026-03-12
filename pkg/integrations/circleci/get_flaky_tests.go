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

const GetFlakyTestsPayloadType = "circleci.flakyTests"

type GetFlakyTests struct{}

type GetFlakyTestsConfiguration struct {
	ProjectSlug string `json:"projectSlug" mapstructure:"projectSlug"`
}

func (c *GetFlakyTests) Name() string {
	return "circleci.getFlakyTests"
}

func (c *GetFlakyTests) Label() string {
	return "Get Flaky Tests"
}

func (c *GetFlakyTests) Description() string {
	return "Identify flaky tests in a CircleCI project"
}

func (c *GetFlakyTests) Documentation() string {
	return `The Get Flaky Tests component identifies flaky tests in a CircleCI project using the Insights API.

## Use Cases

- **Test reliability**: Identify tests that pass and fail inconsistently
- **CI stability**: Find flaky tests that cause unreliable builds
- **Quality improvement**: Prioritize fixing flaky tests for better developer experience

## Configuration

- **Project Slug**: CircleCI project slug (e.g., gh/username/repo)

## Output

Emits a ` + "`circleci.flakyTests`" + ` payload with a list of flaky tests and their flakiness data.`
}

func (c *GetFlakyTests) Icon() string {
	return "workflow"
}

func (c *GetFlakyTests) Color() string {
	return "gray"
}

func (c *GetFlakyTests) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetFlakyTests) Configuration() []configuration.Field {
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
	}
}

func decodeGetFlakyTestsConfiguration(config any) (GetFlakyTestsConfiguration, error) {
	spec := GetFlakyTestsConfiguration{}
	if err := mapstructure.Decode(config, &spec); err != nil {
		return GetFlakyTestsConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.ProjectSlug = strings.TrimSpace(spec.ProjectSlug)
	if spec.ProjectSlug == "" {
		return GetFlakyTestsConfiguration{}, fmt.Errorf("project slug is required")
	}

	return spec, nil
}

func (c *GetFlakyTests) Setup(ctx core.SetupContext) error {
	_, err := decodeGetFlakyTestsConfiguration(ctx.Configuration)
	return err
}

func (c *GetFlakyTests) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetFlakyTests) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetFlakyTestsConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	flakyTests, err := client.GetFlakyTests(spec.ProjectSlug)
	if err != nil {
		return fmt.Errorf("failed to get flaky tests: %w", err)
	}

	tests := make([]map[string]any, 0, len(flakyTests.FlakyTests))
	for _, t := range flakyTests.FlakyTests {
		tests = append(tests, map[string]any{
			"testName":     t.TestName,
			"classname":    t.Classname,
			"pipelineName": t.PipelineName,
			"workflowName": t.WorkflowName,
			"jobName":      t.JobName,
			"timesFlaky":   t.TimesFlaky,
			"source":       t.Source,
			"file":         t.File,
		})
	}

	payload := map[string]any{
		"flakyTests":      tests,
		"totalFlakyTests": flakyTests.TotalFlakyTests,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetFlakyTestsPayloadType,
		[]any{payload},
	)
}

func (c *GetFlakyTests) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetFlakyTests) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetFlakyTests) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetFlakyTests) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetFlakyTests) Cleanup(ctx core.SetupContext) error {
	return nil
}
