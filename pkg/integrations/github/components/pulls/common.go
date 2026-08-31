package pulls

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const (
	prCommentScopeAll     = "all"
	prCommentScopeReplies = "replies"
)

type prCommentTriggerConfiguration struct {
	Repository               string   `json:"repository" mapstructure:"repository"`
	ContentFilter            string   `json:"contentFilter" mapstructure:"contentFilter"`
	IgnoreBots               bool     `json:"ignoreBots" mapstructure:"ignoreBots"`
	AllowedBots              []string `json:"allowedBots" mapstructure:"allowedBots"`
	IncludeReviewSubmissions *bool    `json:"includeReviewSubmissions" mapstructure:"includeReviewSubmissions"`
	CommentScope             string   `json:"commentScope" mapstructure:"commentScope"`
}

func (c prCommentTriggerConfiguration) includeReviewSubmissions() bool {
	if c.IncludeReviewSubmissions == nil {
		return true
	}

	return *c.IncludeReviewSubmissions
}

func (c prCommentTriggerConfiguration) commentScope() string {
	if c.CommentScope == "" {
		return prCommentScopeAll
	}

	return c.CommentScope
}

func prCommentConfigurationFields() []configuration.Field {
	return append(prCommentBaseConfigurationFields(), ignoreBotsConfigurationField(), allowedBotsConfigurationField())
}

func prReviewCommentConfigurationFields() []configuration.Field {
	return append(prCommentBaseConfigurationFields(),
		configuration.Field{
			Name:        "includeReviewSubmissions",
			Label:       "Include Review Submissions",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Also start a run when a pull request review is submitted. Turn this off when a dedicated review trigger handles submissions.",
		},
		configuration.Field{
			Name:        "commentScope",
			Label:       "Comment Scope",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     prCommentScopeAll,
			Description: "All comments include top-level review comments. Replies include only comments that reply to an existing thread.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "All comments", Value: prCommentScopeAll},
						{Label: "Replies only", Value: prCommentScopeReplies},
					},
				},
			},
		},
		ignoreBotsConfigurationField(),
		allowedBotsConfigurationField(),
	)
}

func prCommentBaseConfigurationFields() []configuration.Field {
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
			Placeholder: "e.g., /solve or @superplaneagent",
			Description: "Optional filter on comment content. Mentions that start with @ match as an exact GitHub username. Other values are regular expressions.",
		},
	}
}

func ignoreBotsConfigurationField() configuration.Field {
	return configuration.Field{
		Name:        "ignoreBots",
		Label:       "Ignore Bots",
		Type:        configuration.FieldTypeBool,
		Required:    false,
		Default:     false,
		Description: "Skip comments and reviews written by GitHub Apps and bots.",
	}
}

func allowedBotsConfigurationField() configuration.Field {
	return configuration.Field{
		Name:        "allowedBots",
		Label:       "Allowed Bots",
		Type:        configuration.FieldTypeList,
		Required:    false,
		Description: "React to comments from these GitHub Apps or bots even when the comment does not match the content filter. Enter the bot login, for example coderabbitai.",
		TypeOptions: &configuration.TypeOptions{
			List: &configuration.ListTypeOptions{
				ItemLabel: "Bot username",
				ItemDefinition: &configuration.ListItemDefinition{
					Type: configuration.FieldTypeString,
				},
			},
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

	return applyContentFilter(filter, body)
}

func applyContentFilter(filter string, bodies ...string) (bool, int, error) {
	if filter == "" {
		return true, http.StatusOK, nil
	}

	for _, body := range bodies {
		matched, err := contentFilterMatches(filter, body)
		if err != nil {
			return false, http.StatusBadRequest, err
		}
		if matched {
			return true, http.StatusOK, nil
		}
	}

	return false, http.StatusOK, nil
}

func contentFilterMatches(filter, body string) (bool, error) {
	filter = strings.TrimSpace(filter)
	if strings.HasPrefix(filter, "@") {
		return mentionMatches(filter, body), nil
	}

	matched, err := regexp.MatchString(filter, body)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return matched, nil
}

func mentionMatches(mention, body string) bool {
	mention = strings.ToLower(strings.TrimPrefix(mention, "@"))
	if mention == "" {
		return false
	}

	lower := strings.ToLower(body)
	needle := "@" + mention
	start := 0
	for {
		index := strings.Index(lower[start:], needle)
		if index < 0 {
			return false
		}

		index += start
		after := index + len(needle)
		if after == len(lower) || !isGitHubUsernameChar(lower[after]) {
			return true
		}

		start = after
	}
}

func isGitHubUsernameChar(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-'
}

func isBotAuthor(eventType string, data map[string]any) bool {
	user := authorUser(eventType, data)
	if user == nil {
		return false
	}

	userType, _ := user["type"].(string)
	return strings.EqualFold(userType, "Bot")
}

func authorLogin(eventType string, data map[string]any) string {
	user := authorUser(eventType, data)
	if user == nil {
		return ""
	}

	login, _ := user["login"].(string)
	return login
}

func isAllowedBot(eventType string, data map[string]any, allowed []string) bool {
	if len(allowed) == 0 {
		return false
	}

	if !isBotAuthor(eventType, data) {
		return false
	}

	login := normalizeBotName(authorLogin(eventType, data))
	if login == "" {
		return false
	}

	return slices.ContainsFunc(allowed, func(candidate string) bool {
		return normalizeBotName(candidate) == login
	})
}

func normalizeBotName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.TrimPrefix(name, "@")
	name = strings.TrimSuffix(name, "[bot]")
	return name
}

func authorUser(eventType string, data map[string]any) map[string]any {
	if eventType == "pull_request_review" {
		if review, ok := data["review"].(map[string]any); ok {
			if user, ok := review["user"].(map[string]any); ok {
				return user
			}
		}
	}

	if comment, ok := data["comment"].(map[string]any); ok {
		if user, ok := comment["user"].(map[string]any); ok {
			return user
		}
	}

	if sender, ok := data["sender"].(map[string]any); ok {
		return sender
	}

	return nil
}

func webhookRepositoryFullName(data map[string]any, fallback string) string {
	repository, ok := data["repository"].(map[string]any)
	if !ok {
		return fallback
	}

	fullName, _ := repository["full_name"].(string)
	if fullName != "" {
		return fullName
	}

	return fallback
}

func int64FromJSON(value any) (int64, bool) {
	switch typed := value.(type) {
	case float64:
		return int64(typed), true
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case json.Number:
		parsed, err := typed.Int64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func isReviewReply(data map[string]any) bool {
	comment, ok := data["comment"].(map[string]any)
	if !ok {
		return false
	}

	replyTo, exists := comment["in_reply_to_id"]
	if !exists || replyTo == nil {
		return false
	}

	switch value := replyTo.(type) {
	case float64:
		return value != 0
	case int:
		return value != 0
	case int64:
		return value != 0
	case json.Number:
		parsed, err := value.Int64()
		return err == nil && parsed != 0
	case string:
		return value != "" && value != "0"
	default:
		return true
	}
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
