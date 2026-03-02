package github

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type AddReaction struct{}

const (
	ReactionTargetIssueComment  = "issueComment"
	ReactionTargetReviewComment = "reviewComment"
)

type AddReactionConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Target     string `json:"target" mapstructure:"target"`
	CommentID  string `json:"commentId" mapstructure:"commentId"`
	Content    string `json:"content" mapstructure:"content"`
}

func (c *AddReaction) Name() string {
	return "github.addReaction"
}

func (c *AddReaction) Label() string {
	return "Add Reaction"
}

func (c *AddReaction) Description() string {
	return "Add a reaction to a GitHub comment"
}

func (c *AddReaction) Documentation() string {
	return `The Add Reaction component adds a reaction emoji to a GitHub comment.

## Use Cases

- **Acknowledge commands**: Add eyes to PR comments to indicate automation saw them
- **Workflow feedback**: React with +1 or rocket on success paths
- **Fast triage signals**: Use reactions to show status without posting extra comments

## Configuration

- **Repository**: Select the GitHub repository
- **Target**: Choose PR conversation comment or PR review line comment
- **Comment ID**: The GitHub comment ID to react to (supports expressions)
- **Reaction**: One of GitHub's supported reaction values

## Output

Returns the created GitHub reaction object, including id, content, user, and timestamp.`
}

func (c *AddReaction) Icon() string {
	return "github"
}

func (c *AddReaction) Color() string {
	return "gray"
}

func (c *AddReaction) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddReaction) Configuration() []configuration.Field {
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
			Name:     "target",
			Label:    "Target",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  ReactionTargetIssueComment,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "PR conversation comment", Value: ReactionTargetIssueComment},
						{Label: "PR review line comment", Value: ReactionTargetReviewComment},
					},
				},
			},
		},
		{
			Name:        "commentId",
			Label:       "Comment ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "ID of the comment to react to",
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

func (c *AddReaction) Setup(ctx core.SetupContext) error {
	var config AddReactionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository == "" {
		return errors.New("repository is required")
	}

	if strings.TrimSpace(config.CommentID) == "" {
		return errors.New("comment ID is required")
	}

	if config.Content == "" {
		return errors.New("reaction content is required")
	}

	if config.Target == "" {
		return errors.New("target is required")
	}

	if config.Target != ReactionTargetIssueComment && config.Target != ReactionTargetReviewComment {
		return fmt.Errorf("invalid target: %s", config.Target)
	}

	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *AddReaction) Execute(ctx core.ExecutionContext) error {
	var config AddReactionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Target != ReactionTargetIssueComment && config.Target != ReactionTargetReviewComment {
		return fmt.Errorf("invalid target: %s", config.Target)
	}

	commentID, err := parseCommentID(config.CommentID)
	if err != nil {
		return fmt.Errorf("comment ID is not a number: %v", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	reactionRequest := map[string]string{
		"content": config.Content,
	}

	var requestPath string
	switch config.Target {
	case ReactionTargetIssueComment:
		requestPath = fmt.Sprintf("repos/%s/%s/issues/comments/%d/reactions", appMetadata.Owner, config.Repository, commentID)
	case ReactionTargetReviewComment:
		requestPath = fmt.Sprintf("repos/%s/%s/pulls/comments/%d/reactions", appMetadata.Owner, config.Repository, commentID)
	}

	request, err := client.NewRequest("POST", requestPath, reactionRequest)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	reaction := &github.Reaction{}
	_, err = client.Do(context.Background(), request, reaction)
	if err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.reaction",
		[]any{reaction},
	)
}

func (c *AddReaction) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AddReaction) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *AddReaction) Actions() []core.Action {
	return []core.Action{}
}

func (c *AddReaction) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *AddReaction) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddReaction) Cleanup(ctx core.SetupContext) error {
	return nil
}

func parseCommentID(value string) (int64, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return 0, fmt.Errorf("value is empty")
	}

	commentID, err := strconv.ParseInt(trimmedValue, 10, 64)
	if err == nil {
		return commentID, nil
	}

	floatValue, floatErr := strconv.ParseFloat(trimmedValue, 64)
	if floatErr != nil {
		return 0, err
	}

	if math.IsNaN(floatValue) || math.IsInf(floatValue, 0) {
		return 0, fmt.Errorf("value is not finite")
	}

	if floatValue != math.Trunc(floatValue) {
		return 0, fmt.Errorf("value has decimals")
	}

	if floatValue > float64(math.MaxInt64) || floatValue < float64(math.MinInt64) {
		return 0, fmt.Errorf("value is out of range")
	}

	return int64(floatValue), nil
}
