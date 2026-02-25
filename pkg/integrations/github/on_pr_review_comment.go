package github

import (
	"fmt"
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnPRReviewComment struct{}

func (p *OnPRReviewComment) Name() string {
	return "github.onPRReviewComment"
}

func (p *OnPRReviewComment) Label() string {
	return "On PR Review Comment"
}

func (p *OnPRReviewComment) Description() string {
	return "Listen to pull request review comment events"
}

func (p *OnPRReviewComment) Documentation() string {
	return `The On PR Review Comment trigger starts a workflow execution when review comments are added to pull requests.

## Use Cases

- **Code review automation**: React to line-level review comments
- **Review workflows**: Trigger follow-up workflows when a review is submitted
- **Notification systems**: Notify teams when new review comments are posted

## Configuration

- **Repository**: Select the GitHub repository to monitor
- **Content Filter**: Optional regex pattern to filter comment/review body (e.g., ` + "`/solve`" + `)

## Event Data

This trigger handles two GitHub webhook events:
- **pull_request_review_comment**: line-level code review comments (` + "`comment`" + ` and ` + "`pull_request`" + `)
- **pull_request_review**: submitted review comments (` + "`review`" + ` and ` + "`pull_request`" + `)

SuperPlane passes through the full GitHub webhook payload under data.

Common expression paths:
- PR number: ` + "`root().data.pull_request.number`" + `
- Branch name: ` + "`root().data.pull_request.head.ref`" + `
- Head SHA: ` + "`root().data.pull_request.head.sha`" + `
- Review comment body: ` + "`root().data.comment.body`" + ` (pull_request_review_comment)
- Review submission body: ` + "`root().data.review.body`" + ` (pull_request_review)

## Webhook Setup

This trigger automatically sets up a GitHub webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (p *OnPRReviewComment) Icon() string {
	return "github"
}

func (p *OnPRReviewComment) Color() string {
	return "gray"
}

func (p *OnPRReviewComment) Configuration() []configuration.Field {
	return prCommentConfigurationFields()
}

func (p *OnPRReviewComment) Setup(ctx core.TriggerContext) error {
	return setupPRCommentTrigger(ctx, WebhookConfiguration{
		EventTypes: []string{"pull_request_review_comment", "pull_request_review"},
	})
}

func (p *OnPRReviewComment) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnPRReviewComment) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPRReviewComment) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config, err := decodePRCommentConfiguration(ctx.Configuration)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	eventType, err := extractGitHubEventType(ctx.Headers)
	if err != nil {
		return http.StatusBadRequest, err
	}

	if eventType != "pull_request_review_comment" && eventType != "pull_request_review" {
		return http.StatusOK, nil
	}

	data, code, err := verifyAndParseWebhookData(ctx)
	if err != nil {
		return code, err
	}

	if !isExpectedPRCommentAction(eventType, data) {
		return http.StatusOK, nil
	}

	matched, code, err := applyPRCommentContentFilter(config.ContentFilter, eventType, data)
	if err != nil {
		return code, err
	}

	if !matched {
		return http.StatusOK, nil
	}

	if err := ctx.Events.Emit("github.prReviewComment", data); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (p *OnPRReviewComment) Cleanup(ctx core.TriggerContext) error {
	return nil
}
