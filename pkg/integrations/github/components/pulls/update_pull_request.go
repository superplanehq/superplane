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

const (
	pullRequestStateOpen   = "open"
	pullRequestStateClosed = "closed"
)

type UpdatePullRequest struct{}

type UpdatePullRequestConfiguration struct {
	Repository string   `mapstructure:"repository" json:"repository"`
	PullNumber any      `mapstructure:"pullNumber" json:"pullNumber"`
	Title      string   `mapstructure:"title" json:"title"`
	Body       string   `mapstructure:"body" json:"body"`
	State      string   `mapstructure:"state" json:"state"`
	Base       string   `mapstructure:"base" json:"base"`
	Assignees  []string `mapstructure:"assignees" json:"assignees"`
	Labels     []string `mapstructure:"labels" json:"labels"`
}

type updatePullRequestInput struct {
	Repository string
	PullNumber int
	Title      string
	Body       string
	State      string
	Base       string
	Assignees  []string
	Labels     []string
}

func (c *UpdatePullRequest) Name() string {
	return "github.updatePullRequest"
}

func (c *UpdatePullRequest) Label() string {
	return "Update Pull Request"
}

func (c *UpdatePullRequest) Description() string {
	return "Update a pull request's title, body, state, base branch, labels, or assignees in a GitHub repository"
}

func (c *UpdatePullRequest) Documentation() string {
	return `The Update Pull Request component updates an existing pull request in a GitHub repository. Use it instead of Update Issue when the change is about a pull request - GitHub models a pull request's title, body, state, labels, and assignees on its underlying issue, but "Update Issue" is confusing to reach for when working with pull requests.

## Use Cases

- **Retitle/rebody automation**: Update a pull request's title or description as a workflow progresses
- **Open/close automation**: Close a pull request when it is superseded, or reopen one automatically
- **Retargeting**: Change the base branch a pull request merges into
- **Label and assignee management**: Replace a pull request's labels or assignees from a workflow

## Configuration

- **Repository**: Select the GitHub repository containing the pull request
- **Pull Request Number**: Pull request number to update. Supports expressions.
- **Title**: New title for the pull request (optional; leaves the title unchanged if empty)
- **Body**: New body/description for the pull request (optional; leaves the body unchanged if empty)
- **State**: Change the pull request state to "open" or "closed" (optional; unset leaves the state unchanged)
- **Base Branch**: Retarget the pull request onto a different base branch (optional)
- **Assignees**: Non-empty list replaces the full set of assignees on the pull request (optional)
- **Labels**: Non-empty list replaces the full set of labels on the pull request (optional)

At least one of title, body, state, base, assignees, or labels is required.

## Out of scope

The following pull request operations have their own dedicated components:

- Marking a draft pull request ready for review: use Mark Pull Request Ready for Review
- Adding reviewers: use Add Pull Request Reviewers
- Merging: use Merge Pull Request

## Implementation notes

- Title, body, state, and base are updated through GitHub's Pulls API.
- Assignees and labels are updated through GitHub's Issues API, since GitHub models a pull request's labels and assignees on its underlying issue.
- GitHub ignores a base branch change in the same request that sets the state to "closed".

## Output

Returns the updated pull request object with all current information.`
}

func (c *UpdatePullRequest) Icon() string {
	return "github"
}

func (c *UpdatePullRequest) Color() string {
	return "gray"
}

func (c *UpdatePullRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdatePullRequest) Configuration() []configuration.Field {
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
			Description: "Pull request number to update. Supports expressions.",
		},
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "New title for the pull request. Leave empty to keep the current title.",
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "New body for the pull request. Leave empty to keep the current body.",
		},
		{
			Name:        "state",
			Label:       "State",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Change the pull request state. Leave unset to keep the current state.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Open", Value: pullRequestStateOpen},
						{Label: "Closed", Value: pullRequestStateClosed},
					},
				},
			},
		},
		{
			Name:        "base",
			Label:       "Base Branch",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Retarget the pull request onto a different base branch. Must live in the selected repository.",
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
			Name:        "assignees",
			Label:       "Assignees",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Non-empty list replaces the full set of assignees on the pull request.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Assignee",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Non-empty list replaces the full set of labels on the pull request.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
	}
}

func (c *UpdatePullRequest) Setup(ctx core.SetupContext) error {
	var config UpdatePullRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateUpdatePullRequestSetup(config); err != nil {
		return err
	}

	return common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)
}

func (c *UpdatePullRequest) Execute(ctx core.ExecutionContext) error {
	var config UpdatePullRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	input, err := buildUpdatePullRequestInput(config)
	if err != nil {
		return err
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	//
	// Title, body, state, and base are updated through the Pulls API.
	//
	if input.hasPullRequestFields() {
		pullRequestUpdate := &github.PullRequest{}

		if input.Title != "" {
			pullRequestUpdate.Title = &input.Title
		}

		if input.Body != "" {
			pullRequestUpdate.Body = &input.Body
		}

		if input.State != "" {
			pullRequestUpdate.State = &input.State
		}

		if input.Base != "" {
			pullRequestUpdate.Base = &github.PullRequestBranch{Ref: &input.Base}
		}

		if _, _, err := client.EditPullRequest(context.Background(), input.Repository, input.PullNumber, pullRequestUpdate); err != nil {
			return fmt.Errorf("failed to update pull request: %w", explainGitHubError(err))
		}
	}

	//
	// GitHub models a pull request's labels and assignees on its underlying
	// issue, so those are updated through the Issues API instead.
	//
	if input.hasIssueFields() {
		issueRequest := &github.IssueRequest{}

		if len(input.Assignees) > 0 {
			issueRequest.Assignees = &input.Assignees
		}

		if len(input.Labels) > 0 {
			issueRequest.Labels = &input.Labels
		}

		if _, _, err := client.EditIssue(context.Background(), input.Repository, input.PullNumber, issueRequest); err != nil {
			return fmt.Errorf("failed to update pull request labels/assignees: %w", explainGitHubError(err))
		}
	}

	//
	// The Issues API response does not carry the full pull request shape, so
	// the pull request is re-fetched to emit an up-to-date object either way.
	//
	pullRequest, _, err := client.GetPullRequest(context.Background(), input.Repository, input.PullNumber)
	if err != nil {
		return fmt.Errorf("failed to get pull request: %w", explainGitHubError(err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.pullRequest",
		[]any{pullRequest},
	)
}

func (c *UpdatePullRequest) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdatePullRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *UpdatePullRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdatePullRequest) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *UpdatePullRequest) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdatePullRequest) Cleanup(ctx core.SetupContext) error {
	return nil
}

func validateUpdatePullRequestSetup(config UpdatePullRequestConfiguration) error {
	if strings.TrimSpace(config.Repository) == "" {
		return errors.New("repository is required")
	}

	if pullNumberText(config.PullNumber) == "" {
		return errors.New("pull request number is required")
	}

	if err := validatePullRequestState(config.State); err != nil {
		return err
	}

	if err := validateAtLeastOneUpdateField(config); err != nil {
		return err
	}

	if common.IsExpression(pullNumberText(config.PullNumber)) {
		return nil
	}

	_, err := parsePullNumber(config.PullNumber)
	return err
}

func buildUpdatePullRequestInput(config UpdatePullRequestConfiguration) (*updatePullRequestInput, error) {
	if strings.TrimSpace(config.Repository) == "" {
		return nil, errors.New("repository is required")
	}

	pullNumber, err := parsePullNumber(config.PullNumber)
	if err != nil {
		return nil, err
	}

	if err := validatePullRequestState(config.State); err != nil {
		return nil, err
	}

	if err := validateAtLeastOneUpdateField(config); err != nil {
		return nil, err
	}

	return &updatePullRequestInput{
		Repository: strings.TrimSpace(config.Repository),
		PullNumber: pullNumber,
		Title:      config.Title,
		Body:       config.Body,
		State:      strings.TrimSpace(config.State),
		Base:       strings.TrimSpace(config.Base),
		Assignees:  config.Assignees,
		Labels:     config.Labels,
	}, nil
}

func validatePullRequestState(state string) error {
	trimmed := strings.TrimSpace(state)
	if trimmed == "" {
		return nil
	}

	switch trimmed {
	case pullRequestStateOpen, pullRequestStateClosed:
		return nil
	default:
		return errors.New("state must be one of: open, closed")
	}
}

func validateAtLeastOneUpdateField(config UpdatePullRequestConfiguration) error {
	if strings.TrimSpace(config.Title) != "" ||
		strings.TrimSpace(config.Body) != "" ||
		strings.TrimSpace(config.State) != "" ||
		strings.TrimSpace(config.Base) != "" ||
		len(config.Assignees) > 0 ||
		len(config.Labels) > 0 {
		return nil
	}

	return errors.New("at least one of title, body, state, base, assignees, or labels is required")
}

func (i *updatePullRequestInput) hasPullRequestFields() bool {
	return i.Title != "" || i.Body != "" || i.State != "" || i.Base != ""
}

func (i *updatePullRequestInput) hasIssueFields() bool {
	return len(i.Assignees) > 0 || len(i.Labels) > 0
}
