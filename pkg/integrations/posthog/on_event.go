package posthog

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

type OnEvent struct{}

type OnEventConfiguration struct {
	ProjectID          string   `json:"projectId" mapstructure:"projectId"`
	Events             []string `json:"events" mapstructure:"events"`
	FilterTestAccounts bool     `json:"filterTestAccounts" mapstructure:"filterTestAccounts"`
}

func (t *OnEvent) Name() string {
	return "posthog.onEvent"
}

func (t *OnEvent) Label() string {
	return "On Event"
}

func (t *OnEvent) Description() string {
	return "Listen to product analytics events captured by PostHog"
}

func (t *OnEvent) Documentation() string {
	return `The On Event trigger starts a workflow execution when PostHog captures a matching event.

## Use Cases

- **Activation workflows**: Kick off onboarding when a user reaches a milestone event
- **Incident response**: React to error or failure events captured from your product
- **Revenue automation**: Run a workflow when a purchase or upgrade event lands
- **Data pipelines**: Forward selected product events to other systems as they happen

## Configuration

- **Project**: The PostHog project to listen to.
- **Events**: The event names to listen for. Leave empty to receive every event in the project.
- **Ignore test accounts**: Applies the project's "internal and test users" filter, so events from your own team do not start workflows. Enabled by default.

## Webhook Setup

The webhook is created automatically in PostHog when you save the canvas. No manual setup is required.

SuperPlane uses the PostHog API (with your configured personal API key) to create a
**webhook destination** in the selected project, filtered to the events you chose. The
destination is removed again when the trigger is deleted.

PostHog does not sign webhook payloads. SuperPlane therefore generates a random secret per
webhook, stores it encrypted, and sets it as a request header on the destination. Deliveries
that do not present that secret are rejected.

## Event Data

The payload contains the PostHog event, the person it is attributed to, and the project it
came from:

- ` + "`event`" + ` — the captured event, including ` + "`event`" + ` (its name), ` + "`uuid`" + `, ` + "`distinct_id`" + `, ` + "`timestamp`" + `, ` + "`properties`" + `, and ` + "`url`" + `
- ` + "`person`" + ` — the person the event belongs to, including their ` + "`properties`" + `
- ` + "`project`" + ` — the ` + "`id`" + `, ` + "`name`" + `, and ` + "`url`" + ` of the PostHog project

## Notes

- Webhook destinations run on PostHog's data pipeline, which is a paid feature on PostHog
  Cloud. Check your plan before relying on this trigger.
- Events are filtered inside PostHog, so only matching events are ever sent to SuperPlane.`
}

func (t *OnEvent) Icon() string {
	return "posthog"
}

func (t *OnEvent) Color() string {
	return "gray"
}

func (t *OnEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectId",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The PostHog project to listen to",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:        "events",
			Label:       "Events",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "The events to listen for. Leave empty to receive every event in the project.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "event",
					Multi: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "projectId",
							ValueFrom: &configuration.ParameterValueFrom{Field: "projectId"},
						},
					},
				},
			},
		},
		{
			Name:        "filterTestAccounts",
			Label:       "Ignore test accounts",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Apply the project's internal and test user filter, so events from your own team do not start workflows.",
		},
	}
}

func (t *OnEvent) Setup(ctx core.TriggerContext) error {
	config := OnEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.ProjectID) == "" {
		return fmt.Errorf("project is required")
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		ProjectID:          config.ProjectID,
		Events:             config.Events,
		FilterTestAccounts: config.FilterTestAccounts,
	})
}

func (t *OnEvent) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnEvent) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	config := OnEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil || len(secret) == 0 {
		return http.StatusForbidden, nil, fmt.Errorf("webhook secret is not available; the webhook may still be provisioning")
	}

	//
	// PostHog does not sign its webhook payloads, so authentication is the
	// shared secret SuperPlane put on the destination when it was created.
	//
	token := ctx.Headers.Get(WebhookTokenHeader)
	if subtle.ConstantTimeCompare([]byte(token), secret) != 1 {
		return http.StatusForbidden, nil, fmt.Errorf("invalid %s header", WebhookTokenHeader)
	}

	payload := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %w", err)
	}

	name := eventName(payload)
	if name == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing event name in payload")
	}

	//
	// PostHog already filters by event name, so this only catches a destination
	// that drifted - for example after someone edited it in the PostHog UI.
	//
	if len(config.Events) > 0 && !slices.Contains(config.Events, name) {
		ctx.Logger.Infof("posthog webhook: event %q not in trigger config (configured: %v), acknowledging without emitting", name, config.Events)
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit("posthog.event", payload); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %w", err)
	}

	ctx.Logger.Infof("posthog webhook: emitted %q for workflow %s", name, ctx.WorkflowID)
	return http.StatusOK, nil, nil
}

func (t *OnEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// eventName reads the captured event's name out of the delivered payload,
// which nests it under the event object as `event.event`.
func eventName(payload map[string]any) string {
	event, ok := payload["event"].(map[string]any)
	if !ok {
		return ""
	}

	name, _ := event["event"].(string)
	return name
}
