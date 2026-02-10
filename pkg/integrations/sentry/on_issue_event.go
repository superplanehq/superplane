package sentry

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

type OnIssueEvent struct{}

type OnIssueEventConfiguration struct {
	Events []string `json:"events"`
}

type WebhookConfiguration struct {
	Events []string `json:"events"`
}

type sentryIssueWebhookPayload struct {
	Action       string         `json:"action"`
	Installation map[string]any `json:"installation"`
	Data         struct {
		Issue map[string]any `json:"issue"`
	} `json:"data"`
	Actor map[string]any `json:"actor"`
}

func (t *OnIssueEvent) Name() string {
	return "sentry.onIssueEvent"
}

func (t *OnIssueEvent) Label() string {
	return "On Issue Event"
}

func (t *OnIssueEvent) Description() string {
	return "Start a workflow when Sentry sends issue events (created, resolved, assigned, etc.)"
}

func (t *OnIssueEvent) Documentation() string {
	return `The On Issue Event trigger runs when Sentry sends webhook events for issues.

## Use Cases

- **Notify on new issues**: Send Slack/Discord messages or create Jira tickets when an issue is created
- **Resolve after deploy**: When an issue is resolved in Sentry, run follow-up steps
- **Assign or triage**: React to assigned or archived events

## Configuration

- **Events**: Choose which issue actions to listen for (created, resolved, assigned, archived, unresolved)

## Setup

1. In Sentry: add a webhook URL in your integration/app settings. Use the webhook URL shown for this trigger in SuperPlane.
2. Subscribe to **Issue** events (e.g. issue.created, issue.resolved).
`
}

func (t *OnIssueEvent) Icon() string {
	return "alert-triangle"
}

func (t *OnIssueEvent) Color() string {
	return "gray"
}

func (t *OnIssueEvent) ExampleData() map[string]any {
	return exampleDataOnIssueEvent()
}

func (t *OnIssueEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "events",
			Label:    "Events",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"created"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "created"},
						{Label: "Resolved", Value: "resolved"},
						{Label: "Assigned", Value: "assigned"},
						{Label: "Archived", Value: "archived"},
						{Label: "Unresolved", Value: "unresolved"},
					},
				},
			},
		},
	}
}

func (t *OnIssueEvent) Setup(ctx core.TriggerContext) error {
	var config OnIssueEventConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}
	if len(config.Events) == 0 {
		return fmt.Errorf("at least one event must be selected")
	}
	return ctx.Integration.RequestWebhook(WebhookConfiguration{Events: config.Events})
}

func (t *OnIssueEvent) Actions() []core.Action {
	return nil
}

func (t *OnIssueEvent) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIssueEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	resource := ctx.Headers.Get("Sentry-Hook-Resource")
	if resource != "issue" {
		return http.StatusOK, nil
	}

	var payload sentryIssueWebhookPayload
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err)
	}

	action := strings.TrimPrefix(payload.Action, "issue.")

	var config OnIssueEventConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("decode configuration: %w", err)
	}
	if !slices.Contains(config.Events, action) {
		return http.StatusOK, nil
	}

	eventType := fmt.Sprintf("sentry.issue.%s", action)
	out := map[string]any{
		"action":       action,
		"installation": payload.Installation,
		"issue":        payload.Data.Issue,
		"actor":        payload.Actor,
	}
	if err := ctx.Events.Emit(eventType, out); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("emit event: %w", err)
	}
	return http.StatusOK, nil
}

func (t *OnIssueEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}
