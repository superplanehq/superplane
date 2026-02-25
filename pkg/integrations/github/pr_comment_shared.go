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

type prCommentTriggerConfiguration struct {
	Repository    string `json:"repository" mapstructure:"repository"`
	ContentFilter string `json:"contentFilter" mapstructure:"contentFilter"`
}

func prCommentConfigurationFields() []configuration.Field {
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
			Name:        "contentFilter",
			Label:       "Content Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., /solve",
			Description: "Optional regex pattern to filter comments by content",
		},
	}
}

func decodePRCommentConfiguration(configuration any) (prCommentTriggerConfiguration, error) {
	config := prCommentTriggerConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return prCommentTriggerConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	return config, nil
}

func setupPRCommentTrigger(ctx core.TriggerContext, webhookConfig WebhookConfiguration) error {
	err := ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
	if err != nil {
		return err
	}

	config, err := decodePRCommentConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	webhookConfig.Repository = config.Repository
	return ctx.Integration.RequestWebhook(webhookConfig)
}

func extractGitHubEventType(headers http.Header) (string, error) {
	eventType := headers.Get("X-GitHub-Event")
	if eventType == "" {
		return "", fmt.Errorf("missing X-GitHub-Event header")
	}

	return eventType, nil
}

func verifyAndParseWebhookData(ctx core.WebhookRequestContext) (map[string]any, int, error) {
	code, err := verifySignature(ctx)
	if err != nil {
		return nil, code, err
	}

	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	return data, http.StatusOK, nil
}

func isPRIssueComment(data map[string]any) bool {
	issue, ok := data["issue"].(map[string]any)
	if !ok {
		return false
	}

	_, hasPR := issue["pull_request"]
	return hasPR
}

func isExpectedPRCommentAction(eventType string, data map[string]any) bool {
	action, ok := data["action"].(string)
	if !ok {
		return false
	}

	if eventType == "pull_request_review" {
		return action == "submitted"
	}

	return action == "created"
}

func applyPRCommentContentFilter(filter, eventType string, data map[string]any) (bool, int, error) {
	if filter == "" {
		return true, http.StatusOK, nil
	}

	body, err := extractPRCommentBody(eventType, data)
	if err != nil {
		return false, http.StatusBadRequest, err
	}

	matched, err := regexp.MatchString(filter, body)
	if err != nil {
		return false, http.StatusBadRequest, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return matched, http.StatusOK, nil
}

func extractPRCommentBody(eventType string, data map[string]any) (string, error) {
	if eventType == "pull_request_review" {
		review, ok := data["review"].(map[string]any)
		if !ok {
			return "", fmt.Errorf("invalid review structure")
		}

		// Review body can be empty (for example, an approval without text).
		body, _ := review["body"].(string)
		return body, nil
	}

	comment, ok := data["comment"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid comment structure")
	}

	body, ok := comment["body"].(string)
	if !ok {
		return "", fmt.Errorf("invalid comment body")
	}

	return body, nil
}
