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

// OnEvent triggers when an incident timeline event is created or updated in Rootly.
type OnEvent struct{}

// OnEventConfiguration holds the configuration for the OnEvent trigger.
type OnEventConfiguration struct {
	Events     []string `json:"events" mapstructure:"events"`
	Statuses   []string `json:"statuses" mapstructure:"statuses"`
	Severities []string `json:"severities" mapstructure:"severities"`
	Services   []string `json:"services" mapstructure:"services"`
	Teams      []string `json:"teams" mapstructure:"teams"`
	Sources    []string `json:"sources" mapstructure:"sources"`
	Visibility string   `json:"visibility" mapstructure:"visibility"`
	EventKinds []string `json:"event_kinds" mapstructure:"event_kinds"`
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
	return `The On Event trigger starts a workflow execution when Rootly incident timeline events occur (such as notes or annotations).

## Use Cases

- **Note notifications**: Run a workflow when someone adds a note to an incident (e.g. notify channel, update Jira)
- **Timeline sync**: Sync timeline events to Slack or external systems
- **Investigation automation**: Run automation when investigation notes are added
- **Audit trail**: Track and forward incident annotations to external systems

## Configuration

- **Events**: Select which timeline event types to listen for (created, updated)
- **Incident status** (optional): Filter by incident status (e.g. open, resolved)
- **Severity** (optional): Filter by incident severity
- **Service** (optional): Filter by service name
- **Team** (optional): Filter by team name
- **Event source** (optional): Filter by event source
- **Visibility** (optional): Filter by visibility (external/internal)
- **Event kind** (optional): Filter by event kind (e.g. note, annotation)

## Event Data

Each timeline event includes:
- **id**: Event ID
- **event**: Event type (incident_event.created, incident_event.updated)
- **kind**: Event kind (e.g. note, annotation)
- **occurred_at**: When the event occurred
- **created_at**: When the event was created
- **user_display_name**: Display name of the user who created the event
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
			Name:        "statuses",
			Label:       "Incident Status",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter by incident status (leave empty to receive all)",
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
			Name:        "severities",
			Label:       "Severity",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter by incident severity (leave empty to receive all)",
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
			Name:        "services",
			Label:       "Service",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by service name (comma-separated for multiple)",
			Placeholder: "e.g. api-gateway, auth-service",
		},
		{
			Name:        "teams",
			Label:       "Team",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by team name (comma-separated for multiple)",
			Placeholder: "e.g. platform, infrastructure",
		},
		{
			Name:        "sources",
			Label:       "Event Source",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter by event source (leave empty to receive all)",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Web", Value: "web"},
						{Label: "Slack", Value: "slack"},
						{Label: "API", Value: "api"},
						{Label: "Email", Value: "email"},
						{Label: "System", Value: "system"},
					},
				},
			},
		},
		{
			Name:        "visibility",
			Label:       "Visibility",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Filter by event visibility (leave empty to receive all)",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "All", Value: ""},
						{Label: "Internal", Value: "internal"},
						{Label: "External", Value: "external"},
					},
				},
			},
		},
		{
			Name:        "event_kinds",
			Label:       "Event Kind",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter by event kind (leave empty to receive all)",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Note", Value: "note"},
						{Label: "Status Update", Value: "status_update"},
						{Label: "Severity Update", Value: "severity_update"},
						{Label: "Assignment", Value: "assignment"},
						{Label: "Action Item", Value: "action_item"},
						{Label: "Postmortem", Value: "postmortem"},
						{Label: "Alert", Value: "alert"},
						{Label: "Page", Value: "page"},
						{Label: "Slack Message", Value: "slack_message"},
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
	var webhook EventWebhookPayload
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

	// Apply optional filters
	if !matchesEventFilters(webhook.Data, config) {
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

// EventWebhookPayload represents the Rootly webhook payload for timeline events.
type EventWebhookPayload struct {
	Event EventWebhookEvent `json:"event"`
	Data  map[string]any    `json:"data"`
}

// EventWebhookEvent represents the event metadata in a Rootly timeline event webhook.
type EventWebhookEvent struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	IssuedAt string `json:"issued_at"`
}

// matchesEventFilters checks if the event data matches the configured filters.
func matchesEventFilters(data map[string]any, config OnEventConfiguration) bool {
	if data == nil {
		return true
	}

	// Filter by incident status
	if len(config.Statuses) > 0 {
		if incident, ok := data["incident"].(map[string]any); ok {
			if status, ok := incident["status"].(string); ok {
				if !slices.Contains(config.Statuses, status) {
					return false
				}
			}
		}
	}

	// Filter by severity
	if len(config.Severities) > 0 {
		if incident, ok := data["incident"].(map[string]any); ok {
			if severity, ok := incident["severity"].(string); ok {
				if !slices.Contains(config.Severities, severity) {
					return false
				}
			}
		}
	}

	// Filter by service
	if len(config.Services) > 0 {
		matched := false
		if incident, ok := data["incident"].(map[string]any); ok {
			if services, ok := incident["services"].([]any); ok {
				for _, s := range services {
					if service, ok := s.(map[string]any); ok {
						if name, ok := service["name"].(string); ok {
							if slices.Contains(config.Services, name) {
								matched = true
								break
							}
						}
					}
				}
			}
		}
		if !matched {
			return false
		}
	}

	// Filter by team
	if len(config.Teams) > 0 {
		matched := false
		if incident, ok := data["incident"].(map[string]any); ok {
			if teams, ok := incident["teams"].([]any); ok {
				for _, t := range teams {
					if team, ok := t.(map[string]any); ok {
						if name, ok := team["name"].(string); ok {
							if slices.Contains(config.Teams, name) {
								matched = true
								break
							}
						}
					}
				}
			}
		}
		if !matched {
			return false
		}
	}

	// Filter by event source
	if len(config.Sources) > 0 {
		if source, ok := data["source"].(string); ok {
			if !slices.Contains(config.Sources, source) {
				return false
			}
		}
	}

	// Filter by visibility
	if config.Visibility != "" {
		if visibility, ok := data["visibility"].(string); ok {
			if visibility != config.Visibility {
				return false
			}
		}
	}

	// Filter by event kind
	if len(config.EventKinds) > 0 {
		if kind, ok := data["kind"].(string); ok {
			if !slices.Contains(config.EventKinds, kind) {
				return false
			}
		}
	}

	return true
}

// buildEventPayload constructs the output payload for the event.
func buildEventPayload(webhook EventWebhookPayload) map[string]any {
	payload := map[string]any{
		"event":     webhook.Event.Type,
		"event_id":  webhook.Event.ID,
		"issued_at": webhook.Event.IssuedAt,
	}

	if webhook.Data != nil {
		// Extract key fields from the event data
		if id, ok := webhook.Data["id"].(string); ok {
			payload["id"] = id
		}

		if kind, ok := webhook.Data["kind"].(string); ok {
			payload["kind"] = kind
		}

		if occurredAt, ok := webhook.Data["occurred_at"].(string); ok {
			payload["occurred_at"] = occurredAt
		}

		if createdAt, ok := webhook.Data["created_at"].(string); ok {
			payload["created_at"] = createdAt
		}

		if userDisplayName, ok := webhook.Data["user_display_name"].(string); ok {
			payload["user_display_name"] = userDisplayName
		}

		// Include the full incident data
		if incident, ok := webhook.Data["incident"].(map[string]any); ok {
			payload["incident"] = incident
		}

		// Include additional useful fields
		if body, ok := webhook.Data["body"].(string); ok {
			payload["body"] = body
		}

		if source, ok := webhook.Data["source"].(string); ok {
			payload["source"] = source
		}

		if visibility, ok := webhook.Data["visibility"].(string); ok {
			payload["visibility"] = visibility
		}
	}

	return payload
}

func (t *OnEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// ExampleData returns example data for documentation and UI purposes.
func (t *OnEvent) ExampleData() map[string]any {
	return map[string]any{
		"id":                "evt-timeline-123",
		"event":             "incident_event.created",
		"event_id":          "evt-123456",
		"issued_at":         "2026-01-19T12:00:00Z",
		"kind":              "note",
		"occurred_at":       "2026-01-19T11:58:00Z",
		"created_at":        "2026-01-19T12:00:00Z",
		"user_display_name": "John Doe",
		"body":              "Initial investigation shows the issue is related to a recent deployment.",
		"source":            "web",
		"visibility":        "internal",
		"incident": map[string]any{
			"id":       "inc-abc123",
			"title":    "API latency spike detected",
			"status":   "started",
			"severity": "sev2",
		},
	}
}
