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

type OnPRComment struct{}

type OnPRCommentConfiguration struct {
	Repository    string `json:"repository" mapstructure:"repository"`
	ContentFilter string `json:"contentFilter" mapstructure:"contentFilter"`
}

func (p *OnPRComment) Name() string {
	return "github.onPRComment"
}

func (p *OnPRComment) Label() string {
	return "On PR Comment"
}

func (p *OnPRComment) Description() string {
	return "Listen to all comment events on pull requests"
}

func (p *OnPRComment) Icon() string {
	return "github"
}

func (p *OnPRComment) Color() string {
	return "gray"
}

func (p *OnPRComment) Configuration() []configuration.Field {
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

func (p *OnPRComment) Setup(ctx core.TriggerContext) error {
	err := ensureRepoInMetadata(
		ctx.Metadata,
		ctx.AppInstallation,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnPRCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Request a single webhook that listens to all PR comment event types:
	// - pull_request_review_comment: line-level code review comments
	// - issue_comment: PR conversation comments (GitHub sends these for comments on PR's main tab)
	// - pull_request_review: review submission comments (the main comment when clicking "Submit review")
	return ctx.AppInstallation.RequestWebhook(WebhookConfiguration{
		EventTypes: []string{"pull_request_review_comment", "issue_comment", "pull_request_review"},
		Repository: config.Repository,
	})
}

func (p *OnPRComment) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnPRComment) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPRComment) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnPRCommentConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing X-GitHub-Event header")
	}

	// Accept all PR comment event types:
	// - pull_request_review_comment: line-level code comments
	// - issue_comment: PR conversation comments
	// - pull_request_review: review submission comments
	if eventType != "pull_request_review_comment" && eventType != "issue_comment" && eventType != "pull_request_review" {
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

	// For issue_comment events, only process if it's on a PR (not a regular issue)
	// GitHub includes a pull_request field in the issue object for PR comments
	if eventType == "issue_comment" {
		issue, ok := data["issue"].(map[string]any)
		if !ok {
			return http.StatusOK, nil
		}
		if _, hasPR := issue["pull_request"]; !hasPR {
			return http.StatusOK, nil
		}
	}

	// Check action based on event type:
	// - pull_request_review uses "submitted" action
	// - other events use "created" action
	action, ok := data["action"]
	if !ok {
		return http.StatusOK, nil
	}

	if eventType == "pull_request_review" {
		if action != "submitted" {
			return http.StatusOK, nil
		}
	} else {
		if action != "created" {
			return http.StatusOK, nil
		}
	}

	// Apply content filter if configured
	if config.ContentFilter != "" {
		var body string

		if eventType == "pull_request_review" {
			// For review submissions, the body is in review.body
			review, ok := data["review"].(map[string]any)
			if !ok {
				return http.StatusBadRequest, fmt.Errorf("invalid review structure")
			}
			// Review body can be empty (e.g., approving without a comment)
			body, _ = review["body"].(string)
		} else {
			// For other events, the body is in comment.body
			comment, ok := data["comment"].(map[string]any)
			if !ok {
				return http.StatusBadRequest, fmt.Errorf("invalid comment structure")
			}
			body, ok = comment["body"].(string)
			if !ok {
				return http.StatusBadRequest, fmt.Errorf("invalid comment body")
			}
		}

		matched, err := regexp.MatchString(config.ContentFilter, body)
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("invalid regex pattern: %w", err)
		}

		if !matched {
			return http.StatusOK, nil
		}
	}

	err = ctx.Events.Emit("github.prComment", data)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
