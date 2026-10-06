package bitbucket

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdatePullRequest struct {
	pullRequestAction
}

type UpdatePullRequestConfiguration struct {
	Repository string `mapstructure:"repository" json:"repository"`
	PullNumber string `mapstructure:"pullNumber" json:"pullNumber"`
	Title      string `mapstructure:"title" json:"title"`
	Body       string `mapstructure:"body" json:"body"`
}

func (c *UpdatePullRequest) Name() string {
	return "bitbucket.updatePullRequest"
}

func (c *UpdatePullRequest) Label() string {
	return "Update Pull Request"
}

func (c *UpdatePullRequest) Description() string {
	return "Update the title or description of a Bitbucket pull request"
}

func (c *UpdatePullRequest) Documentation() string {
	return `The Update Pull Request component changes the title or description of a pull request in a Bitbucket repository.

## Configuration

- **Repository**: The Bitbucket repository, in workspace/repository format
- **Pull Request ID**: The pull request number. Supports expressions.
- **Title**: Optional new title
- **Body**: Optional new description. Supports Markdown and expressions.

Set at least one of Title or Body. Empty values are not sent.

## Output

Emits the updated pull request that Bitbucket returns.`
}

func (c *UpdatePullRequest) ExampleOutput() map[string]any {
	return examplePullRequestOutput()
}

func (c *UpdatePullRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdatePullRequest) Configuration() []configuration.Field {
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
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional new title. Supports expressions.",
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Optional new description. Supports Markdown and expressions.",
		},
	}
}

func (c *UpdatePullRequest) Setup(ctx core.SetupContext) error {
	config, err := decodeUpdatePullRequestConfiguration(ctx.Configuration)
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

func (c *UpdatePullRequest) Execute(ctx core.ExecutionContext) error {
	config, err := decodeUpdatePullRequestConfiguration(ctx.Configuration)
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

	pullRequest, err := client.UpdatePullRequest(config.Repository, id, UpdatePullRequestRequest{
		Title:       config.Title,
		Description: config.Body,
	})
	if err != nil {
		return fmt.Errorf("failed to update pull request: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PayloadTypePullRequest, []any{pullRequest})
}

func decodeUpdatePullRequestConfiguration(raw any) (UpdatePullRequestConfiguration, error) {
	config := UpdatePullRequestConfiguration{}
	if err := mapstructure.Decode(raw, &config); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.Title = strings.TrimSpace(config.Title)

	if strings.TrimSpace(config.Repository) == "" {
		return config, errors.New("repository is required")
	}
	if strings.TrimSpace(config.PullNumber) == "" {
		return config, errors.New("pull request ID is required")
	}
	if config.Title == "" && strings.TrimSpace(config.Body) == "" {
		return config, errors.New("title or body is required")
	}
	return config, nil
}
