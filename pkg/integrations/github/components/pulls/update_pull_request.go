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
	Repository string    `mapstructure:"repository" json:"repository"`
	PullNumber any       `mapstructure:"pullNumber" json:"pullNumber"`
	Title      *string   `mapstructure:"title" json:"title"`
	Body       *string   `mapstructure:"body" json:"body"`
	State      *string   `mapstructure:"state" json:"state"`
	Base       *string   `mapstructure:"base" json:"base"`
	Assignees  *[]string `mapstructure:"assignees" json:"assignees"`
	Labels     *[]string `mapstructure:"labels" json:"labels"`
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
- **Title**, **Body**, **State**, **Base Branch**, **Replace Assignees**, **Replace Labels**: each field has its own toggle. Only toggled-on fields are changed; toggled-off fields are left untouched on the pull request.

At least one field must be toggled on.

## Replacing labels and assignees

**Replace Assignees** and **Replace Labels** replace the pull request's full set when toggled on, including replacing it with an empty list to clear all assignees or labels.

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
			Togglable:   true,
			Description: "New title for the pull request.",
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Description: "New body for the pull request.",
		},
		{
			Name:        "state",
			Label:       "State",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Description: "Change the pull request state.",
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
			Togglable:   true,
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
			Label:       "Replace Assignees",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Replace the pull request's assignees with this list. An empty list clears all assignees.",
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
			Label:       "Replace Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Replace the pull request's labels with this list. An empty list clears all labels.",
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

	if err := validateUpdatePullRequestConfiguration(config); err != nil {
		return err
	}

	if !common.IsExpression(pullNumberText(config.PullNumber)) {
		if _, err := parsePullNumber(config.PullNumber); err != nil {
			return err
		}
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

	if err := validateUpdatePullRequestConfiguration(config); err != nil {
		return err
	}

	repository := strings.TrimSpace(config.Repository)
	pullNumber, err := parsePullNumber(config.PullNumber)
	if err != nil {
		return err
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	editPullRequest := func(update *github.PullRequest) error {
		if _, _, err := client.EditPullRequest(context.Background(), repository, pullNumber, update); err != nil {
			return fmt.Errorf("failed to update pull request: %w", explainGitHubError(err))
		}
		return nil
	}

	// Base is always sent in its own request, separate from title/body/state:
	// GitHub's Pulls API silently drops a base change bundled with a request
	// that also closes the pull request, and separately rejects retargeting a
	// pull request that is closed. Whether the base request runs before or
	// after the title/body/state request depends on which direction the pull
	// request is moving: reopening must happen before the retarget (since a
	// closed pull request can't be retargeted), while closing must happen
	// after it (since retargeting is silently dropped from a closing request).
	var baseUpdate *github.PullRequest
	if config.Base != nil {
		base := strings.TrimSpace(*config.Base)
		baseUpdate = &github.PullRequest{Base: &github.PullRequestBranch{Ref: &base}}
	}

	var titleBodyStateUpdate *github.PullRequest
	if config.Title != nil || config.Body != nil || config.State != nil {
		titleBodyStateUpdate = &github.PullRequest{
			Title: config.Title,
			Body:  config.Body,
			State: config.State,
		}
	}

	reopening := config.State != nil && strings.TrimSpace(*config.State) == pullRequestStateOpen

	if reopening && titleBodyStateUpdate != nil {
		if err := editPullRequest(titleBodyStateUpdate); err != nil {
			return err
		}
	}

	if baseUpdate != nil {
		if err := editPullRequest(baseUpdate); err != nil {
			return err
		}
	}

	if !reopening && titleBodyStateUpdate != nil {
		if err := editPullRequest(titleBodyStateUpdate); err != nil {
			return err
		}
	}

	//
	// GitHub models a pull request's labels and assignees on its underlying
	// issue, so those are updated through the Issues API instead.
	//
	if hasIssueFields(config) {
		// The Issues API succeeds on any issue number, whether or not it's
		// actually a pull request. When no Pulls API call happened above,
		// nothing has confirmed pullNumber refers to a pull request yet, so
		// check first rather than mutating a plain issue's labels/assignees
		// only to fail once the pull request re-fetch below 404s.
		if titleBodyStateUpdate == nil && baseUpdate == nil {
			if _, _, err := client.GetPullRequest(context.Background(), repository, pullNumber); err != nil {
				return fmt.Errorf("failed to get pull request: %w", explainGitHubError(err))
			}
		}

		issueRequest := &github.IssueRequest{
			Labels: config.Labels,
		}

		if config.Assignees != nil {
			assignees := common.SanitizeAssignees(*config.Assignees)
			issueRequest.Assignees = &assignees
		}

		if _, _, err := client.EditIssue(context.Background(), repository, pullNumber, issueRequest); err != nil {
			return fmt.Errorf("failed to update pull request labels/assignees: %w", explainGitHubError(err))
		}
	}

	//
	// The Issues API response does not carry the full pull request shape, so
	// the pull request is re-fetched to emit an up-to-date object either way.
	//
	pullRequest, _, err := client.GetPullRequest(context.Background(), repository, pullNumber)
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

func validateUpdatePullRequestConfiguration(config UpdatePullRequestConfiguration) error {
	if strings.TrimSpace(config.Repository) == "" {
		return errors.New("repository is required")
	}

	if pullNumberText(config.PullNumber) == "" {
		return errors.New("pull request number is required")
	}

	// Unlike body, an empty title or base is never a valid "clear" instruction:
	// GitHub rejects an empty pull request title outright, and an empty base
	// branch cannot resolve to a real branch to retarget onto.
	if config.Title != nil && strings.TrimSpace(*config.Title) == "" {
		return errors.New("title cannot be empty")
	}

	if config.Base != nil && strings.TrimSpace(*config.Base) == "" {
		return errors.New("base branch cannot be empty")
	}

	if err := validatePullRequestState(config.State); err != nil {
		return err
	}

	return validateAtLeastOneUpdateField(config)
}

func validatePullRequestState(state *string) error {
	if state == nil {
		return nil
	}

	switch strings.TrimSpace(*state) {
	case pullRequestStateOpen, pullRequestStateClosed:
		return nil
	default:
		return errors.New("state must be one of: open, closed")
	}
}

// validateAtLeastOneUpdateField requires at least one togglable field to be
// toggled on. A field counts as toggled on when its key is present in the
// submitted configuration (decoded as a non-nil pointer), regardless of
// whether its value is empty - an empty list for assignees/labels is a valid
// "clear" instruction, not an unset field.
func validateAtLeastOneUpdateField(config UpdatePullRequestConfiguration) error {
	if hasPullRequestFields(config) || hasIssueFields(config) {
		return nil
	}

	return errors.New("at least one of title, body, state, base, assignees, or labels is required")
}

func hasPullRequestFields(config UpdatePullRequestConfiguration) bool {
	return config.Title != nil || config.Body != nil || config.State != nil || config.Base != nil
}

func hasIssueFields(config UpdatePullRequestConfiguration) bool {
	return config.Assignees != nil || config.Labels != nil
}
