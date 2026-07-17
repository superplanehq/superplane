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

//go:embed example_output_create_issue_comment.json
var exampleOutputCreateIssueComment []byte

type CreateIssueComment struct{}

type CreateIssueCommentConfiguration struct {
	Project  string `mapstructure:"project"`
	IssueIID string `mapstructure:"issueIid"`
	Body     string `mapstructure:"body"`
}

func (c *CreateIssueComment) Name() string {
	return "gitlab.createIssueComment"
}

func (c *CreateIssueComment) Label() string {
	return "Create Issue Comment"
}

func (c *CreateIssueComment) Description() string {
	return "Add a comment to a GitLab issue"
}

func (c *CreateIssueComment) Documentation() string {
	return `The Create Issue Comment component adds a comment (note) to an existing GitLab issue.

## Use Cases

- **Automated updates**: Post deployment status or remediation updates to GitLab issues
- **Runbook linking**: Add runbook links, error details, or status for responders
- **Cross-platform sync**: Sync Slack or PagerDuty notes into GitLab as comments
- **Automated comments**: Add automated comments based on workflow events

## Configuration

- **Project** (required): The GitLab project containing the issue
- **Issue IID** (required): The internal ID (IID) of the issue to comment on (supports expressions)
- **Body** (required): The comment text (supports Markdown and expressions)

## Output

Returns the created note object, including:
- **id**: The ID of the note
- **body**: The comment text
- **author**: The user who created the comment
- **created_at**: When the comment was created`
}

func (c *CreateIssueComment) Icon() string {
	return "gitlab"
}

func (c *CreateIssueComment) Color() string {
	return "orange"
}

func (c *CreateIssueComment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateIssueComment) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputCreateIssueComment, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *CreateIssueComment) Configuration() []configuration.Field {
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
			Description: "The internal ID (IID) of the issue to comment on",
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The comment text. Supports Markdown formatting.",
		},
	}
}

func (c *CreateIssueComment) Setup(ctx core.SetupContext) error {
	var config CreateIssueCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return errors.New("project is required")
	}

	if config.IssueIID == "" {
		return errors.New("issue IID is required")
	}

	if config.Body == "" {
		return errors.New("body is required")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *CreateIssueComment) Execute(ctx core.ExecutionContext) error {
	var config CreateIssueCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	note, err := client.CreateIssueNote(context.Background(), config.Project, config.IssueIID, &CreateNoteRequest{Body: config.Body})
	if err != nil {
		return fmt.Errorf("failed to create issue comment: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.createIssueComment",
		[]any{note},
	)
}

func (c *CreateIssueComment) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateIssueComment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *CreateIssueComment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateIssueComment) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateIssueComment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateIssueComment) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
