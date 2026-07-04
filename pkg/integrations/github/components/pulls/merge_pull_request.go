package pulls

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const (
	defaultMergeMethod = "merge"

	mergeMethodMerge  = "merge"
	mergeMethodSquash = "squash"
	mergeMethodRebase = "rebase"
)

type MergePullRequest struct{}

type MergePullRequestConfiguration struct {
	Repository    string `mapstructure:"repository" json:"repository"`
	PullNumber    any    `mapstructure:"pullNumber" json:"pullNumber"`
	MergeMethod   string `mapstructure:"mergeMethod" json:"mergeMethod"`
	SHA           string `mapstructure:"sha" json:"sha"`
	CommitTitle   string `mapstructure:"commitTitle" json:"commitTitle"`
	CommitMessage string `mapstructure:"commitMessage" json:"commitMessage"`
}

type mergePullRequestInput struct {
	Repository    string
	PullNumber    int
	MergeMethod   string
	SHA           string
	CommitTitle   string
	CommitMessage string
}

func (c *MergePullRequest) Name() string {
	return "github.mergePullRequest"
}

func (c *MergePullRequest) Label() string {
	return "Merge Pull Request"
}

func (c *MergePullRequest) Description() string {
	return "Merge an open pull request in a GitHub repository"
}

func (c *MergePullRequest) Documentation() string {
	return `The Merge Pull Request component merges an open pull request in a GitHub repository.

## Use Cases

- **Automated PR merge gates**: Merge a pull request after status checks or check runs pass
- **Release automation**: Merge approved promotion branches into a release branch
- **Queue workflows**: Merge pull requests from a controlled SuperPlane workflow

## Configuration

- **Repository**: Select the GitHub repository containing the pull request
- **Pull Request Number**: Pull request number to merge. Expressions are supported.
- **Merge Method**: Merge strategy to use. Defaults to "merge".
- **Expected SHA**: Optional head SHA guard. GitHub rejects the merge if the pull request head has changed.
- **Commit Title**: Optional title for the merge commit, squash commit, or rebase commit.
- **Commit Message**: Optional commit message for the merge commit or squash commit.

## Output

Returns GitHub's merge result, including whether the pull request was merged, the resulting commit SHA, and GitHub's message.`
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
			Name:        "pullNumber",
			Label:       "Pull Request Number",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "42 or {{event.data.pull_request.number}}",
			Description: "Pull request number to merge. Supports expressions.",
		},
		{
			Name:        "mergeMethod",
			Label:       "Merge Method",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     defaultMergeMethod,
			Description: "Merge strategy to use. Defaults to merge commit.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Merge Commit", Value: mergeMethodMerge},
						{Label: "Squash", Value: mergeMethodSquash},
						{Label: "Rebase", Value: mergeMethodRebase},
					},
				},
			},
		},
		{
			Name:        "sha",
			Label:       "Expected SHA",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "{{event.data.pull_request.head.sha}}",
			Description: "Optional pull request head SHA. GitHub rejects the merge if the head changed.",
		},
		{
			Name:        "commitTitle",
			Label:       "Commit Title",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional title for the merge commit, squash commit, or rebase commit.",
		},
		{
			Name:        "commitMessage",
			Label:       "Commit Message",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Optional commit message for the merge commit or squash commit.",
		},
	}
}

func (c *MergePullRequest) Setup(ctx core.SetupContext) error {
	var config MergePullRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateMergePullRequestSetup(config); err != nil {
		return err
	}

	return common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)
}

func (c *MergePullRequest) Execute(ctx core.ExecutionContext) error {
	var config MergePullRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	input, err := buildMergePullRequestInput(config)
	if err != nil {
		return err
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	options := &github.PullRequestOptions{
		MergeMethod: input.MergeMethod,
		CommitTitle: input.CommitTitle,
		SHA:         input.SHA,
	}

	result, _, err := client.MergePullRequest(
		context.Background(),
		input.Repository,
		input.PullNumber,
		input.CommitMessage,
		options,
	)
	if err != nil {
		return fmt.Errorf("failed to merge pull request: %w", explainGitHubError(err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.pullRequestMerge",
		[]any{result},
	)
}

func (c *MergePullRequest) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *MergePullRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *MergePullRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *MergePullRequest) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *MergePullRequest) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *MergePullRequest) Cleanup(ctx core.SetupContext) error {
	return nil
}

func validateMergePullRequestSetup(config MergePullRequestConfiguration) error {
	if strings.TrimSpace(config.Repository) == "" {
		return errors.New("repository is required")
	}

	if pullNumberText(config.PullNumber) == "" {
		return errors.New("pull request number is required")
	}

	if _, err := normalizeMergeMethod(config.MergeMethod); err != nil {
		return err
	}

	if common.IsExpression(pullNumberText(config.PullNumber)) {
		return nil
	}

	_, err := parsePullNumber(config.PullNumber)
	return err
}

func buildMergePullRequestInput(config MergePullRequestConfiguration) (*mergePullRequestInput, error) {
	if strings.TrimSpace(config.Repository) == "" {
		return nil, errors.New("repository is required")
	}

	pullNumber, err := parsePullNumber(config.PullNumber)
	if err != nil {
		return nil, err
	}

	mergeMethod, err := normalizeMergeMethod(config.MergeMethod)
	if err != nil {
		return nil, err
	}

	return &mergePullRequestInput{
		Repository:    strings.TrimSpace(config.Repository),
		PullNumber:    pullNumber,
		MergeMethod:   mergeMethod,
		SHA:           strings.TrimSpace(config.SHA),
		CommitTitle:   config.CommitTitle,
		CommitMessage: config.CommitMessage,
	}, nil
}

func parsePullNumber(value any) (int, error) {
	trimmed := pullNumberText(value)
	if trimmed == "" {
		return 0, errors.New("pull request number is required")
	}

	number, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, errors.New("pull request number must be a positive integer")
	}

	if number <= 0 {
		return 0, errors.New("pull request number must be a positive integer")
	}

	return number, nil
}

func pullNumberText(value any) string {
	if value == nil {
		return ""
	}

	return strings.TrimSpace(fmt.Sprint(value))
}

func normalizeMergeMethod(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultMergeMethod, nil
	}

	switch trimmed {
	case mergeMethodMerge, mergeMethodSquash, mergeMethodRebase:
		return trimmed, nil
	default:
		return "", errors.New("merge method must be one of: merge, squash, rebase")
	}
}
