package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	issueEventCreated = "jira:issue_created"
	issueEventUpdated = "jira:issue_updated"
	issueEventDeleted = "jira:issue_deleted"

	// IssueEventPayloadType is the event type emitted for every matching issue webhook.
	IssueEventPayloadType = "jira.issue"
)

type OnIssue struct{}

type OnIssueConfiguration struct {
	Project string   `json:"project" mapstructure:"project"`
	Events  []string `json:"events" mapstructure:"events"`
}

type OnIssueMetadata struct {
	Project    *Project `json:"project,omitempty" mapstructure:"project,omitempty"`
	WebhookURL string   `json:"webhookUrl" mapstructure:"webhookUrl"`
	WebhookID  *int64   `json:"webhookId,omitempty" mapstructure:"webhookId,omitempty"`
}

// IssueWebhookPayload is the shape Jira Cloud sends for issue event webhooks.
type IssueWebhookPayload struct {
	Timestamp    int64           `json:"timestamp,omitempty"`
	WebhookEvent string          `json:"webhookEvent"`
	Issue        *Issue          `json:"issue"`
	User         *User           `json:"user,omitempty"`
	Changelog    *IssueChangelog `json:"changelog,omitempty"`
}

type IssueChangelog struct {
	ID    string               `json:"id,omitempty"`
	Items []IssueChangelogItem `json:"items,omitempty"`
}

type IssueChangelogItem struct {
	Field      string `json:"field"`
	FieldType  string `json:"fieldtype,omitempty"`
	From       string `json:"from,omitempty"`
	FromString string `json:"fromString,omitempty"`
	To         string `json:"to,omitempty"`
	ToString   string `json:"toString,omitempty"`
}

// IssueEvent is the event SuperPlane emits for each matching issue webhook.
type IssueEvent struct {
	Action    string          `json:"action"`
	Issue     *Issue          `json:"issue"`
	User      *User           `json:"user,omitempty"`
	Changelog *IssueChangelog `json:"changelog,omitempty"`
}

func (t *OnIssue) Name() string {
	return "jira.onIssue"
}

func (t *OnIssue) Label() string {
	return "On Issue"
}

func (t *OnIssue) Description() string {
	return "Listen to issue created, updated, or deleted events in Jira"
}

func (t *OnIssue) Documentation() string {
	return `The On Issue trigger starts a workflow execution when issue events occur in a Jira project.

## Use Cases

- **Issue automation**: Automate responses to new or updated Jira issues
- **Notification workflows**: Send notifications when issues are created or updated
- **Sync workflows**: Mirror Jira issues into other tools

## Configuration

- **Project**: The Jira project to listen for issue events in
- **Events**: Which issue events to listen for (Created, Updated, Deleted)

## Webhook Setup

This is provisioned automatically. Jira's dynamic webhook registration API (` + "`POST /rest/api/3/webhook`" + `) is only reachable by Connect/OAuth apps, not accounts authenticated with an API token - since this integration connects via OAuth 2.0 (3LO), SuperPlane registers the webhook directly on save (scoped to the selected project via a JQL filter) and removes it automatically when the trigger is deleted.

## Output

Emits one event per matching issue webhook with:
- **action**: ` + "`created`" + `, ` + "`updated`" + `, or ` + "`deleted`" + `
- **issue**: The full issue (id, key, self, fields)
- **user**: The user who triggered the event
- **changelog**: The list of changed fields (only present for updates)`
}

func (t *OnIssue) Icon() string {
	return "jira"
}

func (t *OnIssue) Color() string {
	return "blue"
}

func (t *OnIssue) ExampleData() map[string]any {
	return onIssueExampleData()
}

func (t *OnIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Jira project to listen for issue events in",
			Placeholder: "Select a project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:        "events",
			Label:       "Events",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    true,
			Default:     []string{"created"},
			Description: "Which issue events to listen for",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "created"},
						{Label: "Updated", Value: "updated"},
						{Label: "Deleted", Value: "deleted"},
					},
				},
			},
		},
	}
}

func (t *OnIssue) Setup(ctx core.TriggerContext) error {
	config := OnIssueConfiguration{}
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

	project, err := requireProject(ctx.HTTP, ctx.Integration, projectKey)
	if err != nil {
		return err
	}

	if ctx.Webhook == nil {
		return fmt.Errorf("missing webhook context")
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup webhook: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Tear down a previously-registered webhook so editing project/events doesn't leak orphaned registrations.
	existing := OnIssueMetadata{}
	_ = mapstructure.Decode(ctx.Metadata.Get(), &existing)
	if existing.WebhookID != nil {
		if delErr := client.DeleteIssueWebhooks([]int64{*existing.WebhookID}); delErr != nil {
			ctx.Logger.Warnf("failed to remove previous Jira webhook: %v", delErr)
		}
	}

	nativeEvents := make([]string, 0, len(config.Events))
	for _, event := range config.Events {
		nativeEvents = append(nativeEvents, issueEventWebhookName(event))
	}

	webhookID, err := client.CreateIssueWebhook(webhookURL, fmt.Sprintf("project = %q", project.Key), nativeEvents)
	if err != nil {
		return fmt.Errorf("failed to create Jira webhook: %w", err)
	}

	return ctx.Metadata.Set(OnIssueMetadata{
		Project:    project,
		WebhookURL: webhookURL,
		WebhookID:  &webhookID,
	})
}

func (t *OnIssue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnIssue) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	config := OnIssueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	metadata := OnIssueMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	payload := IssueWebhookPayload{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %w", err)
	}

	action, ok := issueEventAction(payload.WebhookEvent)
	if !ok {
		ctx.Logger.Infof("Ignoring event - unsupported webhookEvent %q", payload.WebhookEvent)
		return http.StatusOK, nil, nil
	}

	if !slices.Contains(config.Events, action) {
		ctx.Logger.Infof("Ignoring event - action %q is not configured", action)
		return http.StatusOK, nil, nil
	}

	if payload.Issue == nil {
		ctx.Logger.Info("Ignoring event - missing issue")
		return http.StatusOK, nil, nil
	}

	if metadata.Project != nil {
		if projectKey := issueProjectKey(payload.Issue); projectKey != "" && !strings.EqualFold(projectKey, metadata.Project.Key) {
			ctx.Logger.Infof("Ignoring event for project %s", projectKey)
			return http.StatusOK, nil, nil
		}
	}

	event := IssueEvent{
		Action:    action,
		Issue:     payload.Issue,
		User:      payload.User,
		Changelog: payload.Changelog,
	}

	if err := ctx.Events.Emit(IssueEventPayloadType, event); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %w", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnIssue) Cleanup(ctx core.TriggerContext) error {
	metadata := OnIssueMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return nil
	}
	if metadata.WebhookID == nil {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil
	}

	if err := client.DeleteIssueWebhooks([]int64{*metadata.WebhookID}); err != nil {
		ctx.Logger.Warnf("failed to delete Jira webhook during cleanup: %v", err)
	}
	return nil
}

// issueEventAction maps a native Jira webhookEvent value to this trigger's short action name.
func issueEventAction(webhookEvent string) (string, bool) {
	switch webhookEvent {
	case issueEventCreated:
		return "created", true
	case issueEventUpdated:
		return "updated", true
	case issueEventDeleted:
		return "deleted", true
	default:
		return "", false
	}
}

// issueEventWebhookName maps this trigger's short action name to the native Jira webhookEvent value.
func issueEventWebhookName(action string) string {
	switch action {
	case "created":
		return issueEventCreated
	case "updated":
		return issueEventUpdated
	case "deleted":
		return issueEventDeleted
	default:
		return action
	}
}

func issueProjectKey(issue *Issue) string {
	if issue == nil || issue.Fields == nil {
		return ""
	}
	project, ok := issue.Fields["project"].(map[string]any)
	if !ok {
		return ""
	}
	key, _ := project["key"].(string)
	return key
}
