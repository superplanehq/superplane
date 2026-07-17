package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_get_issue.json
var exampleOutputGetIssue []byte

type GetIssue struct{}

type GetIssueConfiguration struct {
	Project  string `mapstructure:"project"`
	IssueIID string `mapstructure:"issueIid"`
}

func (c *GetIssue) Name() string {
	return "gitlab.getIssue"
}

func (c *GetIssue) Label() string {
	return "Get Issue"
}

func (c *GetIssue) Description() string {
	return "Fetch a single GitLab issue by its IID"
}

func (c *GetIssue) Documentation() string {
	return `The Get Issue component fetches a single issue from a GitLab project by its internal ID (IID).

## Use Cases

- **Read state, then act**: Look up an issue's current title, description, state, labels, or assignees before deciding what to do next
- **Data enrichment**: Pull issue details into a workflow to combine with other information
- **Status checking**: Check whether an issue is open or closed before performing an action

## Configuration

- **Project** (required): The GitLab project containing the issue
- **Issue IID** (required): The internal ID (IID) of the issue to fetch (supports expressions)

## Output

Returns the issue object, including:
- **id** / **iid**: The internal and project-relative IDs of the issue
- **title** / **description**: The issue's content
- **state**: The current state of the issue (opened/closed)
- **labels**, **assignees**, **milestone**: Metadata on the issue
- **web_url**: The URL to view the issue in GitLab`
}

func (c *GetIssue) Icon() string {
	return "gitlab"
}

func (c *GetIssue) Color() string {
	return "orange"
}

func (c *GetIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetIssue) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputGetIssue, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *GetIssue) Configuration() []configuration.Field {
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
			Name:        "issueIid",
			Label:       "Issue IID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The internal ID (IID) of the issue",
		},
	}
}

func (c *GetIssue) Setup(ctx core.SetupContext) error {
	var config GetIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return errors.New("project is required")
	}

	if config.IssueIID == "" {
		return errors.New("issue IID is required")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *GetIssue) Execute(ctx core.ExecutionContext) error {
	var config GetIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	issue, err := client.GetIssue(context.Background(), config.Project, config.IssueIID)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.getIssue",
		[]any{issue},
	)
}

func (c *GetIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *GetIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetIssue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetIssue) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
