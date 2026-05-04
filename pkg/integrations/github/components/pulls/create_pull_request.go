package pulls

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type CreatePullRequest struct{}

type CreatePullRequestConfiguration struct {
	Repository string `mapstructure:"repository"`
	Head       string `mapstructure:"head"`
	Base       string `mapstructure:"base"`
	Title      string `mapstructure:"title"`
	Body       string `mapstructure:"body"`
	Draft      bool   `mapstructure:"draft"`
}

func (c *CreatePullRequest) Name() string {
	return "github.createPullRequest"
}

func (c *CreatePullRequest) Label() string {
	return "Create Pull Request"
}

func (c *CreatePullRequest) Description() string {
	return "Create a new pull request in a GitHub repository"
}

func (c *CreatePullRequest) Documentation() string {
	return `The Create Pull Request component creates a new pull request in a specified GitHub repository.

## Use Cases

- **Automated PR creation**: Open pull requests automatically as part of CI/CD pipelines
- **Branch promotion**: Create PRs to promote changes between branches
- **Workflow automation**: Generate PRs from external triggers or scheduled workflows

## Configuration

- **Repository**: Select the GitHub repository where the pull request will be created
- **Head**: The branch containing the changes (source branch)
- **Base**: The branch you want the changes pulled into (target branch, defaults to "main")
- **Title**: The pull request title (supports expressions)
- **Body**: Optional pull request description (supports markdown and expressions)
- **Draft**: Whether to create the pull request as a draft

## Output

Returns the created pull request object with details including:
- Pull request number
- URL
- State
- Head and base branch information
- Created timestamp`
}

func (c *CreatePullRequest) Icon() string {
	return "github"
}

func (c *CreatePullRequest) Color() string {
	return "gray"
}

func (c *CreatePullRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreatePullRequest) Configuration() []configuration.Field {
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
			Name:        "head",
			Label:       "Head Branch",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The branch containing the changes to be merged.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "branch",
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{Name: "repository", ValueFrom: &configuration.ParameterValueFrom{Field: "repository"}},
					},
				},
			},
		},
		{
			Name:        "base",
			Label:       "Base Branch",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Default:     "main",
			Description: "The branch the changes will be merged into. Must be different from the head branch.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "branch",
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{Name: "repository", ValueFrom: &configuration.ParameterValueFrom{Field: "repository"}},
					},
				},
			},
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
			Description: "Optional pull request description. Supports markdown and expressions.",
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
	return common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *CreatePullRequest) Execute(ctx core.ExecutionContext) error {
	var config CreatePullRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository == "" {
		return errors.New("repository is required")
	}

	if config.Head == "" {
		return errors.New("head branch is required")
	}

	if config.Base == "" {
		return errors.New("base branch is required")
	}

	if config.Title == "" {
		return errors.New("title is required")
	}

	if config.Head == config.Base {
		return errors.New("head and base branches must be different")
	}

	var appMetadata common.Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	client, err := common.NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	prRequest := &github.NewPullRequest{
		Title: &config.Title,
		Head:  &config.Head,
		Base:  &config.Base,
	}

	if config.Body != "" {
		prRequest.Body = &config.Body
	}

	if config.Draft {
		prRequest.Draft = &config.Draft
	}

	pr, _, err := client.PullRequests.Create(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		prRequest,
	)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", explainGitHubError(err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.pullRequest",
		[]any{pr},
	)
}

func (c *CreatePullRequest) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreatePullRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *CreatePullRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreatePullRequest) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *CreatePullRequest) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreatePullRequest) Cleanup(ctx core.SetupContext) error {
	return nil
}

// explainGitHubError unwraps a *github.ErrorResponse into a more user-friendly
// error so common GitHub 422 messages (e.g. "A pull request already exists",
// "No commits between base and head") surface in the run log instead of a
// generic transport error.
func explainGitHubError(err error) error {
	var ghErr *github.ErrorResponse
	if !errors.As(err, &ghErr) {
		return err
	}

	msg := ghErr.Message
	for _, inner := range ghErr.Errors {
		if inner.Message == "" {
			continue
		}
		if msg == "" {
			msg = inner.Message
			continue
		}
		msg = fmt.Sprintf("%s: %s", msg, inner.Message)
	}

	if msg == "" {
		return err
	}
	return errors.New(msg)
}
