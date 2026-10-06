package bitbucket

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	FindPullRequestFoundChannel    = "found"
	FindPullRequestNotFoundChannel = "notFound"

	findPullRequestStateOpen     = "open"
	findPullRequestStateMerged   = "merged"
	findPullRequestStateDeclined = "declined"
	findPullRequestStateAll      = "all"
)

type FindPullRequest struct {
	pullRequestAction
}

type FindPullRequestConfiguration struct {
	Repository string `mapstructure:"repository" json:"repository"`
	Head       string `mapstructure:"head" json:"head"`
	Base       string `mapstructure:"base" json:"base"`
	State      string `mapstructure:"state" json:"state"`
}

func (c *FindPullRequest) Name() string {
	return "bitbucket.findPullRequest"
}

func (c *FindPullRequest) Label() string {
	return "Find Pull Request"
}

func (c *FindPullRequest) Description() string {
	return "Find a pull request in a Bitbucket repository by branch"
}

func (c *FindPullRequest) Documentation() string {
	return `The Find Pull Request component looks up a pull request in a Bitbucket repository by its source branch. Use it to decide if a workflow must create a new pull request or update an existing one.

## Configuration

- **Repository**: The Bitbucket repository, in workspace/repository format
- **Head Branch**: The source branch of the pull request
- **Base Branch**: Optional. When set, only pull requests into this branch match.
- **State**: Which pull requests to consider: Open, Merged, Declined, or All. Defaults to Open.

## Output Channels

- **Found**: Emits the first matching pull request
- **Not Found**: Emits when no pull request matches`
}

func (c *FindPullRequest) ExampleOutput() map[string]any {
	return examplePullRequestOutput()
}

func (c *FindPullRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: FindPullRequestFoundChannel, Label: "Found", Description: "Emits the first matching pull request"},
		{Name: FindPullRequestNotFoundChannel, Label: "Not Found", Description: "Emits when no pull request matches"},
	}
}

func (c *FindPullRequest) Configuration() []configuration.Field {
	return []configuration.Field{
		repositoryField(),
		{
			Name:        "head",
			Label:       "Head Branch",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The source branch of the pull request.",
		},
		{
			Name:        "base",
			Label:       "Base Branch",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional. When set, only pull requests into this branch match.",
		},
		{
			Name:     "state",
			Label:    "State",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  findPullRequestStateOpen,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Open", Value: findPullRequestStateOpen},
						{Label: "Merged", Value: findPullRequestStateMerged},
						{Label: "Declined", Value: findPullRequestStateDeclined},
						{Label: "All", Value: findPullRequestStateAll},
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
	if _, err := findPullRequestStates(config.State); err != nil && !isExpression(config.State) {
		return err
	}
	return setupRepository(ctx, config.Repository)
}

func (c *FindPullRequest) Execute(ctx core.ExecutionContext) error {
	config, err := decodeFindPullRequestConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	states, err := findPullRequestStates(config.State)
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

	pullRequests, err := client.ListPullRequests(config.Repository, PullRequestListOptions{
		SourceBranch:      strings.TrimSpace(config.Head),
		DestinationBranch: strings.TrimSpace(config.Base),
		States:            states,
	})
	if err != nil {
		return fmt.Errorf("failed to list pull requests: %w", err)
	}

	if len(pullRequests) == 0 {
		return ctx.ExecutionState.Emit(
			FindPullRequestNotFoundChannel,
			"bitbucket.findPullRequest.notFound",
			[]any{map[string]any{
				"repository": config.Repository,
				"head":       config.Head,
				"base":       config.Base,
				"state":      config.State,
			}},
		)
	}

	return ctx.ExecutionState.Emit(FindPullRequestFoundChannel, PayloadTypePullRequest, []any{pullRequests[0]})
}

func decodeFindPullRequestConfiguration(raw any) (FindPullRequestConfiguration, error) {
	config := FindPullRequestConfiguration{}
	if err := mapstructure.Decode(raw, &config); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(config.Repository) == "" {
		return config, errors.New("repository is required")
	}
	if strings.TrimSpace(config.Head) == "" {
		return config, errors.New("head branch is required")
	}
	return config, nil
}

func findPullRequestStates(state string) ([]string, error) {
	switch strings.TrimSpace(state) {
	case "", findPullRequestStateOpen:
		return []string{PullRequestStateOpen}, nil
	case findPullRequestStateMerged:
		return []string{PullRequestStateMerged}, nil
	case findPullRequestStateDeclined:
		return []string{PullRequestStateDeclined}, nil
	case findPullRequestStateAll:
		return []string{PullRequestStateOpen, PullRequestStateMerged, PullRequestStateDeclined}, nil
	default:
		return nil, errors.New("state must be one of: open, merged, declined, all")
	}
}
