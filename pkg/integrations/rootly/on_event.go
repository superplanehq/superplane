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
	IncidentStatus []string `json:"incidentStatus,omitempty"`
	Severity       []string `json:"severity,omitempty"`
	Service        []string `json:"service,omitempty"`
	Team           []string `json:"team,omitempty"`
	EventSource    []string `json:"eventSource,omitempty"`
	Visibility     []string `json:"visibility,omitempty"`
	EventKind      []string `json:"eventKind,omitempty"`
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
	return `The On Event trigger starts a workflow execution when incident timeline events are created or updated in Rootly.

## Use Cases

- **Timeline automation**: React to notes and annotations added to incidents
- **External sync**: Sync timeline events to Slack or external systems  
- **Investigation workflows**: Trigger actions when investigation notes are added
- **Communication workflows**: Send notifications when incident updates occur

## Configuration

Configure optional filters to listen to specific types of events:
- **Incident Status**: Filter by incident status (open, resolved, etc.)
- **Severity**: Filter by incident severity level
- **Service**: Filter by specific services
- **Team**: Filter by team assignments
- **Event Source**: Filter by event source (web, api, slack, etc.)
- **Visibility**: Filter by visibility (external, internal)
- **Event Kind**: Filter by event types (note, annotation, status_update, etc.)

## Event Data

Each timeline event includes:
- **id**: Unique event identifier
- **event**: Event content/text
- **kind**: Type of event (note, annotation, etc.)
- **occurred_at**: When the event occurred
- **created_at**: When the event was created in Rootly
- **user_display_name**: Name of the user who created the event
- **incident**: Complete incident information

## Webhook Setup

This trigger automatically sets up a Rootly webhook endpoint when configured. The endpoint is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnEvent) Icon() string {
	return "message-circle"
}

func (t *OnEvent) Color() string {
	return "blue"
}

func (t *OnEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentStatus",
			Label:       "Incident Status",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter events by incident status",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Open", Value: "open"},
						{Label: "Investigating", Value: "investigating"},
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
			Description: "Filter events by incident severity",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Critical", Value: "critical"},
						{Label: "High", Value: "high"},
						{Label: "Medium", Value: "medium"},
						{Label: "Low", Value: "low"},
					},
				},
			},
		},
		{
			Name:        "service",
			Label:       "Service",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter events by service",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "API Gateway", Value: "api-gateway"},
						{Label: "Database", Value: "database"},
						{Label: "Frontend", Value: "frontend"},
						{Label: "Backend", Value: "backend"},
					},
				},
			},
		},
		{
			Name:        "team",
			Label:       "Team",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter events by team name",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Backend Engineering", Value: "backend"},
						{Label: "Frontend Engineering", Value: "frontend"},
						{Label: "Infrastructure", Value: "infrastructure"},
						{Label: "Database Team", Value: "database"},
						{Label: "DevOps", Value: "devops"},
					},
				},
			},
		},
		{
			Name:        "eventSource",
			Label:       "Event Source",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter events by source",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Web Interface", Value: "web"},
						{Label: "API", Value: "api"},
						{Label: "Slack", Value: "slack"},
						{Label: "Email", Value: "email"},
						{Label: "Mobile", Value: "mobile"},
					},
				},
			},
		},
		{
			Name:        "visibility",
			Label:       "Visibility",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter events by visibility",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "External", Value: "external"},
						{Label: "Internal", Value: "internal"},
					},
				},
			},
		},
		{
			Name:        "eventKind",
			Label:       "Event Kind",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter events by type",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Note", Value: "note"},
						{Label: "Annotation", Value: "annotation"},
						{Label: "Status Update", Value: "status_update"},
						{Label: "Assignment", Value: "assignment"},
						{Label: "Escalation", Value: "escalation"},
					},
				},
			},
		},
	}
}

func (t *OnEvent) Setup(ctx core.TriggerContext) error {
	// Request webhook for incident event types that include timeline events
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Events: []string{
			"incident_event.created",
			"incident_event.updated", 
		},
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

	// Filter for incident event types only
	if !slices.Contains([]string{"incident_event.created", "incident_event.updated"}, eventType) {
		return http.StatusOK, nil
	}

	// Apply filters based on configuration
	if !passesFilters(webhook, config) {
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

func passesFilters(webhook WebhookPayload, config OnEventConfiguration) bool {
	// Extract event data
	eventData, ok := webhook.Data["incident_event"].(map[string]any)
	if !ok {
		return false
	}

	incidentData, ok := webhook.Data["incident"].(map[string]any)
	if !ok {
		return false
	}

	// Filter by incident status
	if len(config.IncidentStatus) > 0 {
		status, _ := incidentData["status"].(string)
		if !slices.Contains(config.IncidentStatus, status) {
			return false
		}
	}

	// Filter by severity
	if len(config.Severity) > 0 {
		severity, _ := incidentData["severity"].(string)
		if !slices.Contains(config.Severity, severity) {
			return false
		}
	}

	// Filter by service
	if len(config.Service) > 0 {
		services, _ := incidentData["services"].([]any)
		found := false
		for _, service := range services {
			if serviceMap, ok := service.(map[string]any); ok {
				if serviceID, ok := serviceMap["id"].(string); ok {
					if slices.Contains(config.Service, serviceID) {
						found = true
						break
					}
				}
			}
		}
		if !found {
			return false
		}
	}

	// Filter by team
	if len(config.Team) > 0 {
		teams, _ := incidentData["teams"].([]any)
		found := false
		for _, team := range teams {
			if teamMap, ok := team.(map[string]any); ok {
				if teamName, ok := teamMap["name"].(string); ok {
					if slices.Contains(config.Team, teamName) {
						found = true
						break
					}
				}
			}
		}
		if !found {
			return false
		}
	}

	// Filter by event source
	if len(config.EventSource) > 0 {
		source, _ := eventData["source"].(string)
		if !slices.Contains(config.EventSource, source) {
			return false
		}
	}

	// Filter by visibility
	if len(config.Visibility) > 0 {
		visibility, _ := eventData["visibility"].(string)
		if !slices.Contains(config.Visibility, visibility) {
			return false
		}
	}

	// Filter by event kind
	if len(config.EventKind) > 0 {
		kind, _ := eventData["kind"].(string)
		if !slices.Contains(config.EventKind, kind) {
			return false
		}
	}

	return true
}

func buildEventPayload(webhook WebhookPayload) map[string]any {
	payload := map[string]any{
		"event_type": webhook.Event.Type,
		"event_id":   webhook.Event.ID,
		"issued_at":  webhook.Event.IssuedAt,
	}

	if eventData, ok := webhook.Data["incident_event"].(map[string]any); ok {
		// Extract event details
		if id, ok := eventData["id"]; ok {
			payload["id"] = id
		}
		if event, ok := eventData["event"]; ok {
			payload["event"] = event
		}
		if kind, ok := eventData["kind"]; ok {
			payload["kind"] = kind
		}
		if occurredAt, ok := eventData["occurred_at"]; ok {
			payload["occurred_at"] = occurredAt
		}
		if createdAt, ok := eventData["created_at"]; ok {
			payload["created_at"] = createdAt
		}
		if userDisplayName, ok := eventData["user_display_name"]; ok {
			payload["user_display_name"] = userDisplayName
		}
	}

	if webhook.Data != nil {
		if incidentData, ok := webhook.Data["incident"]; ok {
			payload["incident"] = incidentData
		}
	}

	return payload
}

func (t *OnEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}