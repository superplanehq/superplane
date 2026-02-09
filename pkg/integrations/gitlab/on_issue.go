package gitlab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIssue struct{}

type OnIssueConfiguration struct {
	Project string                    `json:"project" mapstructure:"project"`
	Actions []string                  `json:"actions" mapstructure:"actions"`
	Labels  []configuration.Predicate `json:"labels" mapstructure:"labels"`
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
					Type: ResourceTypeProject,
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
			Label:    "Labels",
			Type:     configuration.FieldTypeAnyPredicateList,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
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

	if err := ensureProjectInMetadata(ctx.Metadata, ctx.Integration, config.Project); err != nil {
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

	//
	// Verify that the action is in the allowed list
	//
	if len(config.Actions) > 0 && !i.whitelistedAction(data, config.Actions) {
		return http.StatusOK, nil
	}

	//
	// Verify that the labels are in the allowed list
	//
	if len(config.Labels) > 0 && !i.hasWhitelistedLabel(data, config.Labels) {
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

func (i *OnIssue) whitelistedAction(data map[string]any, allowedActions []string) bool {
	attrs, ok := data["object_attributes"].(map[string]any)
	if !ok {
		return false
	}

	action, ok := attrs["action"].(string)
	if !ok {
		return false
	}

	if !slices.Contains(allowedActions, action) {
		return false
	}

	//
	// If not an update action, just return true,
	// since it's in the allowed list.
	//
	if action != "update" {
		return true
	}

	//
	// Otherwise, we are dealing with an update,
	// and for those, we only accept if the issue is opened.
	//
	state, ok := attrs["state"].(string)
	if !ok {
		return false
	}

	return state == "opened"
}

func (i *OnIssue) hasWhitelistedLabel(data map[string]any, allowedLabels []configuration.Predicate) bool {
	labels, ok := data["labels"].([]any)
	if !ok {
		return false
	}

	for _, label := range labels {
		labelMap := label.(map[string]any)
		title, _ := labelMap["title"].(string)
		if configuration.MatchesAnyPredicate(allowedLabels, title) {
			return true
		}
	}

	return false
}
