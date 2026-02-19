package rootly

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

const rootlyIncidentEventPayloadType = "rootly.onEvent"

var rootlyIncidentEventWebhookTypes = []string{
	"incident_event.created",
	"incident_event.updated",
}

type OnEvent struct{}

type OnEventConfiguration struct {
	Visibility string `json:"visibility" mapstructure:"visibility"`
}

type OnEventMetadata struct {
	EventStates map[string]string `json:"eventStates"`
}

func (t *OnEvent) Name() string {
	return "rootly.onEvent"
}

func (t *OnEvent) Label() string {
	return "On Event"
}

func (t *OnEvent) Description() string {
	return "Listen to incident timeline events"
}

func (t *OnEvent) Documentation() string {
	return `The On Event trigger starts a workflow execution when Rootly incident timeline events are created or updated.

## Use Cases

- **Note automation**: Run workflows when investigation notes are added
- **Timeline sync**: Sync incident timeline events to Slack or external systems
- **Annotation tracking**: Track updates to incident annotations
- **Audit logging**: Capture timeline events for compliance or reporting

## Configuration

- **Visibility**: Optional filter by event visibility (internal or external)

## Event Data

Each incident event includes:
- **id**: Event ID
- **event**: Event content
- **event_type**: Event type (incident_event.created or incident_event.updated)
- **kind**: Event kind
- **occurred_at**: When the event occurred
- **created_at**: When the event was created
- **incident**: Incident information

## Webhook Setup

This trigger automatically sets up a Rootly webhook endpoint when configured. The endpoint is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnEvent) Icon() string {
	return "message-square"
}

func (t *OnEvent) Color() string {
	return "gray"
}

func (t *OnEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "visibility",
			Label:    "Visibility",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Internal", Value: "internal"},
						{Label: "External", Value: "external"},
					},
				},
			},
			Description: "Only emit events with this visibility",
		},
	}
}

func (t *OnEvent) Setup(ctx core.TriggerContext) error {
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Events: []string{"incident_event.created", "incident_event.updated"},
	})
}

func (t *OnEvent) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnEvent) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	signature := ctx.Headers.Get("X-Rootly-Signature")
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting secret: %v", err)
	}

	if err := verifyWebhookSignature(signature, ctx.Body, secret); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature: %v", err)
	}

	var webhook WebhookPayload
	if err := json.Unmarshal(ctx.Body, &webhook); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	if !slices.Contains(rootlyIncidentEventWebhookTypes, webhook.Event.Type) {
		return http.StatusOK, nil
	}

	data := webhook.Data
	if data == nil {
		return http.StatusOK, nil
	}

	incidentEvent := extractEventFromData(data)
	if incidentEvent == nil {
		return http.StatusOK, nil
	}

	metadata := loadOnEventMetadata(ctx.Metadata)
	updatedStates := map[string]string{}
	for key, value := range metadata.EventStates {
		updatedStates[key] = value
	}

	emitted := 0
	eventID := extractString(incidentEvent, "id")
	fingerprint := eventFingerprint(incidentEvent)

	if eventID != "" {
		updatedStates[eventID] = fingerprint
	}

	if config.Visibility != "" && !strings.EqualFold(config.Visibility, extractString(incidentEvent, "visibility")) {
		return http.StatusOK, nil
	}

	if eventID != "" {
		if previous, exists := metadata.EventStates[eventID]; exists && previous == fingerprint {
			return http.StatusOK, nil
		}
	}

	payload := buildIncidentEventPayload(nil, incidentEvent, webhook.Event.Type)
	if err := ctx.Events.Emit(rootlyIncidentEventPayloadType, payload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}
	emitted++

	if ctx.Metadata != nil {
		if err := ctx.Metadata.Set(OnEventMetadata{EventStates: updatedStates}); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error updating metadata: %v", err)
		}
	}

	if emitted == 0 {
		return http.StatusOK, nil
	}

	return http.StatusOK, nil
}

func (t *OnEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func loadOnEventMetadata(ctx core.MetadataContext) OnEventMetadata {
	if ctx == nil {
		return OnEventMetadata{EventStates: map[string]string{}}
	}

	metadata := OnEventMetadata{}
	if err := mapstructure.Decode(ctx.Get(), &metadata); err != nil || metadata.EventStates == nil {
		return OnEventMetadata{EventStates: map[string]string{}}
	}

	return metadata
}

func extractEventFromData(data map[string]any) map[string]any {
	if event, ok := data["incident_event"].(map[string]any); ok {
		if isIncidentEventPayload(event) {
			return event
		}
	}

	if isIncidentEventPayload(data) {
		return data
	}

	return nil
}

func buildIncidentEventPayload(incident map[string]any, incidentEvent map[string]any, eventType string) map[string]any {
	if incident == nil {
		if incidentID := extractString(incidentEvent, "incident_id", "incidentId"); incidentID != "" {
			incident = map[string]any{"id": incidentID}
		}
	}

	payload := map[string]any{
		"id":          extractString(incidentEvent, "id"),
		"event":       extractString(incidentEvent, "event"),
		"event_type":  eventType,
		"kind":        extractString(incidentEvent, "kind"),
		"occurred_at": extractString(incidentEvent, "occurred_at"),
		"created_at":  extractString(incidentEvent, "created_at"),
		"visibility":  extractString(incidentEvent, "visibility"),
		"incident":    incident,
	}

	return payload
}

func extractString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := data[key]
		if !ok || value == nil {
			continue
		}

		str, ok := value.(string)
		if ok && str != "" {
			return str
		}
	}

	return ""
}

func eventFingerprint(event map[string]any) string {
	if value := extractString(event, "updated_at"); value != "" {
		return value
	}

	if value := extractString(event, "created_at"); value != "" {
		return value
	}

	if value := extractString(event, "occurred_at"); value != "" {
		return value
	}

	raw, err := json.Marshal(event)
	if err != nil {
		return ""
	}

	return string(raw)
}

func isIncidentEventPayload(data map[string]any) bool {
	if data == nil {
		return false
	}

	if extractString(data, "event", "kind") != "" {
		return true
	}

	if extractString(data, "occurred_at", "created_at") != "" {
		return true
	}

	return false
}
