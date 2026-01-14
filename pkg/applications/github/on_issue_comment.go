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

type OnIssueComment struct{}

type OnIssueCommentConfiguration struct {
	Repository    string `json:"repository" mapstructure:"repository"`
	ContentFilter string `json:"contentFilter" mapstructure:"contentFilter"`
}

func (i *OnIssueComment) Name() string {
	return "github.onIssueComment"
}

func (i *OnIssueComment) Label() string {
	return "On Issue Comment"
}

func (i *OnIssueComment) Description() string {
	return "Listen to issue comment events"
}

func (i *OnIssueComment) Icon() string {
	return "github"
}

func (i *OnIssueComment) Color() string {
	return "gray"
}

func (i *OnIssueComment) Configuration() []configuration.Field {
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

func (i *OnIssueComment) Setup(ctx core.TriggerContext) error {
	err := ensureRepoInMetadata(
		ctx.Metadata,
		ctx.AppInstallation,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnIssueCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.AppInstallation.RequestWebhook(WebhookConfiguration{
		EventType:  "issue_comment",
		Repository: config.Repository,
	})
}

func (i *OnIssueComment) Actions() []core.Action {
	return []core.Action{}
}

func (i *OnIssueComment) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (i *OnIssueComment) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnIssueCommentConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	code, err := verifySignature(ctx, "issue_comment")
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

	err = ctx.Events.Emit("github.issueComment", data)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
