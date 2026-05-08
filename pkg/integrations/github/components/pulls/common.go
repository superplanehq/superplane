package pulls

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

// enrichPRTimeout caps the GitHub API lookup so the webhook handler can finish
// well before GitHub's ~10s delivery timeout - blowing past it triggers retries
// and duplicate events.
const enrichPRTimeout = 7 * time.Second

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

func setupPRCommentTrigger(ctx core.TriggerContext, webhookConfig common.WebhookConfiguration) error {
	err := common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
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
	code, err := common.VerifySignature(ctx)
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

// enrichPRPayload best-effort attaches the full PR (head/base SHA, branch refs)
// to data["pull_request"]. The issue_comment webhook only carries URLs.
// On any failure we log and return; the caller still emits the original event.
func enrichPRPayload(ctx core.WebhookRequestContext, data map[string]any, repository string) {
	if ctx.Integration == nil {
		return
	}

	number, ok := extractPRNumber(data)
	if !ok {
		ctx.Logger.Warn("Skipping PR payload enrichment - missing PR number in webhook")
		return
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		ctx.Logger.Warnf("Skipping PR payload enrichment - failed to build GitHub client: %v", err)
		return
	}

	apiCtx, cancel := context.WithTimeout(context.Background(), enrichPRTimeout)
	defer cancel()

	pr, _, err := client.GetPullRequest(apiCtx, repository, number)
	if err != nil {
		ctx.Logger.Warnf("Skipping PR payload enrichment - GetPullRequest failed: %v", err)
		return
	}

	asMap, err := structToMap(pr)
	if err != nil {
		ctx.Logger.Warnf("Skipping PR payload enrichment - failed to encode PR: %v", err)
		return
	}

	data["pull_request"] = asMap
}

func extractPRNumber(data map[string]any) (int, bool) {
	issue, ok := data["issue"].(map[string]any)
	if !ok {
		return 0, false
	}

	switch n := issue["number"].(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	}
	return 0, false
}

// structToMap round-trips through JSON so we honor the json tags on
// *github.PullRequest. mapstructure would key off Go field names instead.
func structToMap(v any) (map[string]any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
