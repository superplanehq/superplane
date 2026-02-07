package sentry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type OnIssueEvent struct{}

type OnIssueEventConfiguration struct {
	Events []string `json:"events"`
}

type WebhookConfiguration struct {
	WebhookSecret string   `json:"webhookSecret"`
	Events        []string `json:"events"`
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
- **Webhook secret**: Paste the Client Secret from your Sentry Internal Integration (used to verify webhook signatures)

## Setup

1. In Sentry: add a webhook URL in your Internal Integration (Developer Settings → your integration → Webhooks). Use the webhook URL shown for this trigger in SuperPlane.
2. Subscribe to **Issue** events (e.g. issue.created, issue.resolved).
3. Copy the integration's **Client Secret** into the Webhook secret field below.`
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
		{
			Name:        "webhookSecret",
			Label:       "Webhook Secret",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Client Secret from your Sentry Internal Integration (used to verify webhook signatures)",
			Placeholder: "Paste from Sentry integration settings",
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
	var webhookSecret string
	if m, ok := ctx.Configuration.(map[string]any); ok {
		webhookSecret, _ = m["webhookSecret"].(string)
	}
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		WebhookSecret: webhookSecret,
		Events:        config.Events,
	})
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

	signature := ctx.Headers.Get("Sentry-Hook-Signature")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing Sentry-Hook-Signature")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil || len(secret) == 0 {
		return http.StatusInternalServerError, fmt.Errorf("webhook secret not configured: %w", err)
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature: %w", err)
	}

	var payload sentryIssueWebhookPayload
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err)
	}

	var config OnIssueEventConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("decode configuration: %w", err)
	}
	if !slices.Contains(config.Events, payload.Action) {
		return http.StatusOK, nil
	}

	eventType := fmt.Sprintf("sentry.issue.%s", payload.Action)
	out := map[string]any{
		"action":       payload.Action,
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
