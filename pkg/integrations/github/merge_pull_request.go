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

type MergePullRequest struct{}

type MergePullRequestConfiguration struct {
	Repository  string `mapstructure:"repository"`
	PRNumber    int    `mapstructure:"prNumber"`
	CommitTitle string `mapstructure:"commitTitle"`
	MergeMethod string `mapstructure:"mergeMethod"`
}

func (c *MergePullRequest) Name() string {
	return "github.mergePullRequest"
}

func (c *MergePullRequest) Label() string {
	return "Merge Pull Request"
}

func (c *MergePullRequest) Description() string {
	return "Merge a pull request in a GitHub repository"
}

func (c *MergePullRequest) Documentation() string {
	return `The Merge Pull Request component merges an existing pull request in a specified GitHub repository.

## Use Cases

- **Automated deployment**: Merge pull requests automatically after successful CI/CD checks
- **Streamlined dev flow**: Automatically merge trivial PRs or dependency updates
- **Coordinated releases**: Merge multiple PRs across repositories as part of a release workflow

## Configuration

- **Repository**: Select the GitHub repository where the pull request exists
- **PR Number**: The numeric ID of the pull request to merge
- **Commit Title**: Optional title for the merge commit
- **Merge Method**: The method to use for merging (merge, squash, or rebase)

## Output

Returns the merge result object with details including:
- SHA of the merge commit
- Result message
- Merged status`
}

func (c *MergePullRequest) Icon() string {
	return "github"
}

func (c *MergePullRequest) Color() string {
	return "gray"
}

func (c *MergePullRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *MergePullRequest) Configuration() []configuration.Field {
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
			Name:     "prNumber",
			Label:    "Pull Request Number",
			Type:     configuration.FieldTypeNumber,
			Required: true,
		},
		{
			Name:     "commitTitle",
			Label:    "Commit Title",
			Type:     configuration.FieldTypeString,
			Required: false,
		},
		{
			Name:     "mergeMethod",
			Label:    "Merge Method",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "merge",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Merge commit", Value: "merge"},
						{Label: "Squash and merge", Value: "squash"},
						{Label: "Rebase and merge", Value: "rebase"},
					},
				},
			},
		},
	}
}

func (c *MergePullRequest) Setup(ctx core.SetupContext) error {
	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *MergePullRequest) Execute(ctx core.ExecutionContext) error {
	var config MergePullRequestConfiguration
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
	// Prepare the request options
	//
	options := &github.PullRequestOptions{
		MergeMethod: config.MergeMethod,
	}

	// Merge the pull request
	result, _, err := client.PullRequests.Merge(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		config.PRNumber,
		config.CommitTitle,
		options,
	)

	if err != nil {
		return fmt.Errorf("failed to merge pull request: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.pull_request_merge_result",
		[]any{result},
	)
}

func (c *MergePullRequest) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *MergePullRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *MergePullRequest) Actions() []core.Action {
	return []core.Action{}
}

func (c *MergePullRequest) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *MergePullRequest) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *MergePullRequest) Cleanup(ctx core.SetupContext) error {
	return nil
}
