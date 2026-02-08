package rootly

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnEvent struct{}

type OnEventConfiguration struct {
	Events     []string `json:"events"`
	EventKinds []string `json:"eventKinds,omitempty"`
	Visibility []string `json:"visibility,omitempty"`
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
	return `The On Event trigger starts a workflow execution when Rootly incident timeline events occur.

## Use Cases

- **Timeline automation**: Run a workflow when someone adds a note to an incident
- **Sync timeline events**: Sync notes and annotations to Slack or external systems
- **Investigation tracking**: React to new investigation notes being added
- **Status updates**: Trigger notifications when timeline annotations are created

## Configuration

- **Events**: Select which timeline events to listen for (created, updated)
- **Event Kinds** (optional): Filter by event kind (note, status_update, action_item, etc.)
- **Visibility** (optional): Filter by visibility (internal, external)

## Event Data

Each timeline event includes:
- **event**: Event type (timeline_event.created, timeline_event.updated)
- **id**: Timeline event ID
- **kind**: Event kind (note, annotation, status_update, etc.)
- **body**: Event content/body
- **occurred_at**: When the event occurred
- **created_at**: When the event was created
- **user_display_name**: Who created the event
- **incident**: Associated incident information

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
			Name:     "events",
			Label:    "Events",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"timeline_event.created"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "timeline_event.created"},
						{Label: "Updated", Value: "timeline_event.updated"},
					},
				},
			},
		},
		{
			Name:        "eventKinds",
			Label:       "Event Kinds",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter by event kind (leave empty for all)",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Note", Value: "note"},
						{Label: "Annotation", Value: "annotation"},
						{Label: "Status Update", Value: "status_update"},
						{Label: "Action Item", Value: "action_item"},
						{Label: "Alert", Value: "alert"},
						{Label: "Page", Value: "page"},
						{Label: "Slack Message", Value: "slack_message"},
					},
				},
			},
		},
		{
			Name:        "visibility",
			Label:       "Visibility",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter by visibility (leave empty for all)",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Internal", Value: "internal"},
						{Label: "External", Value: "external"},
					},
				},
			},
		},
	}
}

func (t *OnEvent) Setup(ctx core.TriggerContext) error {
	config := OnEventConfiguration{}
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

func (t *OnEvent) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnEvent) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnEventConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Verify signature
	signature := ctx.Headers.Get("X-Rootly-Signature")
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting secret: %v", err)
	}

	if err := verifyWebhookSignature(signature, ctx.Body, secret); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature: %v", err)
	}

	// Parse webhook payload
	var webhook TimelineEventWebhookPayload
	err = json.Unmarshal(ctx.Body, &webhook)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	eventType := webhook.Event.Type

	// Filter by event type
	if !slices.Contains(config.Events, eventType) {
		return http.StatusOK, nil
	}

	// Filter by event kind if specified
	if len(config.EventKinds) > 0 {
		eventKind, ok := webhook.Data["kind"].(string)
		if !ok || !slices.Contains(config.EventKinds, eventKind) {
			return http.StatusOK, nil
		}
	}

	// Filter by visibility if specified
	if len(config.Visibility) > 0 {
		visibility, ok := webhook.Data["visibility"].(string)
		if !ok || !slices.Contains(config.Visibility, visibility) {
			return http.StatusOK, nil
		}
	}

	err = ctx.Events.Emit(
		fmt.Sprintf("rootly.%s", eventType),
		buildTimelineEventPayload(webhook),
	)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

// TimelineEventWebhookPayload represents the Rootly webhook payload for timeline events
type TimelineEventWebhookPayload struct {
	Event WebhookEvent   `json:"event"`
	Data  map[string]any `json:"data"`
}

func buildTimelineEventPayload(webhook TimelineEventWebhookPayload) map[string]any {
	payload := map[string]any{
		"event":     webhook.Event.Type,
		"event_id":  webhook.Event.ID,
		"issued_at": webhook.Event.IssuedAt,
	}

	// Extract timeline event fields
	if webhook.Data != nil {
		if id, ok := webhook.Data["id"]; ok {
			payload["id"] = id
		}
		if kind, ok := webhook.Data["kind"]; ok {
			payload["kind"] = kind
		}
		if body, ok := webhook.Data["body"]; ok {
			payload["body"] = body
		}
		if occurredAt, ok := webhook.Data["occurred_at"]; ok {
			payload["occurred_at"] = occurredAt
		}
		if createdAt, ok := webhook.Data["created_at"]; ok {
			payload["created_at"] = createdAt
		}
		if userDisplayName, ok := webhook.Data["user_display_name"]; ok {
			payload["user_display_name"] = userDisplayName
		}
		if visibility, ok := webhook.Data["visibility"]; ok {
			payload["visibility"] = visibility
		}
		if incident, ok := webhook.Data["incident"]; ok {
			payload["incident"] = incident
		}
	}

	return payload
}

func (t *OnEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}
