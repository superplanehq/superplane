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
	Project     string   `json:"project" mapstructure:"project"`
	Actions     []string `json:"actions" mapstructure:"actions"`
	Labels      []string `json:"labels" mapstructure:"labels"`
	AssigneeIDs []string `json:"assignee_ids" mapstructure:"assigneeIds"`
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

- **Notify Slack** when an issue is created or assigned for triage
- **Create a Jira issue** when a GitLab issue is created for traceability
- **Update external dashboards** or close linked tickets when an issue is closed

## Configuration

- **Project** (required): GitLab project to monitor
- **Actions** (required): Select which issue actions to listen for (opened, closed, reopened, etc.). Default: opened.
- **Labels** (optional): Only trigger for issues with specific labels
- **Assignees** (optional): Only trigger when issue is assigned to specific users

## Outputs

- **Default channel**: Emits issue payload including issue IID, title, state, labels, assignees, author, and action type

## Webhook Setup

This trigger automatically sets up a GitLab webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (i *OnIssue) Icon() string {
	return "gitlab"
}

func (i *OnIssue) Color() string {
	return "orange"
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
		{
			Name:     "labels",
			Label:    "Label Filter",
			Type:     configuration.FieldTypeList,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:     "assigneeIds",
			Label:    "Assignee Filter",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "member",
					Multi:          true,
					UseNameAsValue: false, // Use ID
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

	if len(config.Actions) > 0 && !whitelistedAction(data, config.Actions) {
		return http.StatusOK, nil
	}

	if len(config.Labels) > 0 {
		eventLabels, ok := data["labels"].([]any)
		if !ok {
			return http.StatusOK, nil
		}

		found := false
		for _, label := range eventLabels {
			labelMap, ok := label.(map[string]any)
			if ok {
				title, _ := labelMap["title"].(string)
				for _, requiredLabel := range config.Labels {
					if title == requiredLabel {
						found = true
						break
					}
				}
			}
			if found {
				break
			}
		}
		if !found {
			return http.StatusOK, nil
		}
	}

	// Filter by Assignees
	if len(config.AssigneeIDs) > 0 {

		eventAssignees, ok := data["assignees"].([]any)
		if !ok {
			return http.StatusOK, nil
		}

		found := false
		for _, assignee := range eventAssignees {
			assigneeMap, ok := assignee.(map[string]any)
			if ok {
				idFloat, _ := assigneeMap["id"].(float64)
				idStr := fmt.Sprintf("%.0f", idFloat)

				for _, requiredID := range config.AssigneeIDs {
					if idStr == requiredID {
						found = true
						break
					}
				}
			}
			if found {
				break
			}
		}
		if !found {
			return http.StatusOK, nil
		}
	}

	if err := ctx.Events.Emit("gitlab.issue", data); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (i *OnIssue) Cleanup(ctx core.TriggerContext) error {
	return nil
}
