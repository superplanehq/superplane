package bitbucket

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_create_pr_comment.json
var exampleOutputCreatePRComment []byte

type CreatePRComment struct{}

type CreatePRCommentConfiguration struct {
	Repository    string `json:"repository" mapstructure:"repository"`
	PullRequestID string `json:"pullRequestId" mapstructure:"pullRequestId"`
	Body          string `json:"body" mapstructure:"body"`
}

func (c *CreatePRComment) Name() string {
	return "bitbucket.createPRComment"
}

func (c *CreatePRComment) Label() string {
	return "Create Pull Request Comment"
}

func (c *CreatePRComment) Description() string {
	return "Post a comment on a Bitbucket pull request"
}

func (c *CreatePRComment) Documentation() string {
	return `The Create Pull Request Comment component posts a comment on a Bitbucket pull request.

## Use Cases

- **Preview environment URLs**: Post the URL of the environment provisioned for the pull request
- **Agent output**: Return the result of a review or a test run to the people looking at the pull request
- **Deployment feedback**: Report which environment the branch was deployed to, and when

## Configuration

- **Repository** (required): The repository containing the pull request
- **Pull Request ID** (required): The numeric ID of the pull request (supports expressions)
- **Body** (required): The comment body, in Markdown. Supports expressions.

## Permissions

The token needs the ` + "`pullrequest:write`" + ` scope on the repository.

## Output

The component emits the created comment, including:
- **id**: The comment ID
- **content.raw**: The Markdown that was posted
- **links.html.href**: The URL of the comment on the pull request`
}

func (c *CreatePRComment) Icon() string {
	return "bitbucket"
}

func (c *CreatePRComment) Color() string {
	return "blue"
}

func (c *CreatePRComment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreatePRComment) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputCreatePRComment, &example); err != nil {
		return map[string]any{}
	}

	return example
}

func (c *CreatePRComment) Configuration() []configuration.Field {
	return []configuration.Field{
		repositoryField(),
		pullRequestIDField(),
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The comment body, in Markdown",
		},
	}
}

func (c *CreatePRComment) Setup(ctx core.SetupContext) error {
	config := CreatePRCommentConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.PullRequestID == "" {
		return fmt.Errorf("pull request ID is required")
	}

	if config.Body == "" {
		return fmt.Errorf("body is required")
	}

	_, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	return err
}

func (c *CreatePRComment) Execute(ctx core.ExecutionContext) error {
	config := CreatePRCommentConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	pullRequestID, err := parsePullRequestID(config.PullRequestID)
	if err != nil {
		return err
	}

	target, err := resolveRepositoryTarget(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	if err != nil {
		return err
	}

	comment, err := target.Client.CreatePullRequestComment(target.Workspace, target.Repository, pullRequestID, config.Body)
	if err != nil {
		return fmt.Errorf("failed to create pull request comment: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"bitbucket.pullRequestComment",
		[]any{comment},
	)
}

func (c *CreatePRComment) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreatePRComment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *CreatePRComment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreatePRComment) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreatePRComment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreatePRComment) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
