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

const OnIssuePayloadType = "jira.issue"

const (
	JiraEventIssueCreated = "jira:issue_created"
	JiraEventIssueUpdated = "jira:issue_updated"
	JiraEventIssueDeleted = "jira:issue_deleted"
)

type OnIssue struct{}

type OnIssueConfiguration struct {
	Project string   `json:"project,omitempty" mapstructure:"project"`
	Events  []string `json:"events" mapstructure:"events"`
	JQL     string   `json:"jql,omitempty" mapstructure:"jql"`
}

type OnIssueMetadata struct {
	Project *Project `json:"project,omitempty" mapstructure:"project"`
}

func (t *OnIssue) Name() string {
	return "jira.onIssue"
}

func (t *OnIssue) Label() string {
	return "On Issue"
}

func (t *OnIssue) Description() string {
	return "Listen to issue webhook events from Jira"
}

func (t *OnIssue) Documentation() string {
	return `The On Issue trigger starts a workflow execution when Jira sends an issue webhook to SuperPlane.

## Use Cases

- **Triage automation**: react when a new issue is created or assigned
- **Cross-tool sync**: mirror Jira issue state changes into other systems
- **Compliance / escalation**: detect issues matching a JQL filter and route them to follow-up workflows

## Configuration

- **Project**: Optionally limit the trigger to a single Jira project
- **Events**: Issue webhook events to listen for (created, updated, deleted)
- **JQL**: Optional JQL filter applied on the Jira side. Defaults to ` + "`issuekey is not EMPTY`" + ` so all issues match.

## Webhook setup

This trigger registers a webhook with Jira automatically using the OAuth credentials of the integration. No manual configuration in Jira is required.`
}

func (t *OnIssue) Icon() string {
	return "jira"
}

func (t *OnIssue) Color() string {
	return "blue"
}

func (t *OnIssue) ExampleData() map[string]any {
	return getExampleOnIssue()
}

func (t *OnIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optionally limit the trigger to issues in this project",
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
			Description: "Issue events to listen for",
			Default:     []string{JiraEventIssueCreated},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Issue Created", Value: JiraEventIssueCreated},
						{Label: "Issue Updated", Value: JiraEventIssueUpdated},
						{Label: "Issue Deleted", Value: JiraEventIssueDeleted},
					},
				},
			},
		},
		{
			Name:        "jql",
			Label:       "JQL Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional JQL filter applied on the Jira side",
			Placeholder: "project = PROJ AND assignee = currentUser()",
		},
	}
}

func (t *OnIssue) Setup(ctx core.TriggerContext) error {
	config := OnIssueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if len(config.Events) == 0 {
		return fmt.Errorf("at least one event must be selected")
	}

	metadata := OnIssueMetadata{}
	if ctx.Metadata != nil && ctx.Metadata.Get() != nil {
		if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
			return fmt.Errorf("failed to decode trigger metadata: %w", err)
		}
	}

	jqlFilter := strings.TrimSpace(config.JQL)
	if strings.TrimSpace(config.Project) != "" {
		project, err := requireProject(ctx.Integration, config.Project)
		if err != nil {
			return err
		}
		metadata.Project = project
		jqlFilter = scopeJQLToProject(jqlFilter, project.Key)
	} else {
		metadata.Project = nil
	}

	if err := ctx.Integration.RequestWebhook(WebhookConfiguration{
		Events:    config.Events,
		JQLFilter: jqlFilter,
	}); err != nil {
		return fmt.Errorf("failed to request webhook: %w", err)
	}

	if ctx.Metadata == nil {
		return nil
	}
	return ctx.Metadata.Set(metadata)
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

	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	eventName, _ := data["webhookEvent"].(string)
	if eventName == "" {
		return http.StatusOK, nil, nil
	}

	if len(config.Events) > 0 && !slices.Contains(config.Events, eventName) {
		return http.StatusOK, nil, nil
	}

	if strings.TrimSpace(config.Project) != "" {
		projectKey := extractProjectKey(data)
		if projectKey != "" && projectKey != config.Project {
			return http.StatusOK, nil, nil
		}
	}

	if err := ctx.Events.Emit(OnIssuePayloadType, data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnIssue) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func extractProjectKey(data map[string]any) string {
	issue, ok := data["issue"].(map[string]any)
	if !ok {
		return ""
	}

	fields, ok := issue["fields"].(map[string]any)
	if !ok {
		return ""
	}

	project, ok := fields["project"].(map[string]any)
	if !ok {
		return ""
	}

	if key, ok := project["key"].(string); ok {
		return key
	}
	return ""
}

// scopeJQLToProject ensures the user-supplied JQL is constrained to the
// configured project. If they wrote nothing, we emit `project = KEY`. If they
// wrote a clause and it doesn't already reference the project, we wrap their
// filter with an `AND`.
func scopeJQLToProject(filter, projectKey string) string {
	projectClause := fmt.Sprintf("project = %s", projectKey)
	if filter == "" {
		return projectClause
	}
	if strings.Contains(strings.ToLower(filter), "project") {
		return filter
	}
	return fmt.Sprintf("%s AND (%s)", projectClause, filter)
}
