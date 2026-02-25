package github

import (
	"fmt"
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnPRComment struct{}

func (p *OnPRComment) Name() string {
	return "github.onPRComment"
}

func (p *OnPRComment) Label() string {
	return "On PR Comment"
}

func (p *OnPRComment) Description() string {
	return "Listen to PR conversation comment events"
}

func (p *OnPRComment) Documentation() string {
	return `The On PR Comment trigger starts a workflow execution when comments are added on a pull request conversation.

## Use Cases

- **Command processing**: Process slash commands in PR conversation comments (e.g., /deploy, /test)
- **Bot interactions**: Respond to comments with automated actions
- **Notification systems**: Notify teams when important PR conversation comments are added

## Configuration

- **Repository**: Select the GitHub repository to monitor
- **Content Filter**: Optional regex pattern to filter comments (e.g., ` + "`/solve`" + ` to only trigger on comments containing "/solve")

## Event Data

Each comment event includes:
- **comment**: Comment information including body, author, created timestamp
- **issue**: Issue/PR information; for this trigger it is always a pull request issue
- **repository**: Repository information
- **sender**: User who added the comment

SuperPlane passes through the full GitHub webhook payload under data for the issue_comment event type.

Common expression paths:
- PR number: ` + "`root().data.issue.number`" + `
- PR title: ` + "`root().data.issue.title`" + `
- PR URL: ` + "`root().data.issue.pull_request.html_url`" + `
- Comment body: ` + "`root().data.comment.body`" + `

## Webhook Setup

This trigger automatically sets up a GitHub webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (p *OnPRComment) Icon() string {
	return "github"
}

func (p *OnPRComment) Color() string {
	return "gray"
}

func (p *OnPRComment) Configuration() []configuration.Field {
	return prCommentConfigurationFields()
}

func (p *OnPRComment) Setup(ctx core.TriggerContext) error {
	return setupPRCommentTrigger(ctx, WebhookConfiguration{EventType: "issue_comment"})
}

func (p *OnPRComment) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnPRComment) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPRComment) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config, err := decodePRCommentConfiguration(ctx.Configuration)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	eventType, err := extractGitHubEventType(ctx.Headers)
	if err != nil {
		return http.StatusBadRequest, err
	}

	if eventType != "issue_comment" {
		return http.StatusOK, nil
	}

	data, code, err := verifyAndParseWebhookData(ctx)
	if err != nil {
		return code, err
	}

	if !isPRIssueComment(data) {
		return http.StatusOK, nil
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

	if err := ctx.Events.Emit("github.prComment", data); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (p *OnPRComment) Cleanup(ctx core.TriggerContext) error {
	return nil
}
