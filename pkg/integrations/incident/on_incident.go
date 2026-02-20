package incident

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// Event type names from incident.io webhook API (v2 public incident).
const (
	EventIncidentCreatedV2 = "public_incident.incident_created_v2"
	EventIncidentUpdatedV2 = "public_incident.incident_updated_v2"
)

type OnIncident struct{}

type OnIncidentConfiguration struct {
	Events        []string `json:"events"`
	SigningSecret string   `json:"signingSecret"`
}

// OnIncidentMetadata is stored after Setup and includes the webhook URL for the user to copy into incident.io.
type OnIncidentMetadata struct {
	WebhookURL string `json:"webhookUrl" mapstructure:"webhookUrl"`
}

func (t *OnIncident) Name() string {
	return "incident.onIncident"
}

func (t *OnIncident) Label() string {
	return "On Incident"
}

func (t *OnIncident) Description() string {
	return "Listen to incident created and updated events from incident.io"
}

func (t *OnIncident) Documentation() string {
	return `The On Incident trigger starts a workflow execution when incident.io sends webhooks for incident created or updated events.

## Use Cases

- **Incident automation**: Notify Slack, update a status page, or create a Jira ticket when an incident is opened or updated
- **Notification workflows**: Send notifications when incidents are created or their status changes
- **Integration workflows**: Sync incidents with external systems

## Configuration

- **Events**: Select which events to listen for (Incident created, Incident updated)
- **Signing secret**: Paste the signing secret from your incident.io webhook endpoint (Settings > Webhooks). Required to verify webhook authenticity; you can add it after creating the endpoint with the URL shown in the trigger settings.

## Webhook Setup

incident.io does not provide an API to register webhook endpoints. After adding this trigger:

1. Copy the webhook URL shown for this trigger (after saving the canvas).
2. In incident.io go to **Settings > Webhooks** and create a new endpoint with that URL.
3. Subscribe to exactly these events: **Public incident created (v2)** and **Public incident updated (v2)**.
4. Copy the **Signing secret** from the endpoint and paste it into the trigger's Signing secret field.`
}

func (t *OnIncident) Icon() string {
	return "alert-triangle"
}

func (t *OnIncident) Color() string {
	return "gray"
}

func (t *OnIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "events",
			Label:    "Events",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{EventIncidentCreatedV2},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Incident created", Value: EventIncidentCreatedV2},
						{Label: "Incident updated", Value: EventIncidentUpdatedV2},
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

	if ctx.Integration == nil {
		return fmt.Errorf("integration is required to set up the incident.io webhook trigger")
	}

	signingSecret := config.SigningSecret
	if signingSecret == "" {
		if b, getErr := ctx.Integration.GetConfig("webhookSigningSecret"); getErr == nil && len(b) > 0 {
			signingSecret = string(b)
		}
	}

	// Pass only events and hash so the secret is never stored in plaintext in the webhook Configuration column.
	if err := ctx.Integration.RequestWebhook(WebhookConfiguration{
		Events:            config.Events,
		SigningSecretHash: SigningSecretHash(signingSecret),
	}); err != nil {
		return err
	}

	// Persist the signing secret in the encrypted webhook.Secret field (same as Grafana, PagerDuty, etc.).
	if ctx.Webhook != nil && signingSecret != "" {
		if err := ctx.Webhook.SetSecret([]byte(signingSecret)); err != nil {
			return fmt.Errorf("failed to persist webhook signing secret: %w", err)
		}
	}

	// Store webhook URL in metadata so the UI can show it for the user to copy into incident.io.
	if ctx.Webhook != nil {
		webhookURL, err := ctx.Webhook.Setup()
		if err != nil {
			return fmt.Errorf("failed to get webhook URL: %w", err)
		}
		metadata := OnIncidentMetadata{WebhookURL: webhookURL}
		if ctx.Metadata != nil {
			if err := ctx.Metadata.Set(metadata); err != nil {
				return fmt.Errorf("failed to set metadata: %w", err)
			}
		}
	}

	return nil
}

func (t *OnIncident) Actions() []core.Action {
	return nil
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

	// Prefer the secret from the encrypted webhook.Secret field; fall back to integration for backwards compatibility.
	var signingSecret string
	if ctx.Webhook != nil {
		if b, err := ctx.Webhook.GetSecret(); err == nil && len(b) > 0 {
			signingSecret = string(b)
		}
	}
	if signingSecret == "" && ctx.Integration != nil {
		if b, getErr := ctx.Integration.GetConfig("webhookSigningSecret"); getErr == nil && len(b) > 0 {
			signingSecret = string(b)
		}
	}
	if signingSecret == "" {
		return http.StatusForbidden, fmt.Errorf("signing secret is required for webhook verification; add it in the integration (Settings → Integrations) or on the trigger")
	}

	webhookID := ctx.Headers.Get("webhook-id")
	if webhookID == "" {
		webhookID = ctx.Headers.Get("svix-id")
	}
	webhookTimestamp := ctx.Headers.Get("webhook-timestamp")
	if webhookTimestamp == "" {
		webhookTimestamp = ctx.Headers.Get("svix-timestamp")
	}
	webhookSignature := ctx.Headers.Get("webhook-signature")
	if webhookSignature == "" {
		webhookSignature = ctx.Headers.Get("svix-signature")
	}

	if err := VerifySvixSignature(webhookID, webhookTimestamp, webhookSignature, ctx.Body, []byte(signingSecret)); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature: %w", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %w", err)
	}

	// incident.io payload is either:
	// - Svix envelope: { "data": { "event_type": "...", "public_incident.incident_created_v2": {...} }, "type": "...", "timestamp": "..." }
	// - or flat: { "event_type": "...", "public_incident.incident_created_v2": {...} }
	// The incident is under the event type key (e.g. public_incident.incident_created_v2), not "incident".
	source := payload
	if data, ok := payload["data"].(map[string]any); ok && data != nil {
		source = data
	}

	eventType, _ := source["event_type"].(string)
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing event_type in payload")
	}

	if !slices.Contains(config.Events, eventType) {
		return http.StatusOK, nil
	}

	// incident.io puts incident data under the event type key; some docs also show "incident" key.
	incidentData, _ := source["incident"].(map[string]any)
	if incidentData == nil {
		incidentData, _ = source[eventType].(map[string]any)
	}
	emitPayload := map[string]any{
		"event_type": eventType,
		"incident":   incidentData,
	}

	eventName := eventTypeToEventName(eventType)
	if err := ctx.Events.Emit("incident."+eventName, emitPayload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %w", err)
	}

	return http.StatusOK, nil
}

func eventTypeToEventName(eventType string) string {
	switch eventType {
	case EventIncidentCreatedV2:
		return "incident.created"
	case EventIncidentUpdatedV2:
		return "incident.updated"
	default:
		return eventType
	}
}

func (t *OnIncident) Cleanup(ctx core.TriggerContext) error {
	return nil
}
