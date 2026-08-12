package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	commentEventCreated = "comment_created"
	commentEventUpdated = "comment_updated"
	commentEventDeleted = "comment_deleted"

	// IssueCommentEventPayloadType is the event type emitted for every matching comment webhook.
	IssueCommentEventPayloadType = "jira.issueComment"
)

type OnIssueComment struct{}

type OnIssueCommentConfiguration struct {
	Project       string   `json:"project" mapstructure:"project"`
	Events        []string `json:"events" mapstructure:"events"`
	ContentFilter string   `json:"contentFilter" mapstructure:"contentFilter"`
}

type OnIssueCommentMetadata struct {
	Project *Project `json:"project,omitempty" mapstructure:"project,omitempty"`
}

// Comment is a Jira issue comment, as sent in comment webhook payloads.
type Comment struct {
	ID           string `json:"id"`
	Body         any    `json:"body"`
	Author       *User  `json:"author,omitempty"`
	UpdateAuthor *User  `json:"updateAuthor,omitempty"`
}

// CommentWebhookPayload is the shape Jira Cloud sends for comment event webhooks.
type CommentWebhookPayload struct {
	WebhookEvent string   `json:"webhookEvent"`
	Comment      *Comment `json:"comment"`
	Issue        *Issue   `json:"issue"`
}

// IssueCommentEvent is the event SuperPlane emits for each matching comment webhook.
type IssueCommentEvent struct {
	Action  string   `json:"action"`
	Comment *Comment `json:"comment"`
	Issue   *Issue   `json:"issue"`
}

func (t *OnIssueComment) Name() string {
	return "jira.onIssueComment"
}

func (t *OnIssueComment) Label() string {
	return "On Issue Comment"
}

func (t *OnIssueComment) Description() string {
	return "Listen to comment created, updated, or deleted events on Jira issues"
}

func (t *OnIssueComment) Documentation() string {
	return `The On Issue Comment trigger starts a workflow execution when a comment is added, edited, or removed on an issue in a Jira project.

## Use Cases

- **Customer response workflows**: React when a customer comments on a service desk request
- **Notification workflows**: Send notifications when someone comments on an issue
- **Sync workflows**: Mirror Jira comments into other tools

## Configuration

- **Project**: The Jira project to listen for comment events in
- **Events**: Which comment events to listen for (Created, Updated, Deleted)
- **Content Filter** (optional): Regex pattern to filter comments by content, e.g. ` + "`urgent`" + ` to
  only trigger on comments containing "urgent". A comment with no extractable text never matches
  when a filter is set.

## Webhook Setup

This is provisioned automatically, sharing the same webhook registration ` + "`jira.onIssue`" + ` uses - see that trigger's documentation for how Jira's dynamic webhook registration works.

## Output

Emits one event per matching comment webhook with:
- **action**: ` + "`created`" + `, ` + "`updated`" + `, or ` + "`deleted`" + `
- **comment**: The comment (id, body, author, updateAuthor)
- **issue**: The full issue the comment belongs to (id, key, self, fields)`
}

func (t *OnIssueComment) Icon() string {
	return "jira"
}

func (t *OnIssueComment) Color() string {
	return "blue"
}

func (t *OnIssueComment) ExampleData() map[string]any {
	return onIssueCommentExampleData()
}

func (t *OnIssueComment) Configuration() []configuration.Field {
	fields := projectAndEventsFields("The Jira project to listen for comment events in", "Which comment events to listen for")
	return append(fields, configuration.Field{
		Name:        "contentFilter",
		Label:       "Content Filter",
		Type:        configuration.FieldTypeString,
		Required:    false,
		Placeholder: "e.g., urgent",
		Description: "Optional regex pattern to filter comments by content",
	})
}

func (t *OnIssueComment) Setup(ctx core.TriggerContext) error {
	config := OnIssueCommentConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	projectKey := strings.TrimSpace(config.Project)
	if projectKey == "" {
		return fmt.Errorf("project is required")
	}
	if len(config.Events) == 0 {
		return fmt.Errorf("at least one event must be selected")
	}
	if config.ContentFilter != "" {
		if _, err := regexp.Compile(config.ContentFilter); err != nil {
			return fmt.Errorf("invalid content filter pattern: %w", err)
		}
	}

	project, err := requireProject(ctx.HTTP, ctx.Integration, projectKey)
	if err != nil {
		return err
	}

	if err := ctx.Metadata.Set(OnIssueCommentMetadata{Project: project}); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Events: []string{commentEventCreated, commentEventUpdated, commentEventDeleted},
	})
}

func (t *OnIssueComment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnIssueComment) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIssueComment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	config := OnIssueCommentConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	metadata := OnIssueCommentMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	payload := CommentWebhookPayload{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %w", err)
	}

	action, ok := commentEventAction(payload.WebhookEvent)
	if !ok {
		ctx.Logger.Infof("Ignoring event - unsupported webhookEvent %q", payload.WebhookEvent)
		return http.StatusOK, nil, nil
	}

	if !slices.Contains(config.Events, action) {
		ctx.Logger.Infof("Ignoring event - action %q is not configured", action)
		return http.StatusOK, nil, nil
	}

	if payload.Comment == nil || payload.Issue == nil {
		ctx.Logger.Info("Ignoring event - missing comment or issue")
		return http.StatusOK, nil, nil
	}

	if metadata.Project != nil && !strings.EqualFold(issueProjectKey(payload.Issue), metadata.Project.Key) {
		ctx.Logger.Infof("Ignoring event - project does not match %q", metadata.Project.Key)
		return http.StatusOK, nil, nil
	}

	matched, err := matchesContentFilter(config.ContentFilter, payload.Comment.Body)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}
	if !matched {
		ctx.Logger.Info("Ignoring event - comment does not match the content filter")
		return http.StatusOK, nil, nil
	}

	event := IssueCommentEvent{
		Action:  action,
		Comment: payload.Comment,
		Issue:   payload.Issue,
	}

	if err := ctx.Events.Emit(IssueCommentEventPayloadType, event); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %w", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnIssueComment) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// commentEventAction maps a native Jira webhookEvent value to this trigger's short action name.
func commentEventAction(webhookEvent string) (string, bool) {
	switch webhookEvent {
	case commentEventCreated:
		return "created", true
	case commentEventUpdated:
		return "updated", true
	case commentEventDeleted:
		return "deleted", true
	default:
		return "", false
	}
}

// commentPlainText walks an Atlassian Document Format tree (Jira's alternative to a plain string
// comment body) for its text nodes, so content filtering works the same way regardless of shape.
func commentPlainText(body any) string {
	if text, ok := body.(string); ok {
		return text
	}

	node, ok := body.(map[string]any)
	if !ok {
		return ""
	}
	if text, ok := node["text"].(string); ok {
		return text
	}
	if attrs, ok := node["attrs"].(map[string]any); ok {
		if text, ok := attrs["text"].(string); ok {
			return text
		}
	}

	content, ok := node["content"].([]any)
	if !ok {
		return ""
	}

	parts := make([]string, 0, len(content))
	for _, child := range content {
		if part := commentPlainText(child); part != "" {
			parts = append(parts, part)
		}
	}

	if hasBlockChild(content) {
		return strings.Join(parts, " ")
	}
	return strings.Join(parts, "")
}

func hasBlockChild(content []any) bool {
	for _, child := range content {
		if node, ok := child.(map[string]any); ok {
			if _, ok := node["content"].([]any); ok {
				return true
			}
		}
	}
	return false
}

func matchesContentFilter(filter string, body any) (bool, error) {
	if filter == "" {
		return true, nil
	}

	text := commentPlainText(body)
	if text == "" {
		return false, nil
	}

	matched, err := regexp.MatchString(filter, text)
	if err != nil {
		return false, fmt.Errorf("invalid content filter pattern: %w", err)
	}

	return matched, nil
}
