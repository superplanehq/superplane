package pulls

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type OnPRReview struct{}

func (p *OnPRReview) Name() string {
	return "github.onPRReview"
}

func (p *OnPRReview) Label() string {
	return "On PR Review"
}

func (p *OnPRReview) Description() string {
	return "Listen to submitted pull request reviews"
}

func (p *OnPRReview) Documentation() string {
	return `The On PR Review trigger starts a workflow execution when a pull request review is submitted.

It loads every inline comment that belongs to that review before it emits an event.
A review with several comments therefore produces one run, not one run per comment.

## Use Cases

- **Review automation**: Address a submitted review as one unit of work
- **Factory feedback**: Start one agent run when a reviewer mentions the factory
- **Review notifications**: Notify a team when a review is submitted

## Configuration

- **Repository**: Select the GitHub repository to monitor
- **Content Filter**: Optional filter on the review summary and inline comments. Mentions that start with @ match as an exact GitHub username. Other values are regular expressions.
- **Ignore Bots**: Skip reviews written by GitHub Apps and bots
- **Allowed Bots**: React to reviews from these GitHub Apps or bots even when the review does not match the content filter. Enter the bot login, for example coderabbitai.

## Event Data

This trigger handles the GitHub ` + "`pull_request_review`" + ` event with action ` + "`submitted`" + `.

SuperPlane passes through the full GitHub webhook payload under data and adds:

- **review_comments**: Inline comments fetched for the submitted review

Common expression paths:
- PR number: ` + "`root().data.pull_request.number`" + `
- PR URL: ` + "`root().data.pull_request.html_url`" + `
- Review body: ` + "`root().data.review.body`" + `
- Review comments: ` + "`root().data.review_comments`" + `

## Webhook Setup

This trigger automatically sets up a GitHub webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (p *OnPRReview) Icon() string {
	return "github"
}

func (p *OnPRReview) Color() string {
	return "gray"
}

func (p *OnPRReview) Configuration() []configuration.Field {
	return prCommentConfigurationFields()
}

func (p *OnPRReview) Setup(ctx core.TriggerContext) error {
	return setupPRCommentTrigger(ctx, common.WebhookConfiguration{EventType: "pull_request_review"})
}

func (p *OnPRReview) Hooks() []core.Hook {
	return []core.Hook{}
}

func (p *OnPRReview) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPRReview) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	ctx = common.WithWebhookLogger(ctx, p.Name())
	ctx.Logger.Infof("Received GitHub webhook")

	config, err := decodePRCommentConfiguration(ctx.Configuration)
	if err != nil {
		ctx.Logger.Errorf("Failed to decode configuration: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType, err := extractGitHubEventType(ctx.Headers)
	if err != nil {
		ctx.Logger.Errorf("Failed to extract GitHub event type: %v", err)
		return http.StatusBadRequest, nil, fmt.Errorf("failed to extract GitHub event type: %w", err)
	}

	if eventType != "pull_request_review" {
		ctx.Logger.Infof("Ignoring event - event type %q is not a pull_request_review event", eventType)
		return http.StatusOK, nil, nil
	}

	data, code, err := verifyAndParseWebhookData(ctx)
	if err != nil {
		ctx.Logger.Errorf("Failed to verify and parse webhook data: %v", err)
		return code, nil, fmt.Errorf("failed to verify and parse webhook data: %w", err)
	}

	if !isExpectedPRCommentAction(eventType, data) {
		action, _ := common.ExtractAction(data)
		ctx.Logger.Infof("Ignoring event - action %q is not supported", action)
		return http.StatusOK, nil, nil
	}

	allowed := isAllowedBot(eventType, data, config.AllowedBots)
	if config.IgnoreBots && isBotAuthor(eventType, data) && !allowed {
		ctx.Logger.Info("Ignoring event - author is a bot")
		return http.StatusOK, nil, nil
	}

	comments, code, err := loadSubmittedReviewComments(ctx, config.Repository, data)
	if err != nil {
		ctx.Logger.Errorf("Failed to load review comments: %v", err)
		return code, nil, err
	}

	matched := allowed
	if !allowed {
		bodies := reviewFilterBodies(data, comments)
		matched, code, err = applyContentFilter(config.ContentFilter, bodies...)
		if err != nil {
			ctx.Logger.Errorf("Failed to apply PR review content filter: %v", err)
			return code, nil, fmt.Errorf("failed to apply PR review content filter: %w", err)
		}
	} else {
		ctx.Logger.Info("Author is an allowed bot - bypassing content filter")
	}

	if !matched {
		ctx.Logger.Info("Ignoring event - content filter did not match")
		return http.StatusOK, nil, nil
	}

	commentMaps, err := reviewCommentsAsMaps(comments)
	if err != nil {
		ctx.Logger.Errorf("Failed to encode review comments: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to encode review comments: %w", err)
	}
	data["review_comments"] = commentMaps

	if err := ctx.Events.Emit("github.prReview", data); err != nil {
		ctx.Logger.Errorf("Failed to emit event: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (p *OnPRReview) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func loadSubmittedReviewComments(
	ctx core.WebhookRequestContext,
	configuredRepository string,
	data map[string]any,
) ([]*github.PullRequestComment, int, error) {
	review, ok := data["review"].(map[string]any)
	if !ok {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid review structure")
	}

	reviewID, ok := int64FromJSON(review["id"])
	if !ok {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid review id")
	}

	pullRequest, ok := data["pull_request"].(map[string]any)
	if !ok {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid pull request structure")
	}

	pullNumber, ok := int64FromJSON(pullRequest["number"])
	if !ok {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid pull request number")
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	repository := webhookRepositoryFullName(data, configuredRepository)
	comments, err := client.ListPullRequestReviewComments(
		context.Background(),
		repository,
		int(pullNumber),
		reviewID,
	)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load review comments: %w", err)
	}

	return comments, http.StatusOK, nil
}

func reviewFilterBodies(data map[string]any, comments []*github.PullRequestComment) []string {
	bodies := make([]string, 0, len(comments)+1)
	if body, err := extractPRCommentBody("pull_request_review", data); err == nil {
		bodies = append(bodies, body)
	}

	for _, comment := range comments {
		if comment == nil {
			continue
		}
		bodies = append(bodies, comment.GetBody())
	}

	return bodies
}

func reviewCommentsAsMaps(comments []*github.PullRequestComment) ([]any, error) {
	encoded, err := json.Marshal(comments)
	if err != nil {
		return nil, err
	}

	var maps []any
	if err := json.Unmarshal(encoded, &maps); err != nil {
		return nil, err
	}
	if maps == nil {
		return []any{}, nil
	}

	return maps, nil
}
