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

type RequestPullRequestReviewer struct{}

type RequestPullRequestReviewerConfiguration struct {
	Repository    string   `mapstructure:"repository" json:"repository"`
	PullNumber    any      `mapstructure:"pullNumber" json:"pullNumber"`
	Reviewers     []string `mapstructure:"reviewers" json:"reviewers"`
	TeamReviewers []string `mapstructure:"teamReviewers" json:"teamReviewers"`
}

type requestPullRequestReviewerInput struct {
	Repository    string
	PullNumber    int
	Reviewers     []string
	TeamReviewers []string
}

func (c *RequestPullRequestReviewer) Name() string {
	return "github.requestPullRequestReviewer"
}

func (c *RequestPullRequestReviewer) Label() string {
	return "Request Pull Request Reviewer"
}

func (c *RequestPullRequestReviewer) Description() string {
	return "Request reviewers on a GitHub pull request"
}

func (c *RequestPullRequestReviewer) Documentation() string {
	return `The Request Pull Request Reviewer component requests individual users and/or teams to review a GitHub pull request.

## Use Cases

- **Automated reviewer assignment**: Request review from the relevant team when a pull request is opened
- **Escalation workflows**: Add additional reviewers when a pull request needs attention
- **On-call routing**: Request review from the current on-call engineer after checks pass

## Configuration

- **Repository**: Select the GitHub repository containing the pull request
- **Pull Request Number**: Pull request number to request reviewers on. Expressions are supported.
- **Reviewers**: GitHub usernames to request as reviewers
- **Team Reviewers**: Team slugs to request as team reviewers (optional)

At least one reviewer or team reviewer is required.

## Output

Returns the updated pull request object with the current requested reviewers.`
}

func (c *RequestPullRequestReviewer) Icon() string {
	return "github"
}

func (c *RequestPullRequestReviewer) Color() string {
	return "gray"
}

func (c *RequestPullRequestReviewer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RequestPullRequestReviewer) Configuration() []configuration.Field {
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
			Description: "Pull request number to request reviewers on. Supports expressions.",
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

func (c *RequestPullRequestReviewer) Setup(ctx core.SetupContext) error {
	var config RequestPullRequestReviewerConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateRequestPullRequestReviewerSetup(config); err != nil {
		return err
	}

	return common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)
}

func (c *RequestPullRequestReviewer) Execute(ctx core.ExecutionContext) error {
	var config RequestPullRequestReviewerConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	input, err := buildRequestPullRequestReviewerInput(config)
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

	pullRequest, _, err := client.RequestPullRequestReviewers(
		context.Background(),
		input.Repository,
		input.PullNumber,
		request,
	)
	if err != nil {
		return fmt.Errorf("failed to request pull request reviewers: %w", explainGitHubError(err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.pullRequest",
		[]any{pullRequest},
	)
}

func (c *RequestPullRequestReviewer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RequestPullRequestReviewer) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *RequestPullRequestReviewer) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *RequestPullRequestReviewer) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *RequestPullRequestReviewer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RequestPullRequestReviewer) Cleanup(ctx core.SetupContext) error {
	return nil
}

func validateRequestPullRequestReviewerSetup(config RequestPullRequestReviewerConfiguration) error {
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

func buildRequestPullRequestReviewerInput(config RequestPullRequestReviewerConfiguration) (*requestPullRequestReviewerInput, error) {
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

	return &requestPullRequestReviewerInput{
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
