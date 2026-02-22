package jira

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIssueCreated struct{}

type OnIssueCreatedConfiguration struct {
	Project    string   `json:"project" mapstructure:"project"`
	IssueTypes []string `json:"issueTypes" mapstructure:"issueTypes"`
}

func (t *OnIssueCreated) Name() string {
	return "jira.onIssueCreated"
}

func (t *OnIssueCreated) Label() string {
	return "On Issue Created"
}

func (t *OnIssueCreated) Description() string {
	return "Listen for new issues created in Jira"
}

func (t *OnIssueCreated) Documentation() string {
	return `The On Issue Created trigger starts a workflow execution when a new issue is created in a Jira project.

## Use Cases

- **Issue automation**: Automate responses to new issues
- **Notification workflows**: Send notifications when issues are created
- **Task management**: Sync issues with external task management systems
- **Triage automation**: Automatically categorize or assign new issues

## Configuration

- **Project**: Select the Jira project to monitor
- **Issue Types**: Optionally filter by issue type (Task, Bug, Story, etc.)

## Event Data

Each issue created event includes:
- **webhookEvent**: The event type (jira:issue_created)
- **issue**: Complete issue information including key, summary, fields
- **user**: User who created the issue

## Webhook Setup

This trigger automatically sets up a Jira webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnIssueCreated) Icon() string {
	return "jira"
}

func (t *OnIssueCreated) Color() string {
	return "blue"
}

func (t *OnIssueCreated) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:     "issueTypes",
			Label:    "Issue Types",
			Type:     configuration.FieldTypeMultiSelect,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Task", Value: "Task"},
						{Label: "Bug", Value: "Bug"},
						{Label: "Story", Value: "Story"},
						{Label: "Epic", Value: "Epic"},
						{Label: "Sub-task", Value: "Sub-task"},
					},
				},
			},
		},
	}
}

func (t *OnIssueCreated) Setup(ctx core.TriggerContext) error {
	authType, err := ctx.Integration.GetConfig("authType")
	if err != nil {
		return fmt.Errorf("failed to get authType: %w", err)
	}

	if string(authType) != AuthTypeOAuth {
		return fmt.Errorf("webhook triggers require OAuth authentication; API Token integrations do not support webhooks")
	}

	err = ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnIssueCreatedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType: "jira:issue_created",
		Project:   config.Project,
	})
}

func (t *OnIssueCreated) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIssueCreated) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIssueCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnIssueCreatedConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-Atlassian-Webhook-Identifier")
	webhookEvent := ""

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	if we, ok := data["webhookEvent"].(string); ok {
		webhookEvent = we
	}

	if eventType == "" && webhookEvent == "" {
		return http.StatusBadRequest, fmt.Errorf("missing webhook event identifier")
	}

	if webhookEvent != "jira:issue_created" {
		return http.StatusOK, nil
	}

	code, err := verifyJiraSignature(ctx)
	if err != nil {
		return code, err
	}

	if !whitelistedIssueType(data, config.IssueTypes) {
		return http.StatusOK, nil
	}

	err = ctx.Events.Emit("jira.issueCreated", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (t *OnIssueCreated) Cleanup(ctx core.TriggerContext) error {
	return nil
}
