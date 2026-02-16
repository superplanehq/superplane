package rootly

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
	
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnEvent struct{}

type OnEventConfiguration struct {
	IncidentStatuses []string `json:"incidentStatuses"`
	Severities       []string `json:"severities"`
	Services         []string `json:"services"`
	Teams            []string `json:"teams"`
	Visibility       string   `json:"visibility"`
}

var onEventWebhookEvents = []string{"incident.created", "incident.updated"}

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

- **Incident notes**: Notify Slack or other systems when a note is added
- **Timeline sync**: Mirror timeline events into external systems
- **Investigation automation**: React to annotations as they are added or edited

## Configuration

- **Incident Status** (optional): Filter by incident status
- **Severity** (optional): Filter by incident severity
- **Service** (optional): Filter by service name
- **Team** (optional): Filter by team name
- **Visibility** (optional): Filter by visibility (internal/external)

## Event Data

Each event includes:
- **id**: Timeline event ID
- **event**: Event text
- **kind**: Event kind
- **occurred_at**: When the event occurred
- **created_at**: When the event was created
- **user_display_name**: Display name of the event author (if available)
- **incident**: Incident data from Rootly

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
			Name:        "incidentStatuses",
			Label:       "Incident Status",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter by incident status values (e.g. started, resolved)",
			Placeholder: "Select incident statuses",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "In Triage", Value: "in_triage"},
						{Label: "Started", Value: "started"},
						{Label: "Detected", Value: "detected"},

						{Label: "Acknowledged", Value: "acknowledged"},
						{Label: "Mitigated", Value: "mitigated"},
						{Label: "Resolved", Value: "resolved"},
						{Label: "Closed", Value: "closed"},
						{Label: "Cancelled", Value: "cancelled"},
					},
				},
			},
		},
		{
			Name:        "severities",
			Label:       "Severity",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Filter by incident severity",
			Placeholder: "Select severities",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "severity",
					UseNameAsValue: true,
					Multi:          true,
				},
			},
		},
		{
			Name:        "services",
			Label:       "Service",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Filter by service name",
			Placeholder: "Select services",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "service",
					UseNameAsValue: true,
					Multi:          true,
				},
			},
		},
		{
			Name:        "teams",
			Label:       "Team",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Filter by team name",
			Placeholder: "Select teams",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "team",
					UseNameAsValue: true,
					Multi:          true,
				},
			},
		},
		{
			Name:        "visibility",
			Label:       "Visibility",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Filter by visibility values (e.g. internal, external)",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
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
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Events: onEventWebhookEvents,
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
	req, code, err := decodeAndVerifyWebhook(ctx, &config)
	if err != nil {
		return code, err
	}

	webhook := req.payload
	if !slices.Contains(onEventWebhookEvents, webhook.Event.Type) {
		return http.StatusOK, nil
	}

	incident := webhook.Data
	if incident == nil {
		return http.StatusOK, nil
	}

	if !matchesIncidentFilters(incident, config) {
		return http.StatusOK, nil
	}

	events := extractEventList(incident)
	if len(events) == 0 {
		return http.StatusOK, nil
	}

	matchingEvents := filterEvents(events, config)
	if len(matchingEvents) == 0 {
		return http.StatusOK, nil
	}

	selectedEvents := selectLatestEvents(matchingEvents)
	for _, event := range selectedEvents {
		payload := buildIncidentEventPayload(incident, event)
		if err := ctx.Events.Emit("rootly.onEvent", payload); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
		}
	}

	return http.StatusOK, nil
}

func (t *OnEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func matchesIncidentFilters(incident map[string]any, config OnEventConfiguration) bool {
	if len(config.IncidentStatuses) > 0 {
		status := stringFromAny(incident["status"])
		if !containsString(config.IncidentStatuses, status) {
			return false
		}
	}

	if len(config.Severities) > 0 {
		severity := extractResourceName(incident["severity"])
		if !containsString(config.Severities, severity) {
			return false
		}
	}

	if len(config.Services) > 0 {
		services := extractResourceList(incident["services"])
		if !containsAny(config.Services, services) {
			return false
		}
	}

	if len(config.Teams) > 0 {
		teams := extractResourceList(incident["teams"])
		if len(teams) == 0 {
			teams = extractResourceList(incident["groups"])
		}
		if !containsAny(config.Teams, teams) {
			return false
		}
	}

	return true
}

func filterEvents(events []map[string]any, config OnEventConfiguration) []map[string]any {
	filtered := make([]map[string]any, 0, len(events))
	for _, event := range events {
		if config.Visibility != "" {
			visibility := stringFromAny(event["visibility"])
			if config.Visibility != visibility {
				continue
			}
		}

		filtered = append(filtered, event)
	}

	return filtered
}

func selectLatestEvents(events []map[string]any) []map[string]any {
	if len(events) == 0 {
		return nil
	}

	var latest time.Time
	for _, event := range events {
		if eventTime := eventTimestamp(event); eventTime.After(latest) {
			latest = eventTime
		}
	}

	if latest.IsZero() {
		return []map[string]any{events[len(events)-1]}
	}

	selected := make([]map[string]any, 0, len(events))
	for _, event := range events {
		if eventTimestamp(event).Equal(latest) {
			selected = append(selected, event)
		}
	}

	return selected
}

func eventTimestamp(event map[string]any) time.Time {
	if ts := parseTimestamp(stringFromAny(event["updated_at"])); !ts.IsZero() {
		return ts
	}
	if ts := parseTimestamp(stringFromAny(event["created_at"])); !ts.IsZero() {
		return ts
	}
	return parseTimestamp(stringFromAny(event["occurred_at"]))
}

func parseTimestamp(value string) time.Time {
	if value == "" {
		return time.Time{}
	}

	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}

	return parsed
}

func extractEventList(incident map[string]any) []map[string]any {
	items, ok := incident["events"].([]any)
	if !ok {
		return nil
	}

	events := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if event, ok := item.(map[string]any); ok {
			events = append(events, event)
		}
	}

	return events
}

func buildIncidentEventPayload(incident map[string]any, event map[string]any) map[string]any {
	payload := map[string]any{
		"id":                stringFromAny(event["id"]),
		"event":             stringFromAny(event["event"]),
		"kind":              stringFromAny(event["kind"]),
		"source":            stringFromAny(event["source"]),
		"visibility":        stringFromAny(event["visibility"]),
		"occurred_at":       stringFromAny(event["occurred_at"]),
		"created_at":        stringFromAny(event["created_at"]),
		"user_display_name": extractUserDisplayName(incident, event),
		"incident":          incident,
	}

	return payload
}

func extractUserDisplayName(incident map[string]any, event map[string]any) string {
	if name := stringFromAny(event["user_display_name"]); name != "" {
		return name
	}

	if user, ok := event["user"].(map[string]any); ok {
		if name := stringFromAny(user["full_name"]); name != "" {
			return name
		}
		if name := stringFromAny(user["name"]); name != "" {
			return name
		}
	}

	if user, ok := incident["user"].(map[string]any); ok {
		if name := stringFromAny(user["full_name"]); name != "" {
			return name
		}
		if name := stringFromAny(user["name"]); name != "" {
			return name
		}
	}

	return ""
}

func extractResourceName(value any) string {
	if value == nil {
		return ""
	}

	if s, ok := value.(string); ok {
		return s
	}

	resource, ok := value.(map[string]any)
	if !ok {
		return ""
	}

	if name := stringFromAny(resource["name"]); name != "" {
		return name
	}

	if slug := stringFromAny(resource["slug"]); slug != "" {
		return slug
	}

	return stringFromAny(resource["id"])
}

func extractResourceList(value any) []string {
	if value == nil {
		return nil
	}

	items, ok := value.([]any)
	if !ok {
		return nil
	}

	resources := make([]string, 0, len(items))
	for _, item := range items {
		if resource, ok := item.(map[string]any); ok {
			if name := stringFromAny(resource["name"]); name != "" {
				resources = append(resources, name)
			}
			if slug := stringFromAny(resource["slug"]); slug != "" {
				resources = append(resources, slug)
			}
			if id := stringFromAny(resource["id"]); id != "" {
				resources = append(resources, id)
			}
		}
	}

	return resources
}

func containsString(haystack []string, value string) bool {
	if value == "" {
		return false
	}

	for _, item := range haystack {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(value)) {
			return true
		}
	}

	return false
}

func containsAny(expected []string, values []string) bool {
	for _, value := range values {
		if containsString(expected, value) {
			return true
		}
	}

	return false
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}
