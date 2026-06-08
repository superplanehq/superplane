package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	IssueActionCreated = "created"
	IssueActionUpdated = "updated"
	IssueActionDeleted = "deleted"

	jiraWebhookEventCreated = "jira:issue_created"
	jiraWebhookEventUpdated = "jira:issue_updated"
	jiraWebhookEventDeleted = "jira:issue_deleted"
)

type OnIssue struct{}

type OnIssueConfiguration struct {
	Project string   `json:"project" mapstructure:"project"`
	Actions []string `json:"actions" mapstructure:"actions"`
}

func (t *OnIssue) Name() string {
	return "jira.onIssue"
}

func (t *OnIssue) Label() string {
	return "On Issue"
}

func (t *OnIssue) Description() string {
	return "Start a workflow when Jira creates, updates, or deletes an issue"
}

func (t *OnIssue) Documentation() string {
	return `The On Issue trigger starts a workflow execution when Jira sends an issue webhook.

## Use Cases

- **Issue automation**: Run workflows when a Jira issue is created, updated, or deleted
- **Project routing**: Filter issue events to a specific Jira project
- **Notification workflows**: Send updates to other systems when issue activity happens

## Configuration

- **Project**: Optionally filter events to one Jira project. Leave empty to receive issues from all projects.
- **Actions**: Optionally filter to created, updated, or deleted issue events. Leave empty to receive all issue events.

## Webhook Setup

The webhook is created automatically in Jira through the REST API when you save the canvas. SuperPlane keeps one Jira webhook per connected Jira site and routes matching issue events to the configured triggers.`
}

func (t *OnIssue) Icon() string {
	return "jira"
}

func (t *OnIssue) Color() string {
	return "blue"
}

func (t *OnIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Filter by project. Leave empty to receive issues from all projects.",
			Placeholder: "Select a project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:        "actions",
			Label:       "Actions",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter by issue action. Leave empty to receive all issue events.",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: IssueActionCreated},
						{Label: "Updated", Value: IssueActionUpdated},
						{Label: "Deleted", Value: IssueActionDeleted},
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

	for _, action := range config.Actions {
		if !isKnownIssueAction(action) {
			return fmt.Errorf("unsupported issue action: %s", action)
		}
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode Jira metadata: %w", err)
	}

	if metadata.CloudID == "" {
		return fmt.Errorf("Jira integration is not connected yet — complete the OAuth flow before saving this trigger")
	}

	var project *Project
	if strings.TrimSpace(config.Project) != "" {
		found := slices.IndexFunc(metadata.Projects, func(p Project) bool {
			return p.Key == config.Project || p.ID == config.Project
		})
		if found == -1 {
			return fmt.Errorf("project %s is not accessible to integration", config.Project)
		}

		project = &metadata.Projects[found]
	}

	if err := ctx.Metadata.Set(NodeMetadata{Project: project}); err != nil {
		return fmt.Errorf("failed to set node metadata: %w", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{CloudID: metadata.CloudID})
}

func (t *OnIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	if status, err := verifyJiraWebhookAuthorization(ctx); err != nil {
		return status, nil, err
	}

	config := OnIssueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	payload := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("failed to parse Jira webhook payload: %w", err)
	}

	webhookEvent, _ := payload["webhookEvent"].(string)
	action := issueActionFromWebhookEvent(webhookEvent)
	if action == "" {
		return http.StatusOK, nil, nil
	}

	if len(config.Actions) > 0 && !slices.Contains(config.Actions, action) {
		return http.StatusOK, nil, nil
	}

	if config.Project != "" && !payloadMatchesProject(payload, config.Project) {
		return http.StatusOK, nil, nil
	}

	payload["action"] = action
	if err := ctx.Events.Emit("jira.issue."+action, payload); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to emit Jira issue event: %w", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnIssue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnIssue) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIssue) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func verifyJiraWebhookAuthorization(ctx core.WebhookRequestContext) (int, error) {
	config := loadConfiguration(ctx.Integration)
	if config.ClientSecret == "" {
		return http.StatusForbidden, fmt.Errorf("client secret is required for Jira webhook verification")
	}

	header := ctx.Headers.Get("Authorization")
	if header == "" {
		return http.StatusForbidden, fmt.Errorf("missing Authorization header")
	}

	tokenString := stripJiraAuthorizationPrefix(header)
	if tokenString == "" {
		return http.StatusForbidden, fmt.Errorf("invalid Authorization header")
	}

	token, err := jwt.Parse(strings.TrimSpace(tokenString), func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(config.ClientSecret), nil
	})
	if err != nil || !token.Valid {
		return http.StatusForbidden, fmt.Errorf("invalid Jira webhook authorization")
	}

	return http.StatusOK, nil
}

func issueActionFromWebhookEvent(event string) string {
	switch event {
	case jiraWebhookEventCreated:
		return IssueActionCreated
	case jiraWebhookEventUpdated:
		return IssueActionUpdated
	case jiraWebhookEventDeleted:
		return IssueActionDeleted
	default:
		return ""
	}
}

// stripJiraAuthorizationPrefix accepts either the Atlassian Connect-style
// "JWT <token>" header or the more common "Bearer <token>" header.
// Comparison is case-insensitive on the scheme.
func stripJiraAuthorizationPrefix(header string) string {
	header = strings.TrimSpace(header)
	for _, prefix := range []string{"JWT ", "Bearer "} {
		if len(header) >= len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
			return strings.TrimSpace(header[len(prefix):])
		}
	}

	return ""
}

func isKnownIssueAction(action string) bool {
	return action == IssueActionCreated || action == IssueActionUpdated || action == IssueActionDeleted
}

func payloadMatchesProject(payload map[string]any, project string) bool {
	issue, ok := payload["issue"].(map[string]any)
	if !ok {
		return false
	}

	fields, ok := issue["fields"].(map[string]any)
	if !ok {
		return false
	}

	projectData, ok := fields["project"].(map[string]any)
	if !ok {
		return false
	}

	key, _ := projectData["key"].(string)
	id, _ := projectData["id"].(string)
	return project == key || project == id
}
