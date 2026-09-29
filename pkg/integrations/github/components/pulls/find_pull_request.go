package pulls

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const (
	pullRequestStateAll = "all"

	FindPullRequestFoundChannel    = "found"
	FindPullRequestNotFoundChannel = "notFound"
)

type FindPullRequest struct{}

type FindPullRequestConfiguration struct {
	Repository string `mapstructure:"repository" json:"repository"`
	Head       string `mapstructure:"head" json:"head"`
	Base       string `mapstructure:"base" json:"base"`
	State      string `mapstructure:"state" json:"state"`
}

func (c *FindPullRequest) Name() string {
	return "github.findPullRequest"
}

func (c *FindPullRequest) Label() string {
	return "Find Pull Request"
}

func (c *FindPullRequest) Description() string {
	return "Find an existing pull request in a GitHub repository by branch"
}

func (c *FindPullRequest) Documentation() string {
	return `The Find Pull Request component looks up a pull request in a GitHub repository by its head branch, so a workflow can decide whether to create a new pull request or update one that already exists.

## Use Cases

- **Idempotent PR automation**: Check for an existing pull request before creating one, so a workflow that runs more than once does not create duplicates
- **Branch status checks**: Look up the pull request open for a branch to read its number, URL, or state

## Configuration

- **Repository**: Select the GitHub repository to search
- **Head Branch**: The branch containing the changes (source branch). Supply a plain branch name; the component qualifies it with the repository owner as GitHub's API requires.
- **Base Branch**: Optional. When set, only pull requests targeting this branch match.
- **State**: Which pull requests to consider - Open, Closed, or All. Defaults to Open.

## Output Channels

- **Found**: Emitted with the first matching pull request when one exists
- **Not Found**: Emitted when no pull request matches`
}

func (c *FindPullRequest) Icon() string {
	return "github"
}

func (c *FindPullRequest) Color() string {
	return "gray"
}

func (c *FindPullRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:        FindPullRequestFoundChannel,
			Label:       "Found",
			Description: "Emits the first matching pull request",
		},
		{
			Name:        FindPullRequestNotFoundChannel,
			Label:       "Not Found",
			Description: "Emits when no pull request matches",
		},
	}
}

func (c *FindPullRequest) Configuration() []configuration.Field {
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
			Description: "The branch containing the changes to look for. Supply a plain branch name; it is automatically qualified with the repository owner.",
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
			Required:    false,
			Description: "Optional. When set, only pull requests targeting this branch match.",
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
			Name:        "state",
			Label:       "State",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     pullRequestStateOpen,
			Description: "Which pull requests to consider.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Open", Value: pullRequestStateOpen},
						{Label: "Closed", Value: pullRequestStateClosed},
						{Label: "All", Value: pullRequestStateAll},
					},
				},
			},
		},
	}
}

func (c *FindPullRequest) Setup(ctx core.SetupContext) error {
	config, err := decodeFindPullRequestConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateFindPullRequestConfiguration(config); err != nil {
		return err
	}

	return common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)
}

func (c *FindPullRequest) Execute(ctx core.ExecutionContext) error {
	config, err := decodeFindPullRequestConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateFindPullRequestConfiguration(config); err != nil {
		return err
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	opts := &github.PullRequestListOptions{
		Head:  client.HeadFilter(config.Repository, config.Head),
		State: findPullRequestState(config.State),
	}

	if base := strings.TrimSpace(config.Base); base != "" {
		opts.Base = base
	}

	pullRequests, _, err := client.ListPullRequests(context.Background(), config.Repository, opts)
	if err != nil {
		return fmt.Errorf("failed to list pull requests: %w", explainGitHubError(err))
	}

	if len(pullRequests) == 0 {
		return ctx.ExecutionState.Emit(
			FindPullRequestNotFoundChannel,
			"github.findPullRequest.notFound",
			[]any{map[string]any{
				"repository": config.Repository,
				"head":       config.Head,
				"base":       config.Base,
				"state":      findPullRequestState(config.State),
			}},
		)
	}

	return ctx.ExecutionState.Emit(
		FindPullRequestFoundChannel,
		"github.pullRequest",
		[]any{pullRequests[0]},
	)
}

func (c *FindPullRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *FindPullRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *FindPullRequest) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *FindPullRequest) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *FindPullRequest) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeFindPullRequestConfiguration(configuration any) (FindPullRequestConfiguration, error) {
	var config FindPullRequestConfiguration
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return FindPullRequestConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	return config, nil
}

func validateFindPullRequestConfiguration(config FindPullRequestConfiguration) error {
	if strings.TrimSpace(config.Repository) == "" {
		return errors.New("repository is required")
	}

	if strings.TrimSpace(config.Head) == "" {
		return errors.New("head branch is required")
	}

	state := strings.TrimSpace(config.State)
	if state == "" || common.IsExpression(state) {
		return nil
	}

	switch state {
	case pullRequestStateOpen, pullRequestStateClosed, pullRequestStateAll:
		return nil
	default:
		return errors.New("state must be one of: open, closed, all")
	}
}

// findPullRequestState resolves the configured state to the value sent to
// GitHub, defaulting to open when unset.
func findPullRequestState(state string) string {
	state = strings.TrimSpace(state)
	if state == "" {
		return pullRequestStateOpen
	}

	return state
}
