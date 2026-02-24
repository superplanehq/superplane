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

type GetFlakyTests struct{}

type GetFlakyTestsConfiguration struct {
	ProjectSlug     string `json:"projectSlug" mapstructure:"projectSlug"`
	Branch          string `json:"branch" mapstructure:"branch"`
	ReportingWindow string `json:"reportingWindow" mapstructure:"reportingWindow"`
}

type GetFlakyTestsResult struct {
	ProjectSlug     string         `json:"project_slug"`
	Branch          string         `json:"branch,omitempty"`
	ReportingWindow string         `json:"reporting_window,omitempty"`
	FlakyTests      map[string]any `json:"flaky_tests"`
}

func (c *GetFlakyTests) Name() string {
	return "circleci.getFlakyTests"
}

func (c *GetFlakyTests) Label() string {
	return "Get Flaky Tests"
}

func (c *GetFlakyTests) Description() string {
	return "Get flaky tests for a project with flakiness rates and test details"
}

func (c *GetFlakyTests) Documentation() string {
	return `The Get Flaky Tests component retrieves flaky test insights for a CircleCI project.

## Use Cases

- **Flaky test detection**: Identify unstable tests that intermittently fail
- **Reliability improvement**: Prioritize tests by flakiness rate
- **Quality reporting**: Feed flaky-test trends into dashboards and alerts

## Configuration

- **Project Slug**: CircleCI project slug (e.g. ` + "`gh/org/repo`" + `)
- **Branch** (optional): Filter results to a single branch
- **Reporting Window** (optional): CircleCI reporting window (for example ` + "`last-7-days`" + ` or ` + "`last-90-days`" + `)

## Output

Returns:
- ` + "`project_slug`" + `, ` + "`branch`" + `, ` + "`reporting_window`" + `: Request context
- ` + "`flaky_tests`" + `: Raw response from CircleCI flaky tests insights API`
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
			Name:        "projectSlug",
			Label:       "Project Slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "CircleCI project slug (e.g. gh/org/repo)",
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

func (c *GetFlakyTests) Setup(ctx core.SetupContext) error {
	var config GetFlakyTestsConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.ProjectSlug) == "" {
		return fmt.Errorf("projectSlug is required")
	}

	return nil
}

func (c *GetFlakyTests) Execute(ctx core.ExecutionContext) error {
	var config GetFlakyTestsConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	projectSlug := strings.TrimSpace(config.ProjectSlug)
	branch := strings.TrimSpace(config.Branch)
	reportingWindow := strings.TrimSpace(config.ReportingWindow)
	if projectSlug == "" {
		return fmt.Errorf("projectSlug is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	flakyTests, err := client.GetFlakyTests(projectSlug, branch, reportingWindow)
	if err != nil {
		return fmt.Errorf("failed to get flaky tests: %w", err)
	}

	result := GetFlakyTestsResult{
		ProjectSlug:     projectSlug,
		Branch:          branch,
		ReportingWindow: reportingWindow,
		FlakyTests:      flakyTests,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"circleci.flakyTests",
		[]any{result},
	)
}

func (c *GetFlakyTests) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
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
