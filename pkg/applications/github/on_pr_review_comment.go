package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnPullRequestReviewComment struct{}

type OnPullRequestReviewCommentConfiguration struct {
	Repository    string `json:"repository" mapstructure:"repository"`
	ContentFilter string `json:"contentFilter" mapstructure:"contentFilter"`
}

func (p *OnPullRequestReviewComment) Name() string {
	return "github.onPullRequestReviewComment"
}

func (p *OnPullRequestReviewComment) Label() string {
	return "On PR Review Comment"
}

func (p *OnPullRequestReviewComment) Description() string {
	return "Listen to pull request review comment events"
}

func (p *OnPullRequestReviewComment) Icon() string {
	return "github"
}

func (p *OnPullRequestReviewComment) Color() string {
	return "gray"
}

func (p *OnPullRequestReviewComment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeAppInstallationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "contentFilter",
			Label:       "Content Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., /solve",
			Description: "Optional regex pattern to filter comments by content",
		},
	}
}

func (p *OnPullRequestReviewComment) Setup(ctx core.TriggerContext) error {
	err := ensureRepoInMetadata(
		ctx.Metadata,
		ctx.AppInstallation,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnPullRequestReviewCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.AppInstallation.RequestWebhook(WebhookConfiguration{
		EventType:  "pull_request_review_comment",
		Repository: config.Repository,
	})
}

func (p *OnPullRequestReviewComment) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnPullRequestReviewComment) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPullRequestReviewComment) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnPullRequestReviewCommentConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing X-GitHub-Event header")
	}

	if eventType != "pull_request_review_comment" {
		return http.StatusOK, nil
	}

	code, err := verifySignature(ctx)
	if err != nil {
		return code, err
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// Only process "created" actions
	action, ok := data["action"]
	if !ok || action != "created" {
		return http.StatusOK, nil
	}

	// Apply content filter if configured
	if config.ContentFilter != "" {
		comment, ok := data["comment"].(map[string]any)
		if !ok {
			return http.StatusBadRequest, fmt.Errorf("invalid comment structure")
		}

		body, ok := comment["body"].(string)
		if !ok {
			return http.StatusBadRequest, fmt.Errorf("invalid comment body")
		}

		matched, err := regexp.MatchString(config.ContentFilter, body)
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("invalid regex pattern: %w", err)
		}

		if !matched {
			return http.StatusOK, nil
		}
	}

	err = ctx.Events.Emit("github.pullRequestReviewComment", data)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
