package bitbucket

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreatePullRequest struct {
	pullRequestAction
}

type CreatePullRequestConfiguration struct {
	Repository string `mapstructure:"repository" json:"repository"`
	Head       string `mapstructure:"head" json:"head"`
	Base       string `mapstructure:"base" json:"base"`
	Title      string `mapstructure:"title" json:"title"`
	Body       string `mapstructure:"body" json:"body"`
	Draft      bool   `mapstructure:"draft" json:"draft"`
}

func (c *CreatePullRequest) Name() string {
	return "bitbucket.createPullRequest"
}

func (c *CreatePullRequest) Label() string {
	return "Create Pull Request"
}

func (c *CreatePullRequest) Description() string {
	return "Create a pull request in a Bitbucket repository"
}

func (c *CreatePullRequest) Documentation() string {
	return `The Create Pull Request component opens a pull request in a Bitbucket repository.

## Configuration

- **Repository**: The Bitbucket repository, in workspace/repository format
- **Head**: The source branch that contains the changes
- **Base**: The destination branch. Defaults to "main".
- **Title**: The pull request title. Supports expressions.
- **Body**: Optional pull request description. Supports Markdown and expressions.
- **Draft**: Create the pull request as a draft

## Output

Emits the pull request that Bitbucket returns. Use ` + "`id`" + ` for the pull request number and ` + "`links.html.href`" + ` for its URL.

## Limitations

Only pull requests inside one repository are supported. Pull requests from forks are not supported.`
}

func (c *CreatePullRequest) ExampleOutput() map[string]any {
	return examplePullRequestOutput()
}

func (c *CreatePullRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreatePullRequest) Configuration() []configuration.Field {
	return []configuration.Field{
		repositoryField(),
		{
			Name:        "head",
			Label:       "Head Branch",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The source branch that contains the changes.",
		},
		{
			Name:        "base",
			Label:       "Base Branch",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "main",
			Description: "The destination branch. It must be different from the head branch.",
		},
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Pull request title. Supports expressions.",
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Optional pull request description. Supports Markdown and expressions.",
		},
		{
			Name:        "draft",
			Label:       "Draft",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Create the pull request as a draft",
		},
	}
}

func (c *CreatePullRequest) Setup(ctx core.SetupContext) error {
	config, err := decodeCreatePullRequestConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if !isExpression(config.Head) && !isExpression(config.Base) && config.Head == config.Base {
		return errors.New("head and base branches must be different")
	}
	return setupRepository(ctx, config.Repository)
}

func (c *CreatePullRequest) Execute(ctx core.ExecutionContext) error {
	config, err := decodeCreatePullRequestConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if config.Head == config.Base {
		return errors.New("head and base branches must be different")
	}

	client, err := newIntegrationClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize Bitbucket client: %w", err)
	}

	pullRequest, err := client.CreatePullRequest(config.Repository, CreatePullRequestRequest{
		Title:       config.Title,
		Description: config.Body,
		Source:      config.Head,
		Destination: config.Base,
		Draft:       config.Draft,
	})
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PayloadTypePullRequest, []any{pullRequest})
}

func decodeCreatePullRequestConfiguration(raw any) (CreatePullRequestConfiguration, error) {
	config := CreatePullRequestConfiguration{}
	if err := mapstructure.Decode(raw, &config); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.Head = strings.TrimSpace(config.Head)
	config.Base = strings.TrimSpace(config.Base)
	config.Title = strings.TrimSpace(config.Title)

	if strings.TrimSpace(config.Repository) == "" {
		return config, errors.New("repository is required")
	}
	if config.Head == "" {
		return config, errors.New("head branch is required")
	}
	if config.Base == "" {
		return config, errors.New("base branch is required")
	}
	if config.Title == "" {
		return config, errors.New("title is required")
	}
	return config, nil
}
