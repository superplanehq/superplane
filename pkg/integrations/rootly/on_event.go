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

type OnEvent struct{}

type OnEventConfiguration struct {
	Events         []string `json:"events" mapstructure:"events"`
	Visibility     string   `json:"visibility" mapstructure:"visibility"`
	EventKind      string   `json:"eventKind" mapstructure:"eventKind"`
	IncidentStatus string   `json:"incidentStatus" mapstructure:"incidentStatus"`
	Severity       string   `json:"severity" mapstructure:"severity"`
	Service        string   `json:"service" mapstructure:"service"`
	Team           string   `json:"team" mapstructure:"team"`
	EventSource    string   `json:"eventSource" mapstructure:"eventSource"`
}

func (t *OnEvent) Name() string {
	return "rootly.onEvent"
}

func (t *OnEvent) Label() string {
	return "On Event"
}

func (t *OnEvent) Description() string {
	return "Trigger when an incident timeline event is created or updated"
}

func (t *OnEvent) Documentation() string {
	return `The On Event trigger starts a workflow execution when an incident timeline event is created or updated in Rootly.

## Use Cases

- **Timeline monitoring**: React when someone adds a note or annotation to an incident
- **Event sync**: Sync timeline events to Slack, Jira, or external systems
- **Investigation tracking**: Run automation when investigation notes are added
- **Notification workflows**: Send notifications when specific types of events occur

## Configuration

- **Events**: Select which event types to listen for (created, updated)
- **Visibility** (optional): Filter by event visibility (internal or external)
- **Event Kind** (optional): Filter by event kind (e.g. note, status_change)
- **Incident Status** (optional): Filter by incident status (e.g. started, mitigated, resolved)
- **Severity** (optional): Filter by incident severity (e.g. sev0, sev1)
- **Service** (optional): Filter by service name
- **Team** (optional): Filter by team name
- **Event Source** (optional): Filter by event source

## Event Data

Each event includes:
- **id**: The timeline event ID
- **event**: The event content/description
- **kind**: The type of event (note, status_change, etc.)
- **occurred_at**: When the event occurred
- **created_at**: When the event was created
- **user_display_name**: The name of the user who created the event
- **incident**: The associated incident information

## Webhook Setup

This trigger automatically sets up a Rootly webhook endpoint when configured. The endpoint is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnEvent) Icon() string {
	return "file-text"
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
			Name:  "visibility",
			Label: "Visibility",
			Type:  configuration.FieldTypeSelect,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Internal", Value: "internal"},
						{Label: "External", Value: "external"},
					},
				},
			},
			Description: "Filter events by visibility. Leave empty to receive all events.",
		},
		{
			Name:        "eventKind",
			Label:       "Event Kind",
			Type:        configuration.FieldTypeString,
			Description: "Filter by event kind (e.g. note, status_change). Leave empty for all kinds.",
		},
		{
			Name:        "incidentStatus",
			Label:       "Incident Status",
			Type:        configuration.FieldTypeString,
			Description: "Filter by incident status (e.g. started, mitigated, resolved). Leave empty for all statuses.",
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeString,
			Description: "Filter by incident severity (e.g. sev0, sev1). Leave empty for all severities.",
		},
		{
			Name:        "service",
			Label:       "Service",
			Type:        configuration.FieldTypeString,
			Description: "Filter by service name. Leave empty for all services.",
		},
		{
			Name:        "team",
			Label:       "Team",
			Type:        configuration.FieldTypeString,
			Description: "Filter by team name. Leave empty for all teams.",
		},
		{
			Name:        "eventSource",
			Label:       "Event Source",
			Type:        configuration.FieldTypeString,
			Description: "Filter by event source. Leave empty for all sources.",
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

	// Apply optional filters
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
	if data == nil {
		return true
	}

	// Filter by visibility
	if config.Visibility != "" {
		visibility, _ := data["visibility"].(string)
		if !strings.EqualFold(visibility, config.Visibility) {
			return false
		}
	}

	// Filter by event kind
	if config.EventKind != "" {
		kind, _ := data["kind"].(string)
		if !strings.EqualFold(kind, config.EventKind) {
			return false
		}
	}

	// Filter by event source
	if config.EventSource != "" {
		source, _ := data["source"].(string)
		if !strings.EqualFold(source, config.EventSource) {
			return false
		}
	}

	// Filters that require incident data
	incident, _ := data["incident"].(map[string]any)

	if config.IncidentStatus != "" {
		if incident == nil {
			return false
		}
		status, _ := incident["status"].(string)
		if !strings.EqualFold(status, config.IncidentStatus) {
			return false
		}
	}

	if config.Severity != "" {
		if incident == nil {
			return false
		}
		severity := severityString(incident["severity"])
		if !strings.EqualFold(severity, config.Severity) {
			return false
		}
	}

	if config.Service != "" {
		if !matchesService(incident, config.Service) {
			return false
		}
	}

	if config.Team != "" {
		if !matchesTeam(incident, config.Team) {
			return false
		}
	}

	return true
}

func matchesService(incident map[string]any, service string) bool {
	if incident == nil {
		return false
	}

	services, ok := incident["services"].([]any)
	if !ok {
		return false
	}

	for _, s := range services {
		svc, ok := s.(map[string]any)
		if !ok {
			continue
		}
		name, _ := svc["name"].(string)
		slug, _ := svc["slug"].(string)
		if strings.EqualFold(name, service) || strings.EqualFold(slug, service) {
			return true
		}
	}

	return false
}

func matchesTeam(incident map[string]any, team string) bool {
	if incident == nil {
		return false
	}

	groups, ok := incident["groups"].([]any)
	if !ok {
		return false
	}

	for _, g := range groups {
		grp, ok := g.(map[string]any)
		if !ok {
			continue
		}
		name, _ := grp["name"].(string)
		slug, _ := grp["slug"].(string)
		if strings.EqualFold(name, team) || strings.EqualFold(slug, team) {
			return true
		}
	}

	return false
}

func buildEventPayload(webhook WebhookPayload) map[string]any {
	payload := map[string]any{
		"event":     webhook.Event.Type,
		"event_id":  webhook.Event.ID,
		"issued_at": webhook.Event.IssuedAt,
	}

	if webhook.Data != nil {
		if id, ok := webhook.Data["id"]; ok {
			payload["id"] = id
		}
		if event, ok := webhook.Data["event"]; ok {
			payload["event_content"] = event
		}
		if kind, ok := webhook.Data["kind"]; ok {
			payload["kind"] = kind
		}
		if visibility, ok := webhook.Data["visibility"]; ok {
			payload["visibility"] = visibility
		}
		if occurredAt, ok := webhook.Data["occurred_at"]; ok {
			payload["occurred_at"] = occurredAt
		}
		if createdAt, ok := webhook.Data["created_at"]; ok {
			payload["created_at"] = createdAt
		}

		// Extract user display name from nested user object
		if user, ok := webhook.Data["user"].(map[string]any); ok {
			if displayName, ok := user["full_name"].(string); ok {
				payload["user_display_name"] = displayName
			}
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
