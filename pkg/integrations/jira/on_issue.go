package jira

import (
	"crypto/subtle"
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
}

// IssueWebhookPayload is the shape Jira Cloud sends for issue event webhooks
// (jira:issue_created/updated/deleted) - the same "issue" representation
// returned by the REST API with no expand parameters, plus the event envelope.
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

Jira Cloud's dynamic webhook registration API is only available to Connect/OAuth apps, not to accounts authenticated with an API token, so this webhook cannot be provisioned automatically. After saving this trigger to generate its webhook URL, connect it on the Jira side using one of:

1. **Jira Administration → System → WebHooks** (requires Jira site admin access): create a WebHook with the generated URL, tick the Issue **created**/**updated**/**deleted** events you want, and optionally scope it with a JQL filter such as ` + "`project = YOUR_PROJECT_KEY`" + `.
2. **Project settings → Automation** (no site admin access required): create a rule with an Issue created/updated/deleted trigger and a **Send web request** action pointing at the generated URL, with a JSON body shaped like ` + "`{\"webhookEvent\": \"jira:issue_created\", \"issue\": {...}}`" + `.

Jira's native webhook delivery has no support for custom headers, so requests aren't signed by default. If you can add a custom header (for example from an Automation rule), set the **Webhook Shared Secret** field on the Jira integration and send it as ` + "`Authorization: Bearer <secret>`" + ` to have SuperPlane verify each request.

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

	return ctx.Metadata.Set(OnIssueMetadata{
		Project:    project,
		WebhookURL: webhookURL,
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

	if code, err := verifyOnIssueWebhookAuth(ctx); err != nil {
		return code, nil, err
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
	return nil
}

// issueEventAction maps a Jira webhookEvent value to the short action name used
// in this trigger's configuration and emitted events.
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

// verifyOnIssueWebhookAuth enforces the optional "webhookSharedSecret" integration
// config as a Bearer token. Jira's native Admin Webhooks can't send custom
// headers, so verification is skipped entirely when no secret is configured.
func verifyOnIssueWebhookAuth(ctx core.WebhookRequestContext) (int, error) {
	if ctx.Integration == nil {
		return http.StatusOK, nil
	}

	secret, err := optionalIntegrationConfig(ctx.Integration, "webhookSharedSecret")
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to read webhook shared secret: %w", err)
	}
	if secret == "" {
		return http.StatusOK, nil
	}

	authorization := ctx.Headers.Get("Authorization")
	token, hasBearer := strings.CutPrefix(authorization, "Bearer ")
	if !hasBearer || subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
		return http.StatusForbidden, fmt.Errorf("invalid or missing webhook authorization")
	}

	return http.StatusOK, nil
}

func optionalIntegrationConfig(integration core.IntegrationContext, name string) (string, error) {
	value, err := integration.GetConfig(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return "", nil
		}
		return "", err
	}
	return string(value), nil
}
