package firehydrant

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIncident struct{}

type OnIncidentConfiguration struct {
	Events []string `json:"events"`
}

func (t *OnIncident) Name() string {
	return "firehydrant.onIncident"
}

func (t *OnIncident) Label() string {
	return "On Incident"
}

func (t *OnIncident) Description() string {
	return "Listen to incident events"
}

func (t *OnIncident) Documentation() string {
	return `The On Incident trigger starts a workflow execution when FireHydrant incident events occur.

## Use Cases

- **Incident automation**: Automate responses to incident events
- **Notification workflows**: Send notifications when incidents are created or resolved
- **Integration workflows**: Sync incidents with external systems
- **Post-incident actions**: Trigger follow-up workflows when incidents are resolved or archived

## Configuration

- **Events**: Select which incident events to listen for (created, updated, resolved, archived, deleted)

## Event Data

Each incident event includes:
- **event**: Event type (incident.created, incident.updated, etc.)
- **incident**: Complete incident information including name, description, severity, status

## Webhook Setup

This trigger automatically sets up a FireHydrant webhook endpoint when configured. The endpoint is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnIncident) Icon() string {
	return "flame"
}

func (t *OnIncident) Color() string {
	return "orange"
}

func (t *OnIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "events",
			Label:    "Events",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"incident.created"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "incident.created"},
						{Label: "Updated", Value: "incident.updated"},
						{Label: "Resolved", Value: "incident.resolved"},
						{Label: "Archived", Value: "incident.archived"},
						{Label: "Deleted", Value: "incident.deleted"},
					},
				},
			},
		},
	}
}

func (t *OnIncident) Setup(ctx core.TriggerContext) error {
	config := OnIncidentConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if len(config.Events) == 0 {
		return fmt.Errorf("at least one event type must be chosen")
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Events: config.Events,
	})
}

func (t *OnIncident) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIncident) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnIncidentConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Verify signature
	signature := ctx.Headers.Get("X-FireHydrant-Signature")
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting secret: %v", err)
	}

	if err := verifyWebhookSignature(signature, ctx.Body, secret); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature: %v", err)
	}

	// Parse webhook payload
	var webhook WebhookPayload
	err = json.Unmarshal(ctx.Body, &webhook)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	eventType := webhook.Event.Type

	// Since the webhook may be shared and receive more events than this trigger cares about,
	// we need to filter events by their type here.
	if !slices.Contains(config.Events, eventType) {
		return http.StatusOK, nil
	}

	err = ctx.Events.Emit(
		fmt.Sprintf("firehydrant.%s", eventType),
		buildIncidentPayload(webhook),
	)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

// WebhookPayload represents the FireHydrant webhook payload
type WebhookPayload struct {
	Event WebhookEvent   `json:"event"`
	Data  map[string]any `json:"data"`
}

// WebhookEvent represents the event metadata in a FireHydrant webhook
type WebhookEvent struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
}

func buildIncidentPayload(webhook WebhookPayload) map[string]any {
	payload := map[string]any{
		"event":      webhook.Event.Type,
		"event_id":   webhook.Event.ID,
		"created_at": webhook.Event.CreatedAt,
	}

	if webhook.Data != nil {
		payload["incident"] = webhook.Data
	}

	return payload
}

func (t *OnIncident) Cleanup(ctx core.TriggerContext) error {
	return nil
}
