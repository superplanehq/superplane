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

const rootlyIncidentTimelineEventPayloadType = "rootly.onIncidentTimelineEvent"

var rootlyIncidentTimelineEventWebhookTypes = []string{
	"incident_event.created",
	"incident_event.updated",
}

type OnIncidentTimelineEvent struct{}

type OnIncidentTimelineEventConfiguration struct {
	IncidentStatus []string `json:"incidentStatus" mapstructure:"incidentStatus"`
	Severity       []string `json:"severity" mapstructure:"severity"`
	Service        []string `json:"service" mapstructure:"service"`
	Team           []string `json:"team" mapstructure:"team"`
	EventSource    []string `json:"eventSource" mapstructure:"eventSource"`
	Visibility     string   `json:"visibility" mapstructure:"visibility"`
}

type OnIncidentTimelineEventMetadata struct {
	EventStates map[string]string `json:"eventStates"`
}

func (t *OnIncidentTimelineEvent) Name() string {
	return "rootly.onIncidentTimelineEvent"
}

func (t *OnIncidentTimelineEvent) Label() string {
	return "On Incident Timeline Event"
}

func (t *OnIncidentTimelineEvent) Description() string {
	return "Listen to incident timeline events"
}

func (t *OnIncidentTimelineEvent) Documentation() string {
	return `The On Incident Timeline Event trigger starts a workflow execution when Rootly incident timeline events are created or updated.
Only events with kind "event" are emitted.

## Use Cases

- **Note automation**: Run workflows when investigation notes are added
- **Timeline sync**: Sync incident timeline events to Slack or external systems
- **Annotation tracking**: Track updates to incident annotations
- **Audit logging**: Capture timeline events for compliance or reporting

## Configuration

- **Incident Status**: Optional filter by incident status (open, resolved, etc.)
- **Severity**: Optional filter by incident severity
- **Service**: Optional filter by service name
- **Team**: Optional filter by team name
- **Event Source**: Optional filter by event source (web, api, system)
- **Visibility**: Optional filter by event visibility (internal or external)

## Event Data

Each incident event includes:
- **id**: Event ID
- **event**: Event content
- **event_raw**: Raw event content
- **event_id**: Webhook event ID
- **event_type**: Event type (incident_event.created or incident_event.updated)
- **kind**: Event kind
- **source**: Event source
- **visibility**: Event visibility
- **occurred_at**: When the event occurred
- **created_at**: When the event was created
- **updated_at**: When the event was last updated
- **issued_at**: When the webhook was issued
- **incident_id**: Incident ID
- **incident**: Incident information

## Webhook Setup

This trigger automatically sets up a Rootly webhook endpoint when configured. The endpoint is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnIncidentTimelineEvent) Icon() string {
	return "message-square"
}

func (t *OnIncidentTimelineEvent) Color() string {
	return "gray"
}

func (t *OnIncidentTimelineEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentStatus",
			Label:       "Incident Status",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Togglable:   true,
			Description: "Only emit events for incidents with this status",
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
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Only emit events for incidents with this severity",
			Placeholder: "Select a severity",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "severity",
					UseNameAsValue: true,
					Multi:          true,
				},
			},
		},
		{
			Name:        "service",
			Label:       "Service",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Only emit events for incidents impacting this service",
			Placeholder: "Select a service",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "service",
					UseNameAsValue: true,
					Multi:          true,
				},
			},
		},
		{
			Name:        "team",
			Label:       "Team",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Only emit events for incidents owned by this team",
			Placeholder: "Select a team",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "team",
					UseNameAsValue: true,
					Multi:          true,
				},
			},
		},
		{
			Name:        "eventSource",
			Label:       "Event Source",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Togglable:   true,
			Description: "Only emit events from these sources",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Web", Value: "web"},
						{Label: "API", Value: "api"},
						{Label: "System", Value: "system"},
					},
				},
			},
		},
		{
			Name:      "visibility",
			Label:     "Visibility",
			Type:      configuration.FieldTypeSelect,
			Required:  false,
			Togglable: true,
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

func (t *OnIncidentTimelineEvent) Setup(ctx core.TriggerContext) error {
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Events: rootlyIncidentTimelineEventWebhookTypes,
	})
}

func (t *OnIncidentTimelineEvent) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIncidentTimelineEvent) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIncidentTimelineEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnIncidentTimelineEventConfiguration{}
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

	if !slices.Contains(rootlyIncidentTimelineEventWebhookTypes, webhook.Event.Type) {
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

	// Apply event-level filters directly from the webhook payload.
	if !matchesEventFilter(config.EventSource, extractString(incidentEvent, "source")) {
		return http.StatusOK, nil
	}

	if config.Visibility != "" && !strings.EqualFold(config.Visibility, extractString(incidentEvent, "visibility")) {
		return http.StatusOK, nil
	}

	incidentID := extractString(incidentEvent, "incident_id", "incidentId")

	// Only process events with kind "event" to avoid emitting for non-timeline events like "trail/start/close".
	if !strings.EqualFold(extractString(incidentEvent, "kind"), "event") {
		return http.StatusOK, nil
	}

	incidentFiltersEnabled := len(config.IncidentStatus) > 0 || len(config.Severity) > 0 || len(config.Service) > 0 || len(config.Team) > 0
	var incidentDetails map[string]any
	if incidentID != "" && ctx.HTTP != nil && ctx.Integration != nil {
		// Fetch incident details to apply status/severity/service/team filters
		// and enrich the emitted payload with incident context.
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error creating client: %v", err)
		}

		incidentDetails, err = client.GetIncidentDetailed(incidentID)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error fetching incident: %v", err)
		}
	}

	if incidentFiltersEnabled {
		if incidentID == "" {
			return http.StatusOK, nil
		}

		if incidentDetails == nil {
			return http.StatusInternalServerError, fmt.Errorf("incident details unavailable for filtering")
		}

		if !matchesIncidentFilters(incidentDetails, config) {
			return http.StatusOK, nil
		}
	}

	metadata := loadOnIncidentTimelineEventMetadata(ctx.Metadata)
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

	if eventID != "" {
		if previous, exists := metadata.EventStates[eventID]; exists && previous == fingerprint {
			return http.StatusOK, nil
		}
	}

	payload := buildIncidentTimelineEventPayload(incidentDetails, incidentEvent, webhook.Event)
	if err := ctx.Events.Emit(rootlyIncidentTimelineEventPayloadType, payload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}
	emitted++

	if ctx.Metadata != nil {
		if err := ctx.Metadata.Set(OnIncidentTimelineEventMetadata{EventStates: updatedStates}); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error updating metadata: %v", err)
		}
	}

	if emitted == 0 {
		return http.StatusOK, nil
	}

	return http.StatusOK, nil
}

func (t *OnIncidentTimelineEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// loadOnIncidentTimelineEventMetadata pulls persisted state for deduping repeated webhook deliveries.
func loadOnIncidentTimelineEventMetadata(ctx core.MetadataContext) OnIncidentTimelineEventMetadata {
	if ctx == nil {
		return OnIncidentTimelineEventMetadata{EventStates: map[string]string{}}
	}

	metadata := OnIncidentTimelineEventMetadata{}
	if err := mapstructure.Decode(ctx.Get(), &metadata); err != nil || metadata.EventStates == nil {
		return OnIncidentTimelineEventMetadata{EventStates: map[string]string{}}
	}

	return metadata
}

// extractEventFromData normalizes either top-level or nested incident_event payloads.
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

// buildIncidentTimelineEventPayload keeps the Rootly event shape and augments with incident context.
func buildIncidentTimelineEventPayload(incident map[string]any, incidentEvent map[string]any, webhookEvent WebhookEvent) map[string]any {
	incidentPayload := buildIncidentSummaryPayload(incident, incidentEvent)
	payload := map[string]any{
		"id":          extractString(incidentEvent, "id"),
		"event":       extractString(incidentEvent, "event"),
		"event_raw":   extractString(incidentEvent, "event_raw"),
		"kind":        extractString(incidentEvent, "kind"),
		"source":      extractString(incidentEvent, "source"),
		"visibility":  extractString(incidentEvent, "visibility"),
		"occurred_at": extractString(incidentEvent, "occurred_at"),
		"created_at":  extractString(incidentEvent, "created_at"),
		"updated_at":  extractString(incidentEvent, "updated_at"),
		"incident_id": extractString(incidentEvent, "incident_id", "incidentId"),
		"event_id":    webhookEvent.ID,
		"event_type":  webhookEvent.Type,
		"issued_at":   webhookEvent.IssuedAt,
		"incident":    incidentPayload,
	}

	return payload
}

// buildIncidentSummaryPayload emits a compact incident stub for filtering context.
func buildIncidentSummaryPayload(incident map[string]any, incidentEvent map[string]any) map[string]any {
	incidentID := ""
	if incident != nil {
		incidentID = extractString(incident, "id")
	}
	if incidentID == "" {
		incidentID = extractString(incidentEvent, "incident_id", "incidentId")
	}

	if incidentID == "" && incident == nil {
		return nil
	}

	payload := map[string]any{
		"id": incidentID,
	}

	if incident == nil {
		return payload
	}

	if title := extractString(incident, "title"); title != "" {
		payload["title"] = title
	}
	if status := extractString(incident, "status", "state"); status != "" {
		payload["status"] = status
	}
	if severity := severityString(incident["severity"]); severity != "" {
		payload["severity"] = severity
	}
	if services := extractResourceNames(incident, "services"); len(services) > 0 {
		payload["services"] = services
	}
	if teams := extractResourceNames(incident, "groups"); len(teams) > 0 {
		payload["teams"] = teams
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

func matchesEventFilter(filters []string, value string) bool {
	if len(filters) == 0 {
		return true
	}

	return slices.ContainsFunc(filters, func(filter string) bool {
		return strings.EqualFold(filter, value)
	})
}

// eventFingerprint helps avoid double-processing when Rootly retries a webhook.
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

// isIncidentEventPayload checks for expected incident event fields.
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

// matchesIncidentFilters applies incident-level filters from fetched incident details.
func matchesIncidentFilters(incident map[string]any, config OnIncidentTimelineEventConfiguration) bool {
	if len(config.IncidentStatus) > 0 {
		status := extractString(incident, "status", "state")
		if !matchesEventFilter(config.IncidentStatus, status) {
			return false
		}
	}

	if len(config.Severity) > 0 {
		severity := severityString(incident["severity"])
		if severity == "" {
			severity = extractString(incident, "severity")
		}
		if !matchesEventFilter(config.Severity, severity) {
			return false
		}
	}

	if len(config.Service) > 0 {
		services := extractResourceNames(incident, "services")
		if !matchesAnyResource(services, config.Service) {
			return false
		}
	}

	if len(config.Team) > 0 {
		teams := extractResourceNames(incident, "groups")
		if !matchesAnyResource(teams, config.Team) {
			return false
		}
	}

	return true
}

// extractResourceNames pulls service/team names from varied response shapes.
func extractResourceNames(incident map[string]any, key string) []string {
	raw, ok := incident[key]
	if !ok || raw == nil {
		return nil
	}

	switch items := raw.(type) {
	case []any:
		names := make([]string, 0, len(items))
		for _, item := range items {
			resource, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if name := extractString(resource, "name", "slug"); name != "" {
				names = append(names, name)
			}
		}
		return names
	case []map[string]any:
		names := make([]string, 0, len(items))
		for _, resource := range items {
			if name := extractString(resource, "name", "slug"); name != "" {
				names = append(names, name)
			}
		}
		return names
	case map[string]any:
		if name := extractString(items, "name", "slug"); name != "" {
			return []string{name}
		}
	}

	return nil
}

func matchesAnyResource(resources []string, filters []string) bool {
	return slices.ContainsFunc(filters, func(filter string) bool {
		return slices.ContainsFunc(resources, func(resource string) bool {
			return strings.EqualFold(resource, filter)
		})
	})
}
