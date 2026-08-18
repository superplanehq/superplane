package issues

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

type AddIssueReaction struct{}

type AddIssueReactionConfiguration struct {
	Repository  string `json:"repository" mapstructure:"repository"`
	IssueNumber string `json:"issueNumber" mapstructure:"issueNumber"`
	Content     string `json:"content" mapstructure:"content"`
}

func (c *AddIssueReaction) Name() string {
	return "github.addIssueReaction"
}

func (c *AddIssueReaction) Label() string {
	return "Add Issue Reaction"
}

func (c *AddIssueReaction) Description() string {
	return "Add a reaction to a GitHub issue"
}

func (c *AddIssueReaction) Documentation() string {
	return `The Add Issue Reaction component adds a reaction emoji to a GitHub issue.

## Use Cases

- **Acknowledge instructions**: Add eyes to an issue to indicate automation saw it
- **Workflow feedback**: React with +1 or rocket on success paths
- **Fast triage signals**: Show status without posting an extra comment

## Configuration

- **Repository**: Select the GitHub repository
- **Issue Number**: The issue number to react to (supports expressions)
- **Reaction**: One of GitHub's supported reaction values

## Output

Returns the created GitHub reaction object, including id, content, user, and timestamp.

Transient GitHub failures (timeouts, rate limits, and 5xx responses) are attempted up to three times using durable scheduled hooks. Permanent configuration and permission errors fail immediately.`
}

func (c *AddIssueReaction) Icon() string {
	return "github"
}

func (c *AddIssueReaction) Color() string {
	return "gray"
}

func (c *AddIssueReaction) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddIssueReaction) Configuration() []configuration.Field {
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
			Name:        "issueNumber",
			Label:       "Issue Number",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The issue number to react to",
		},
		{
			Name:     "content",
			Label:    "Reaction",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "eyes",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "+1", Value: "+1"},
						{Label: "-1", Value: "-1"},
						{Label: "laugh", Value: "laugh"},
						{Label: "confused", Value: "confused"},
						{Label: "heart", Value: "heart"},
						{Label: "hooray", Value: "hooray"},
						{Label: "rocket", Value: "rocket"},
						{Label: "eyes", Value: "eyes"},
					},
				},
			},
		},
	}
}

func (c *AddIssueReaction) Setup(ctx core.SetupContext) error {
	var config AddIssueReactionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.Repository) == "" {
		return errors.New("repository is required")
	}

	if strings.TrimSpace(config.IssueNumber) == "" {
		return errors.New("issue number is required")
	}

	if strings.TrimSpace(config.Content) == "" {
		return errors.New("reaction content is required")
	}
	if err := common.ValidateReactionContent(config.Content); err != nil {
		return err
	}

	return common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)
}

func (c *AddIssueReaction) Execute(ctx core.ExecutionContext) error {
	var config AddIssueReactionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	issueNumber, err := strconv.Atoi(strings.TrimSpace(config.IssueNumber))
	if err != nil {
		return fmt.Errorf("issue number is not a number: %v", err)
	}
	if issueNumber <= 0 {
		return errors.New("issue number must be positive")
	}
	if err := common.ValidateReactionContent(config.Content); err != nil {
		return err
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	err = common.ExecuteReaction(ctx, func() (*github.Reaction, *github.Response, error) {
		return client.CreateIssueReaction(context.Background(), config.Repository, issueNumber, config.Content)
	})
	if err != nil {
		return fmt.Errorf("failed to create issue reaction: %w", err)
	}

	return nil
}

func (c *AddIssueReaction) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AddIssueReaction) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *AddIssueReaction) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddIssueReaction) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AddIssueReaction) Hooks() []core.Hook {
	return common.ReactionHooks()
}

func (c *AddIssueReaction) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != common.ReactionRetryHookName {
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}

	var config AddIssueReactionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return common.FailReactionHook(ctx, fmt.Errorf("failed to decode configuration: %w", err))
	}

	issueNumber, err := strconv.Atoi(strings.TrimSpace(config.IssueNumber))
	if err != nil {
		return common.FailReactionHook(ctx, fmt.Errorf("issue number is not a number: %v", err))
	}
	if issueNumber <= 0 {
		return common.FailReactionHook(ctx, errors.New("issue number must be positive"))
	}
	if err := common.ValidateReactionContent(config.Content); err != nil {
		return common.FailReactionHook(ctx, err)
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return common.FailReactionHook(ctx, fmt.Errorf("failed to initialize GitHub client: %w", err))
	}

	err = common.RetryReaction(ctx, func() (*github.Reaction, *github.Response, error) {
		return client.CreateIssueReaction(context.Background(), config.Repository, issueNumber, config.Content)
	})
	if err != nil {
		return fmt.Errorf("failed to create issue reaction: %w", err)
	}

	return nil
}
