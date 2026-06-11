package issues

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type UpdateIssueComment struct{}

type UpdateIssueCommentConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
	CommentID  string `json:"commentId" mapstructure:"commentId"`
	Body       string `json:"body" mapstructure:"body"`
}

func (c *UpdateIssueComment) Name() string {
	return "github.updateIssueComment"
}

func (c *UpdateIssueComment) Label() string {
	return "Update Issue Comment"
}

func (c *UpdateIssueComment) Description() string {
	return "Update an existing comment on a GitHub issue or pull request"
}

func (c *UpdateIssueComment) Documentation() string {
	return `The **Update Issue Comment** component edits an existing comment on a GitHub issue or pull request.

## Use Cases

- **Status updates**: Update a summary comment on a PR instead of posting new comments on every run
- **Living reports**: Keep a single comment with the latest test results, coverage, or review status
- **Avoid spam**: Update one comment instead of flooding a PR with repeated bot comments

## Configuration

- **Repository**: Select the GitHub repository
- **Comment ID**: The numeric ID of the comment to update (supports expressions)
- **Body**: The new comment text (supports Markdown and expressions)

## Output

Returns the updated comment object including comment ID, URL, body, and timestamps.

## Notes

- The comment ID is returned in the output of ` + "`" + `github.createIssueComment` + "`" + ` as ` + "`" + `id` + "`" + `
- You can store the comment ID in canvas memory on first run, then use it for subsequent updates
- The authenticated user must have permission to edit the comment (must be the comment author or have admin access)`
}

func (c *UpdateIssueComment) Icon() string {
	return "github"
}

func (c *UpdateIssueComment) Color() string {
	return "gray"
}

func (c *UpdateIssueComment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIssueComment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "commentId",
			Label:       "Comment ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The numeric ID of the comment to update",
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The new comment text. Supports Markdown formatting.",
		},
	}
}

func (c *UpdateIssueComment) Setup(ctx core.SetupContext) error {
	var config UpdateIssueCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository == "" {
		return errors.New("repository is required")
	}

	if config.CommentID == "" {
		return errors.New("comment ID is required")
	}

	if config.Body == "" {
		return errors.New("body is required")
	}

	return common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)
}

func (c *UpdateIssueComment) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIssueComment) Execute(ctx core.ExecutionContext) error {
	var config UpdateIssueCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	commentID, err := strconv.ParseInt(config.CommentID, 10, 64)
	if err != nil {
		return fmt.Errorf("comment ID is not a valid number: %v", err)
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	comment := &github.IssueComment{
		Body: &config.Body,
	}

	updatedComment, _, err := client.EditIssueComment(context.Background(), config.Repository, commentID, comment)
	if err != nil {
		return fmt.Errorf("failed to update issue comment: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.issueComment.updated",
		[]any{updatedComment},
	)
}

func (c *UpdateIssueComment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *UpdateIssueComment) Cancel(ctx core.ExecutionContext) error      { return nil }
func (c *UpdateIssueComment) Cleanup(ctx core.SetupContext) error         { return nil }
func (c *UpdateIssueComment) Hooks() []core.Hook                          { return []core.Hook{} }
func (c *UpdateIssueComment) HandleHook(ctx core.ActionHookContext) error { return nil }
