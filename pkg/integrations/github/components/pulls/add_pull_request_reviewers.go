package pulls

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type AddPullRequestReviewers struct{}

type AddPullRequestReviewersConfiguration struct {
	Repository    string   `mapstructure:"repository" json:"repository"`
	PullNumber    any      `mapstructure:"pullNumber" json:"pullNumber"`
	Reviewers     []string `mapstructure:"reviewers" json:"reviewers"`
	TeamReviewers []string `mapstructure:"teamReviewers" json:"teamReviewers"`
}

type addPullRequestReviewersInput struct {
	Repository    string
	PullNumber    int
	Reviewers     []string
	TeamReviewers []string
}

func (c *AddPullRequestReviewers) Name() string {
	return "github.addPullRequestReviewers"
}

func (c *AddPullRequestReviewers) Label() string {
	return "Add Pull Request Reviewers"
}

func (c *AddPullRequestReviewers) Description() string {
	return "Add reviewers to a GitHub pull request"
}

func (c *AddPullRequestReviewers) Documentation() string {
	return `The Add Pull Request Reviewers component adds individual users and/or teams as reviewers on a GitHub pull request.

## Use Cases

- **Automated reviewer assignment**: Request review from the relevant team when a pull request is opened
- **Escalation workflows**: Add additional reviewers when a pull request needs attention
- **On-call routing**: Request review from the current on-call engineer after checks pass

## Configuration

- **Repository**: Select the GitHub repository containing the pull request
- **Pull Request Number**: Pull request number to add reviewers to. Expressions are supported.
- **Reviewers**: GitHub usernames to request as reviewers
- **Team Reviewers**: Team slugs to request as team reviewers (optional)

At least one reviewer or team reviewer is required.

## Output

Returns the updated pull request object with the current requested reviewers.`
}

func (c *AddPullRequestReviewers) Icon() string {
	return "github"
}

func (c *AddPullRequestReviewers) Color() string {
	return "gray"
}

func (c *AddPullRequestReviewers) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddPullRequestReviewers) Configuration() []configuration.Field {
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
			Description: "Pull request number to add reviewers to. Supports expressions.",
		},
		{
			Name:        "reviewers",
			Label:       "Reviewers",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "GitHub usernames to request as reviewers (e.g. octocat)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Reviewer",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "teamReviewers",
			Label:       "Team Reviewers",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Team slugs to request as team reviewers (e.g. justice-league)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Team",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
	}
}

func (c *AddPullRequestReviewers) Setup(ctx core.SetupContext) error {
	var config AddPullRequestReviewersConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateAddPullRequestReviewersSetup(config); err != nil {
		return err
	}

	return common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)
}

func (c *AddPullRequestReviewers) Execute(ctx core.ExecutionContext) error {
	var config AddPullRequestReviewersConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	input, err := buildAddPullRequestReviewersInput(config)
	if err != nil {
		return err
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	request := github.ReviewersRequest{
		Reviewers:     input.Reviewers,
		TeamReviewers: input.TeamReviewers,
	}

	pullRequest, _, err := client.AddPullRequestReviewers(
		context.Background(),
		input.Repository,
		input.PullNumber,
		request,
	)
	if err != nil {
		return fmt.Errorf("failed to add pull request reviewers: %w", explainGitHubError(err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.pullRequest",
		[]any{pullRequest},
	)
}

func (c *AddPullRequestReviewers) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AddPullRequestReviewers) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *AddPullRequestReviewers) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AddPullRequestReviewers) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *AddPullRequestReviewers) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddPullRequestReviewers) Cleanup(ctx core.SetupContext) error {
	return nil
}

func validateAddPullRequestReviewersSetup(config AddPullRequestReviewersConfiguration) error {
	if strings.TrimSpace(config.Repository) == "" {
		return errors.New("repository is required")
	}

	if pullNumberText(config.PullNumber) == "" {
		return errors.New("pull request number is required")
	}

	if err := validateReviewerLists(config.Reviewers, config.TeamReviewers); err != nil {
		return err
	}

	if common.IsExpression(pullNumberText(config.PullNumber)) {
		return nil
	}

	_, err := parsePullNumber(config.PullNumber)
	return err
}

func buildAddPullRequestReviewersInput(config AddPullRequestReviewersConfiguration) (*addPullRequestReviewersInput, error) {
	if strings.TrimSpace(config.Repository) == "" {
		return nil, errors.New("repository is required")
	}

	pullNumber, err := parsePullNumber(config.PullNumber)
	if err != nil {
		return nil, err
	}

	reviewers, teamReviewers, err := normalizeReviewerLists(config.Reviewers, config.TeamReviewers)
	if err != nil {
		return nil, err
	}

	return &addPullRequestReviewersInput{
		Repository:    strings.TrimSpace(config.Repository),
		PullNumber:    pullNumber,
		Reviewers:     reviewers,
		TeamReviewers: teamReviewers,
	}, nil
}

func validateReviewerLists(reviewers, teamReviewers []string) error {
	normalizedReviewers, normalizedTeams, err := normalizeReviewerLists(reviewers, teamReviewers)
	if err != nil {
		return err
	}

	if len(normalizedReviewers) == 0 && len(normalizedTeams) == 0 {
		return errors.New("at least one reviewer or team reviewer is required")
	}

	return nil
}

func normalizeReviewerLists(reviewers, teamReviewers []string) ([]string, []string, error) {
	normalizedReviewers := sanitizeReviewerUsernames(reviewers)
	normalizedTeams := sanitizeTeamSlugs(teamReviewers)

	if len(normalizedReviewers) == 0 && len(normalizedTeams) == 0 {
		return nil, nil, errors.New("at least one reviewer or team reviewer is required")
	}

	return normalizedReviewers, normalizedTeams, nil
}

func sanitizeReviewerUsernames(reviewers []string) []string {
	sanitized := common.SanitizeAssignees(reviewers)
	result := make([]string, 0, len(sanitized))
	for _, reviewer := range sanitized {
		trimmed := strings.TrimSpace(reviewer)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func sanitizeTeamSlugs(teamReviewers []string) []string {
	result := make([]string, 0, len(teamReviewers))
	for _, team := range teamReviewers {
		trimmed := strings.TrimSpace(team)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
