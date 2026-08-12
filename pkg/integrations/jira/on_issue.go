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
	Project *Project `json:"project,omitempty" mapstructure:"project,omitempty"`
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

This is provisioned automatically. Jira's dynamic webhook registration API (` + "`POST /rest/api/3/webhook`" + `) only allows a single registered callback URL per OAuth connection, so SuperPlane registers one shared webhook per Jira integration - every ` + "`jira.onIssue`" + ` trigger on that connection listens through it, each matching only the events for its own configured project. The shared webhook is removed automatically once the last trigger using it is deleted.

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
	return projectAndEventsFields("The Jira project to listen for issue events in", "Which issue events to listen for")
}

func projectAndEventsFields(projectDescription, eventsDescription string) []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: projectDescription,
			Placeholder: "Select a project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		issueEventsField(eventsDescription),
	}
}

// issueEventsField is the "Events" multi-select shared by triggers scoped to Jira's standard
// issue lifecycle events (created, updated, deleted) - currently jira.onIssue and jira.onIncident,
// since an incident is just an issue on an incident-practice request type.
func issueEventsField(description string) configuration.Field {
	return configuration.Field{
		Name:        "events",
		Label:       "Events",
		Type:        configuration.FieldTypeMultiSelect,
		Required:    true,
		Default:     []string{"created"},
		Description: description,
		TypeOptions: &configuration.TypeOptions{
			MultiSelect: &configuration.MultiSelectTypeOptions{
				Options: []configuration.FieldOption{
					{Label: "Created", Value: "created"},
					{Label: "Updated", Value: "updated"},
					{Label: "Deleted", Value: "deleted"},
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

	if err := ctx.Metadata.Set(OnIssueMetadata{Project: project}); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	// Jira's dynamic webhook API allows only one registered callback URL per OAuth connection.
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Events: []string{issueEventCreated, issueEventUpdated, issueEventDeleted},
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

	// The webhook is shared by every jira.onIssue trigger on the integration (see
	// JiraWebhookHandler), so this project check is the only thing keeping one trigger from
	// reacting to another project's events - fail closed when the payload doesn't carry a
	// project key to compare against, rather than letting an unidentifiable event through.
	if metadata.Project != nil && !strings.EqualFold(issueProjectKey(payload.Issue), metadata.Project.Key) {
		ctx.Logger.Infof("Ignoring event - project does not match %q", metadata.Project.Key)
		return http.StatusOK, nil, nil
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

// Cleanup does nothing: the shared Jira webhook registered via RequestWebhook is torn down by
// the platform's webhook cleanup worker (through JiraWebhookHandler.Cleanup) once the last
// jira.onIssue trigger referencing it is removed.
func (t *OnIssue) Cleanup(ctx core.TriggerContext) error {
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
