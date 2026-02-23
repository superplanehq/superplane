package incident

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

// Event type names from incident.io webhook API (v2 public incident).
const (
	EventIncidentCreatedV2 = "public_incident.incident_created_v2"
	EventIncidentUpdatedV2 = "public_incident.incident_updated_v2"
)

type OnIncident struct{}

type OnIncidentConfiguration struct {
	Events                  []string `json:"events" mapstructure:"events"`
	SigningSecretConfigured bool     `json:"signingSecretConfigured" mapstructure:"signingSecretConfigured"` // Set by UI after setSecret; not used for verification
}

// OnIncidentMetadata is stored after Setup and includes the webhook URL and signing-secret status for the UI.
type OnIncidentMetadata struct {
	WebhookURL              string `json:"webhookUrl" mapstructure:"webhookUrl"`
	SigningSecretConfigured bool   `json:"signingSecretConfigured" mapstructure:"signingSecretConfigured"`
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
- **Webhook signing secret**: Use the **Set signing secret** action below (after creating the webhook in incident.io) to store the signing secret. It is stored securely and never in the workflow configuration.

## Webhook Setup

incident.io does not provide an API to register webhook endpoints. After adding this trigger:

1. Save the canvas to generate the webhook URL, then copy it from this panel.
2. In incident.io go to **Settings > Webhooks** and create a new endpoint with that URL.
3. Subscribe to **Public incident created (v2)** and **Public incident updated (v2)**.
4. Copy the **Signing secret** from the endpoint, then use **Set signing secret** below to store it securely.`
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
			Default:  []string{EventIncidentCreatedV2, EventIncidentUpdatedV2},
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

	if err := ctx.Integration.RequestWebhook(WebhookConfiguration{
		Events: config.Events,
	}); err != nil {
		return err
	}

	var webhookURL string
	if ctx.Webhook != nil {
		var err error
		webhookURL, err = ctx.Webhook.Setup()
		if err != nil {
			return fmt.Errorf("failed to get webhook URL: %w", err)
		}
	}

	// Store webhook URL in metadata; preserve SigningSecretConfigured from existing metadata (set by setSecret action).
	if ctx.Metadata != nil {
		metadata := OnIncidentMetadata{}
		_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
		metadata.WebhookURL = webhookURL
		if err := ctx.Metadata.Set(metadata); err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}
	}

	return nil
}

func (t *OnIncident) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "setSecret",
			Description:    "Set or clear the webhook signing secret from your incident.io endpoint",
			UserAccessible: true,
			Parameters: []configuration.Field{
				{
					Name:        "webhookSigningSecret",
					Label:       "Webhook signing secret",
					Type:        configuration.FieldTypeString,
					Required:    false,
					Sensitive:   true,
					Description: "Paste the signing secret from your incident.io webhook endpoint (Settings → Webhooks). Leave empty to clear.",
					Placeholder: "whsec_...",
				},
			},
		},
	}
}

func (t *OnIncident) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	if ctx.Name != "setSecret" {
		return nil, fmt.Errorf("action %s not supported", ctx.Name)
	}
	return t.setSecret(ctx)
}

func (t *OnIncident) setSecret(ctx core.TriggerActionContext) (map[string]any, error) {
	if ctx.Webhook == nil {
		return nil, fmt.Errorf("webhook is not available")
	}
	var secret string
	if v, ok := ctx.Parameters["webhookSigningSecret"]; ok && v != nil {
		if s, ok := v.(string); ok {
			secret = strings.TrimSpace(s)
		}
	}
	if err := ctx.Webhook.SetSecret([]byte(secret)); err != nil {
		return nil, fmt.Errorf("failed to set webhook signing secret: %w", err)
	}
	configured := secret != ""
	if ctx.Metadata != nil {
		metadata := OnIncidentMetadata{}
		_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
		metadata.SigningSecretConfigured = configured
		if err := ctx.Metadata.Set(metadata); err != nil {
			return nil, fmt.Errorf("failed to update metadata: %w", err)
		}
	}
	return map[string]any{"ok": true, "signingSecretConfigured": configured}, nil
}

func (t *OnIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	if ctx.Logger != nil {
		ctx.Logger.Infof("incident webhook: received for workflow %s", ctx.WorkflowID)
	}

	config := OnIncidentConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	signingSecret := resolveSigningSecret(ctx)
	if signingSecret == "" {
		return http.StatusForbidden, fmt.Errorf("signing secret is required for webhook verification; use the Set signing secret action for this trigger")
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

	// Accept if event type is in config; if config.Events is empty (e.g. old node), accept both known event types so we don't silently drop
	acceptedEvents := config.Events
	if len(acceptedEvents) == 0 {
		acceptedEvents = []string{EventIncidentCreatedV2, EventIncidentUpdatedV2}
	}
	if !slices.Contains(acceptedEvents, eventType) {
		if ctx.Logger != nil {
			ctx.Logger.Infof("incident webhook: event type %q not in trigger config (configured: %v), acknowledging without emitting", eventType, config.Events)
		}
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
	payloadType := "incident." + eventName
	if err := ctx.Events.Emit(payloadType, emitPayload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %w", err)
	}
	if ctx.Logger != nil {
		ctx.Logger.Infof("incident webhook: emitted %s for workflow %s", payloadType, ctx.WorkflowID)
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

// resolveSigningSecret returns the webhook signing secret for verification (stored via setSecret action).
func resolveSigningSecret(ctx core.WebhookRequestContext) string {
	if ctx.Webhook == nil {
		return ""
	}
	b, err := ctx.Webhook.GetSecret()
	if err != nil || len(b) == 0 {
		return ""
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return ""
	}
	return s
}
