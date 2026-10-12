package bitbucket

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreatePullRequestComment struct {
	pullRequestAction
}

type CreatePullRequestCommentConfiguration struct {
	Repository      string `mapstructure:"repository" json:"repository"`
	PullNumber      string `mapstructure:"pullNumber" json:"pullNumber"`
	Body            string `mapstructure:"body" json:"body"`
	ParentCommentID string `mapstructure:"parentCommentId" json:"parentCommentId"`
}

func (c *CreatePullRequestComment) Name() string {
	return "bitbucket.createPullRequestComment"
}

func (c *CreatePullRequestComment) Label() string {
	return "Create Pull Request Comment"
}

func (c *CreatePullRequestComment) Description() string {
	return "Add a comment to a Bitbucket pull request"
}

func (c *CreatePullRequestComment) Documentation() string {
	return `The Create Pull Request Comment component adds a comment to a pull request in a Bitbucket repository.

## Configuration

- **Repository**: The Bitbucket repository, in workspace/repository format
- **Pull Request ID**: The pull request number. Supports expressions.
- **Body**: The comment text. Supports Markdown and expressions.
- **Parent Comment ID**: Optional. Reply inside the thread of another comment. Supports expressions.

## Output

Emits the comment that Bitbucket returns.`
}

func (c *CreatePullRequestComment) ExampleOutput() map[string]any {
	return examplePullRequestCommentOutput()
}

func (c *CreatePullRequestComment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreatePullRequestComment) Configuration() []configuration.Field {
	return []configuration.Field{
		repositoryField(),
		{
			Name:        "pullNumber",
			Label:       "Pull Request ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The pull request number. Supports expressions.",
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The comment text. Supports Markdown and expressions.",
		},
		{
			Name:        "parentCommentId",
			Label:       "Parent Comment ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Reply inside the thread of another comment. Supports expressions.",
		},
	}
}

func (c *CreatePullRequestComment) Setup(ctx core.SetupContext) error {
	config, err := decodeCreatePullRequestCommentConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if !isExpression(config.PullNumber) {
		if _, err := parsePullRequestID(config.PullNumber); err != nil {
			return err
		}
	}
	return setupRepository(ctx, config.Repository)
}

func (c *CreatePullRequestComment) Execute(ctx core.ExecutionContext) error {
	config, err := decodeCreatePullRequestCommentConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	id, err := parsePullRequestID(config.PullNumber)
	if err != nil {
		return err
	}
	if err := requireRepositoryInWorkspace(ctx.Integration, config.Repository); err != nil {
		return err
	}

	client, err := newIntegrationClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize Bitbucket client: %w", err)
	}

	comment, err := client.CreatePullRequestCommentWithParent(config.Repository, id, config.Body, config.ParentCommentID)
	if err != nil {
		return fmt.Errorf("failed to create pull request comment: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "bitbucket.pullRequestComment", []any{comment})
}

func decodeCreatePullRequestCommentConfiguration(raw any) (CreatePullRequestCommentConfiguration, error) {
	config := CreatePullRequestCommentConfiguration{}
	if err := mapstructure.Decode(raw, &config); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(config.Repository) == "" {
		return config, errors.New("repository is required")
	}
	if strings.TrimSpace(config.PullNumber) == "" {
		return config, errors.New("pull request ID is required")
	}
	if strings.TrimSpace(config.Body) == "" {
		return config, errors.New("body is required")
	}
	return config, nil
}
