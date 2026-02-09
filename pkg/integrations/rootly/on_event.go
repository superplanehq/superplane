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
	Status     string   `json:"status"`
	Severity   string   `json:"severity"`
	Service    string   `json:"service"`
	Team       string   `json:"team"`
	Source     string   `json:"source"`
	Visibility string   `json:"visibility"`
	Kind       string   `json:"kind"`
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
	return `The On Event trigger starts a workflow execution when an incident timeline event is created or updated in Rootly.

## Use Cases

- **Note tracking**: Run a workflow when someone adds a note to an incident (e.g. notify channel, update Jira)
- **Timeline sync**: Sync timeline events to Slack or external systems
- **Investigation automation**: Run automation when investigation notes are added

## Configuration

- **Events**: Select which incident events to listen for (created, updated, mitigated, resolved, cancelled, deleted)
- **Incident status**: Filter by incident status (e.g. started, mitigated, resolved)
- **Severity**: Filter by incident severity
- **Service**: Filter by service name
- **Team**: Filter by team name
- **Event source**: Filter by event source (e.g. web, api, slack)
- **Visibility**: Filter by event visibility (internal or external)
- **Event kind**: Filter by event kind (e.g. event, trail)

## Event Data

Each event includes:
- **event**: Event type (incident.created, incident.updated, etc.)
- **event_id**: Unique event identifier
- **issued_at**: Timestamp of the event
- **incident**: Complete incident information including timeline events

## Webhook Setup

This trigger automatically sets up a Rootly webhook endpoint when configured. The endpoint is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnEvent) Icon() string {
	return "activity"
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
			Default:  []string{"incident.created", "incident.updated"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "incident.created"},
						{Label: "Updated", Value: "incident.updated"},
						{Label: "Mitigated", Value: "incident.mitigated"},
						{Label: "Resolved", Value: "incident.resolved"},
						{Label: "Cancelled", Value: "incident.cancelled"},
						{Label: "Deleted", Value: "incident.deleted"},
					},
				},
			},
		},
		{
			Name:        "status",
			Label:       "Incident Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Filter by incident status",
			Placeholder: "Any status",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
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
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Filter by incident severity",
			Placeholder: "Any severity",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "severity",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "service",
			Label:       "Service",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Filter by service name",
			Placeholder: "Any service",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "service",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "team",
			Label:       "Team",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by team name",
			Placeholder: "Any team",
		},
		{
			Name:        "source",
			Label:       "Event Source",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by event source (e.g. web, api, slack)",
			Placeholder: "Any source",
		},
		{
			Name:        "visibility",
			Label:       "Visibility",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Filter by event visibility",
			Placeholder: "Any visibility",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Internal", Value: "internal"},
						{Label: "External", Value: "external"},
					},
				},
			},
		},
		{
			Name:        "kind",
			Label:       "Event Kind",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by event kind (e.g. event, trail)",
			Placeholder: "Any kind",
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

	// Since the webhook may be shared and receive more events than this trigger cares about,
	// we need to filter events by their type here.
	if !slices.Contains(config.Events, eventType) {
		return http.StatusOK, nil
	}

	// Apply optional filters on the incident data
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

// matchesEventFilters checks if the incident data matches the configured filters.
// All filters are optional; if not set, they are ignored.
func matchesEventFilters(data map[string]any, config OnEventConfiguration) bool {
	if data == nil {
		// If any filters are configured, nil data should fail the filter checks
		hasFilters := config.Status != "" || config.Severity != "" || config.Service != "" ||
			config.Team != "" || config.Source != "" || config.Visibility != "" || config.Kind != ""
		return !hasFilters
	}

	if config.Status != "" {
		status, ok := data["status"].(string)
		if !ok || status != config.Status {
			return false
		}
	}

	if config.Severity != "" {
		if !matchSeverity(data, config.Severity) {
			return false
		}
	}

	if config.Service != "" {
		if !matchService(data, config.Service) {
			return false
		}
	}

	if config.Team != "" {
		if !matchTeam(data, config.Team) {
			return false
		}
	}

	// Check timeline event fields from the events array within the incident
	if config.Source != "" || config.Visibility != "" || config.Kind != "" {
		if !matchTimelineEventFields(data, config) {
			return false
		}
	}

	return true
}

// matchSeverity checks if the incident severity matches the configured filter.
// Rootly severity can be a string or an object with a "name" field.
func matchSeverity(data map[string]any, severity string) bool {
	switch v := data["severity"].(type) {
	case string:
		return v == severity
	case map[string]any:
		if name, ok := v["name"].(string); ok {
			return name == severity
		}
	}

	return false
}

// matchService checks if any of the incident's services match the configured filter.
// Rootly services can be an array of objects with a "name" field.
func matchService(data map[string]any, service string) bool {
	services, ok := data["services"].([]any)
	if !ok {
		return false
	}

	for _, s := range services {
		if svc, ok := s.(map[string]any); ok {
			if name, ok := svc["name"].(string); ok {
				if name == service {
					return true
				}
			}
		}
	}

	return false
}

// matchTeam checks if any of the incident's teams match the configured filter.
// Rootly teams can be an array of objects with a "name" field.
func matchTeam(data map[string]any, team string) bool {
	teams, ok := data["teams"].([]any)
	if !ok {
		return false
	}

	for _, t := range teams {
		if tm, ok := t.(map[string]any); ok {
			if name, ok := tm["name"].(string); ok {
				if name == team {
					return true
				}
			}
		}
	}

	return false
}

// matchTimelineEventFields checks if any timeline event within the incident data
// matches the configured source, visibility, and kind filters.
func matchTimelineEventFields(data map[string]any, config OnEventConfiguration) bool {
	events, ok := data["events"].([]any)
	if !ok {
		// If no events array, skip timeline event filtering
		return true
	}

	for _, e := range events {
		event, ok := e.(map[string]any)
		if !ok {
			continue
		}

		if config.Source != "" {
			if source, ok := event["source"].(string); ok {
				if source != config.Source {
					continue
				}
			} else {
				continue
			}
		}

		if config.Visibility != "" {
			if visibility, ok := event["visibility"].(string); ok {
				if visibility != config.Visibility {
					continue
				}
			} else {
				continue
			}
		}

		if config.Kind != "" {
			if kind, ok := event["kind"].(string); ok {
				if kind != config.Kind {
					continue
				}
			} else {
				continue
			}
		}

		// If we get here, all configured filters matched for this event
		return true
	}

	// No event matched all filters
	return false
}

func buildEventPayload(webhook WebhookPayload) map[string]any {
	payload := map[string]any{
		"event":     webhook.Event.Type,
		"event_id":  webhook.Event.ID,
		"issued_at": webhook.Event.IssuedAt,
	}

	if webhook.Data != nil {
		payload["incident"] = webhook.Data
	}

	return payload
}

func (t *OnEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}
