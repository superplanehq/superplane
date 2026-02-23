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

type GetFlakyTestsSpec struct {
	ProjectSlug string `json:"projectSlug" mapstructure:"projectSlug"`
}

type GetFlakyTestsNodeMetadata struct {
	ProjectID   string `json:"projectId" mapstructure:"projectId"`
	ProjectSlug string `json:"projectSlug" mapstructure:"projectSlug"`
	ProjectName string `json:"projectName" mapstructure:"projectName"`
}

func (c *GetFlakyTests) Name() string {
	return "circleci.getFlakyTests"
}

func (c *GetFlakyTests) Label() string {
	return "Get Flaky Tests"
}

func (c *GetFlakyTests) Description() string {
	return "Identify flaky tests in a project with flakiness rate and test details"
}

func (c *GetFlakyTests) Documentation() string {
	return `The Get Flaky Tests component identifies flaky tests in a CircleCI project using the Insights API.

## Use Cases

- **Test reliability monitoring**: Identify tests that intermittently fail across runs
- **Quality gates**: Block deployments when flaky test count exceeds a threshold
- **Maintenance prioritization**: Find the most flaky tests to prioritize for fixing
- **CI/CD health**: Monitor test suite stability over time

## Configuration

- **Project Slug**: CircleCI project slug (e.g., gh/org/repo)

## Output

Emits flaky test data including:
- List of flaky tests with test name, class name, and source file
- Flakiness count (number of times the test flaked)
- Associated workflow and job names
- Total count of flaky tests detected`
}

func (c *GetFlakyTests) Icon() string {
	return "test-tubes"
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
			Label:       "Project slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "CircleCI project slug. Find in CircleCI project settings.",
		},
	}
}

func (c *GetFlakyTests) Setup(ctx core.SetupContext) error {
	spec := GetFlakyTestsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.ProjectSlug) == "" {
		return fmt.Errorf("project slug is required")
	}

	metadata := GetFlakyTestsNodeMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	projectChanged := metadata.ProjectSlug != spec.ProjectSlug
	if projectChanged {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		project, err := client.GetProject(spec.ProjectSlug)
		if err != nil {
			return fmt.Errorf("project not found or inaccessible: %w", err)
		}

		err = ctx.Metadata.Set(GetFlakyTestsNodeMetadata{
			ProjectID:   project.ID,
			ProjectSlug: project.Slug,
			ProjectName: project.Name,
		})
		if err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}
	}

	return nil
}

func (c *GetFlakyTests) Execute(ctx core.ExecutionContext) error {
	spec := GetFlakyTestsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	flakyTests, totalCount, err := client.GetInsightsFlakyTests(spec.ProjectSlug)
	if err != nil {
		return fmt.Errorf("failed to get flaky tests: %w", err)
	}

	output := map[string]any{
		"flaky_tests": flakyTests,
		"total_count": totalCount,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"circleci.insights.flaky-tests",
		[]any{output},
	)
}

func (c *GetFlakyTests) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetFlakyTests) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetFlakyTests) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetFlakyTests) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetFlakyTests) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetFlakyTests) Cleanup(ctx core.SetupContext) error {
	return nil
}
