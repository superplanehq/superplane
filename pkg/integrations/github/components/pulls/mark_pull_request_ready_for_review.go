package pulls

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type MarkPullRequestReadyForReview struct{}

type MarkPullRequestReadyForReviewConfiguration struct {
	Repository string `mapstructure:"repository" json:"repository"`
	PullNumber any    `mapstructure:"pullNumber" json:"pullNumber"`
}

type markPullRequestReadyForReviewInput struct {
	Repository string
	PullNumber int
}

func (c *MarkPullRequestReadyForReview) Name() string {
	return "github.markPullRequestReadyForReview"
}

func (c *MarkPullRequestReadyForReview) Label() string {
	return "Mark Pull Request Ready for Review"
}

func (c *MarkPullRequestReadyForReview) Description() string {
	return "Take a draft pull request out of the draft state in a GitHub repository"
}

func (c *MarkPullRequestReadyForReview) Documentation() string {
	return `The Mark Pull Request Ready for Review component takes a draft pull request out of the draft state, the same as clicking "Ready for review" on GitHub.

## Use Cases

- **Promote drafts automatically**: Mark a draft pull request ready once CI checks pass
- **Release trains**: Open drafts early and promote them for review when the branch is ready
- **Bot workflows**: Let an automation open work as a draft and hand it to reviewers when complete

## Configuration

- **Repository**: Select the GitHub repository containing the pull request
- **Pull Request Number**: Pull request number to mark ready for review. Expressions are supported.

## Behavior

This component is idempotent: if the pull request is already out of the draft state, it succeeds without calling GitHub again and emits the pull request as it is.

## Permissions

GitHub only exposes this operation through its GraphQL API, so the integration needs the **Pull Requests: Read & Write** permission.

## Output

Returns the pull request object after it has been marked ready for review.`
}

func (c *MarkPullRequestReadyForReview) Icon() string {
	return "github"
}

func (c *MarkPullRequestReadyForReview) Color() string {
	return "gray"
}

func (c *MarkPullRequestReadyForReview) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *MarkPullRequestReadyForReview) Configuration() []configuration.Field {
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
			Description: "Pull request number to mark ready for review. Supports expressions.",
		},
	}
}

func (c *MarkPullRequestReadyForReview) Setup(ctx core.SetupContext) error {
	var config MarkPullRequestReadyForReviewConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateMarkPullRequestReadyForReviewSetup(config); err != nil {
		return err
	}

	return common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)
}

func (c *MarkPullRequestReadyForReview) Execute(ctx core.ExecutionContext) error {
	var config MarkPullRequestReadyForReviewConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	input, err := buildMarkPullRequestReadyForReviewInput(config)
	if err != nil {
		return err
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	pullRequest, _, err := client.GetPullRequest(context.Background(), input.Repository, input.PullNumber)
	if err != nil {
		return fmt.Errorf("failed to get pull request: %w", explainGitHubError(err))
	}

	//
	// Marking an already-ready pull request fails on GitHub's side, so a pull
	// request that is not a draft is emitted as-is to keep re-runs idempotent.
	//
	if pullRequest.GetDraft() {
		if pullRequest.GetNodeID() == "" {
			return errors.New("pull request is missing a node ID")
		}

		err = client.MarkPullRequestReadyForReview(context.Background(), pullRequest.GetNodeID())
		if err != nil {
			return fmt.Errorf("failed to mark pull request ready for review: %w", explainGitHubError(err))
		}

		//
		// The mutation response does not carry the full REST pull request shape,
		// so it is re-fetched to emit an up-to-date object.
		//
		pullRequest, _, err = client.GetPullRequest(context.Background(), input.Repository, input.PullNumber)
		if err != nil {
			return fmt.Errorf("failed to get pull request: %w", explainGitHubError(err))
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.pullRequest",
		[]any{pullRequest},
	)
}

func (c *MarkPullRequestReadyForReview) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *MarkPullRequestReadyForReview) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *MarkPullRequestReadyForReview) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *MarkPullRequestReadyForReview) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *MarkPullRequestReadyForReview) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *MarkPullRequestReadyForReview) Cleanup(ctx core.SetupContext) error {
	return nil
}

func validateMarkPullRequestReadyForReviewSetup(config MarkPullRequestReadyForReviewConfiguration) error {
	if strings.TrimSpace(config.Repository) == "" {
		return errors.New("repository is required")
	}

	if pullNumberText(config.PullNumber) == "" {
		return errors.New("pull request number is required")
	}

	if common.IsExpression(pullNumberText(config.PullNumber)) {
		return nil
	}

	_, err := parsePullNumber(config.PullNumber)
	return err
}

func buildMarkPullRequestReadyForReviewInput(config MarkPullRequestReadyForReviewConfiguration) (*markPullRequestReadyForReviewInput, error) {
	if strings.TrimSpace(config.Repository) == "" {
		return nil, errors.New("repository is required")
	}

	pullNumber, err := parsePullNumber(config.PullNumber)
	if err != nil {
		return nil, err
	}

	return &markPullRequestReadyForReviewInput{
		Repository: strings.TrimSpace(config.Repository),
		PullNumber: pullNumber,
	}, nil
}
