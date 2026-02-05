package gitlab

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIssue struct{}

type OnIssueConfiguration struct {
	Project string   `json:"project" mapstructure:"project"`
	Actions []string `json:"actions" mapstructure:"actions"`
}

func (i *OnIssue) Name() string {
	return "gitlab.onIssue"
}

func (i *OnIssue) Label() string {
	return "On Issue"
}

func (i *OnIssue) Description() string {
	return "Listen to issue events from GitLab"
}

func (i *OnIssue) Documentation() string {
	return `The On Issue trigger starts a workflow execution when issue events occur in a GitLab project.

## Use Cases

- **Issue automation**: Automate responses to new or updated issues
- **Notification workflows**: Send notifications when issues are created or closed
- **Task management**: Sync issues with external task management systems
- **Label automation**: Automatically label or categorize issues

## Configuration

- **Project**: Select the GitLab project to monitor
- **Actions**: Select which issue actions to listen for (open, close, reopen, update)

## Event Data

Each issue event includes:
- **object_kind**: The type of event (issue)
- **event_type**: The specific event type
- **object_attributes**: Complete issue information including title, description, state, labels
- **project**: Project information
- **user**: User who triggered the event

## Webhook Setup

This trigger automatically sets up a GitLab webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (i *OnIssue) Icon() string {
	return "gitlab"
}

func (i *OnIssue) Color() string {
	return "orange"
}

func (i *OnIssue) ExampleData() map[string]any {
	return map[string]any{
		"object_kind": "issue",
		"event_type":  "issue",
		"user": map[string]any{
			"id":       1,
			"name":     "John Doe",
			"username": "johndoe",
		},
		"project": map[string]any{
			"id":                  15,
			"name":                "my-project",
			"path_with_namespace": "group/my-project",
			"web_url":             "https://gitlab.com/group/my-project",
		},
		"object_attributes": map[string]any{
			"id":          301,
			"iid":         1,
			"title":       "Example Issue",
			"description": "This is an example issue description",
			"state":       "opened",
			"action":      "open",
			"url":         "https://gitlab.com/group/my-project/-/issues/1",
		},
	}
}

func (i *OnIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: false, // Use ID for GitLab
				},
			},
		},
		{
			Name:     "actions",
			Label:    "Actions",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"open"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Opened", Value: "open"},
						{Label: "Closed", Value: "close"},
						{Label: "Reopened", Value: "reopen"},
						{Label: "Updated", Value: "update"},
					},
				},
			},
		},
	}
}

func (i *OnIssue) Setup(ctx core.TriggerContext) error {
	var config OnIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := ensureRepoInMetadata(ctx.Metadata, ctx.Integration, config.Project); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType: "issues",
		ProjectID: config.Project,
	})
}

func (i *OnIssue) Actions() []core.Action {
	return []core.Action{}
}

func (i *OnIssue) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (i *OnIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	var config OnIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-Gitlab-Event")
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing X-Gitlab-Event header")
	}

	// GitLab sends "Issue Hook" for issue events
	if eventType != "Issue Hook" {
		return http.StatusOK, nil
	}

	code, err := verifyWebhookToken(ctx)
	if err != nil {
		return code, err
	}

	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	if !whitelistedAction(data, config.Actions) {
		return http.StatusOK, nil
	}

	if err := ctx.Events.Emit("gitlab.issue", data); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (i *OnIssue) Cleanup(ctx core.TriggerContext) error {
	return nil
}
