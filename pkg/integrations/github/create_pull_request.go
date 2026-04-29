package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreatePullRequest struct{}

type CreatePullRequestConfiguration struct {
	Repository string `mapstructure:"repository"`
	Base       string `mapstructure:"base"`
	Head       string `mapstructure:"head"`
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

- **Automated patches**: Create pull requests automatically for security fixes or dependency updates
- **Workflow automation**: Propose changes as part of an automated process
- **Inter-system synchronization**: Keep branches synchronized across repositories via PRs

## Configuration

- **Repository**: Select the GitHub repository where the pull request will be created
- **Base**: The name of the branch you want your changes pulled into (e.g., main)
- **Head**: The name of the branch where your changes are implemented
- **Title**: The pull request title
- **Body**: The pull request description (supports markdown and expressions)
- **Draft**: Whether to create the pull request as a draft

## Output

Returns the created pull request object with details including:
- PR number
- URL
- State
- Created timestamp
- All pull request metadata`
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
			Name:     "base",
			Label:    "Base Branch",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "head",
			Label:    "Head Branch",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "title",
			Label:    "Title",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "body",
			Label:    "Body",
			Type:     configuration.FieldTypeText,
			Required: false,
		},
		{
			Name:     "draft",
			Label:    "Draft",
			Type:     configuration.FieldTypeBool,
			Required: false,
		},
	}
}

func (c *CreatePullRequest) Setup(ctx core.SetupContext) error {
	return ensureRepoInMetadata(
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

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	//
	// Prepare the request based on the configuration
	//
	pullRequest := &github.NewPullRequest{
		Title: &config.Title,
		Base:  &config.Base,
		Head:  &config.Head,
		Draft: &config.Draft,
	}

	if config.Body != "" {
		pullRequest.Body = &config.Body
	}

	// Create the pull request
	pr, _, err := client.PullRequests.Create(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		pullRequest,
	)

	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.pull_request",
		[]any{pr},
	)
}

func (c *CreatePullRequest) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreatePullRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *CreatePullRequest) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreatePullRequest) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreatePullRequest) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreatePullRequest) Cleanup(ctx core.SetupContext) error {
	return nil
}
