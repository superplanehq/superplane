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
	Status     []string `json:"status"`
	Severity   []string `json:"severity"`
	Visibility []string `json:"visibility"`
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
	return `The On Event trigger starts a workflow execution when Rootly incident timeline events occur. This is useful for reacting to notes, annotations, and other activities added to an incident's timeline.

## Use Cases

- **Investigation tracking**: Run a workflow when someone adds an investigation note to an incident
- **Status sync**: Forward timeline events to Slack, Jira, or other external systems
- **Automation on notes**: Trigger automations when notes matching certain criteria are posted
- **Audit trail**: Capture timeline events for compliance or auditing purposes

## Configuration

- **Events**: Select which timeline event types to listen for (created, updated)
- **Incident Status** (optional): Only trigger for events on incidents with a specific status (e.g. started, mitigated, resolved)
- **Severity** (optional): Only trigger for events on incidents with a specific severity (e.g. sev0, sev1)
- **Visibility** (optional): Only trigger for events with a specific visibility (internal, external)

## Event Data

Each timeline event includes:
- **event**: The note or annotation text
- **kind**: The type of timeline event (e.g. note, status_update)
- **visibility**: Whether the event is internal or external
- **user_display_name**: The name of the user who created the event
- **incident**: The parent incident details (id, title, status, severity)

## Webhook Setup

This trigger automatically sets up a Rootly webhook endpoint when configured. The endpoint is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnEvent) Icon() string {
	return "alert-triangle"
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
			Default:  []string{"incident_event.created"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "incident_event.created"},
						{Label: "Updated", Value: "incident_event.updated"},
					},
				},
			},
		},
		{
			Name:        "status",
			Label:       "Incident Status",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Only trigger for events on incidents with one of these statuses",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Started", Value: "started"},
						{Label: "Mitigated", Value: "mitigated"},
						{Label: "Resolved", Value: "resolved"},
						{Label: "Cancelled", Value: "cancelled"},
					},
				},
			},
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Only trigger for events on incidents with one of these severities",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "SEV0", Value: "sev0"},
						{Label: "SEV1", Value: "sev1"},
						{Label: "SEV2", Value: "sev2"},
						{Label: "SEV3", Value: "sev3"},
					},
				},
			},
		},
		{
			Name:        "visibility",
			Label:       "Visibility",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Only trigger for events with one of these visibility levels",
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
	var webhook WebhookPayload
	err = json.Unmarshal(ctx.Body, &webhook)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	eventType := webhook.Event.Type

	// Filter by configured event types
	if !slices.Contains(config.Events, eventType) {
		return http.StatusOK, nil
	}

	// Apply optional filters on the event data
	if !matchesEventFilters(config, webhook.Data) {
		return http.StatusOK, nil
	}

	err = ctx.Events.Emit(
		fmt.Sprintf("rootly.%s", eventType),
		buildEventPayload(webhook),
	)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func matchesEventFilters(config OnEventConfiguration, data map[string]any) bool {
	// Filter by visibility (on the event itself)
	if len(config.Visibility) > 0 {
		visibility, _ := data["visibility"].(string)
		if visibility == "" || !slices.Contains(config.Visibility, visibility) {
			return false
		}
	}

	// Filters on the parent incident
	incident, _ := data["incident"].(map[string]any)
	if incident == nil {
		// If there's no incident data but filters are set, skip the event
		if len(config.Status) > 0 || len(config.Severity) > 0 {
			return false
		}
		return true
	}

	// Filter by incident status
	if len(config.Status) > 0 {
		status, _ := incident["status"].(string)
		if status == "" || !slices.Contains(config.Status, status) {
			return false
		}
	}

	// Filter by incident severity
	if len(config.Severity) > 0 {
		severity, _ := incident["severity"].(string)
		if severity == "" || !slices.Contains(config.Severity, severity) {
			return false
		}
	}

	return true
}

func buildEventPayload(webhook WebhookPayload) map[string]any {
	payload := map[string]any{
		"event":     webhook.Event.Type,
		"event_id":  webhook.Event.ID,
		"issued_at": webhook.Event.IssuedAt,
	}

	if webhook.Data != nil {
		payload["incident_event"] = webhook.Data

		// Also include the parent incident at the top level for convenience
		if incident, ok := webhook.Data["incident"].(map[string]any); ok {
			payload["incident"] = incident
		}
	}

	return payload
}

func (t *OnEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}
