package dash0

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const OnAlertEventPayloadType = "dash0.alert.event"

var dash0AlertEventTypeOptions = []configuration.FieldOption{
	{Label: "Fired", Value: "fired"},
	{Label: "Resolved", Value: "resolved"},
}

// OnAlertEvent receives Dash0 webhook events and emits normalized trigger payloads.
type OnAlertEvent struct{}

// Name returns the stable trigger identifier.
func (t *OnAlertEvent) Name() string {
	return "dash0.onAlertEvent"
}

// Label returns the display name used in the workflow builder.
func (t *OnAlertEvent) Label() string {
	return "On Alert Event"
}

// Description returns a short summary of trigger behavior.
func (t *OnAlertEvent) Description() string {
	return "Listen to Dash0 alert webhook events when checks fire or resolve"
}

// Documentation returns markdown help shown in the trigger docs panel.
func (t *OnAlertEvent) Documentation() string {
	return `The On Alert Event trigger starts a workflow when Dash0 sends an alert webhook event.

## Use Cases

- **Incident response**: Trigger workflows when checks fire or resolve
- **Notification routing**: Send fired/resolved events to chat tools and ticketing systems
- **Automation**: Branch downstream execution based on event type and severity

## Configuration

- **Event Types**: Choose which event types to emit (fired, resolved)

## Webhook Setup

After configuring this trigger, use the generated SuperPlane webhook URL as a notification channel in Dash0 alerting.

## Event Data

Each emitted event includes:
- **checkId**
- **checkName**
- **severity**
- **labels**
- **summary**
- **description**
- **eventType** (fired or resolved)
- **timestamp**
- **event** (raw webhook payload)`
}

// Icon returns the Lucide icon name for this trigger.
func (t *OnAlertEvent) Icon() string {
	return "bell-ring"
}

// Color returns the node color used in the UI.
func (t *OnAlertEvent) Color() string {
	return "gray"
}

// Configuration defines trigger settings for event-type filtering.
func (t *OnAlertEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "eventTypes",
			Label:    "Event Types",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"fired", "resolved"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: dash0AlertEventTypeOptions,
				},
			},
			Description: "Select which Dash0 alert event types should trigger the workflow",
		},
	}
}

// Setup validates configuration and requests a webhook endpoint from SuperPlane.
func (t *OnAlertEvent) Setup(ctx core.TriggerContext) error {
	scope := "dash0.onAlertEvent setup"
	config := OnAlertEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: decode configuration: %w", scope, err)
	}

	if len(config.EventTypes) == 0 {
		return fmt.Errorf("%s: at least one event type must be selected", scope)
	}

	return ctx.Integration.RequestWebhook(map[string]any{
		"eventTypes": config.EventTypes,
	})
}

// Actions returns no manual actions for this trigger.
func (t *OnAlertEvent) Actions() []core.Action {
	return []core.Action{}
}

// HandleAction is unused because this trigger has no actions.
func (t *OnAlertEvent) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

// HandleWebhook validates and normalizes incoming Dash0 webhook payloads.
func (t *OnAlertEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	scope := "dash0.onAlertEvent webhook"
	config := OnAlertEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("%s: decode configuration: %w", scope, err)
	}

	if len(ctx.Body) == 0 {
		return http.StatusBadRequest, fmt.Errorf("%s: empty request body", scope)
	}

	payload := AlertWebhookPayload{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("%s: parse request body: %w", scope, err)
	}

	eventPayload := normalizeAlertEventPayload(payload.Data)
	if eventPayload.EventType == "" {
		return http.StatusOK, nil
	}

	if len(config.EventTypes) > 0 && !slices.Contains(config.EventTypes, eventPayload.EventType) {
		return http.StatusOK, nil
	}

	if strings.TrimSpace(eventPayload.CheckID) == "" {
		return http.StatusBadRequest, fmt.Errorf("%s: check id is required", scope)
	}

	if err := ctx.Events.Emit(OnAlertEventPayloadType, eventPayload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("%s: emit event: %w", scope, err)
	}

	return http.StatusOK, nil
}

// Cleanup is a no-op because no external resources are provisioned.
func (t *OnAlertEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// normalizeAlertEventPayload extracts and normalizes canonical event fields.
func normalizeAlertEventPayload(event map[string]any) AlertEventPayload {
	eventType := normalizeAlertEventType(findNestedString(event,
		"eventType",
		"event_type",
		"event.type",
		"type",
		"status",
		"state",
		"alert.state",
		"alert.status",
		"data.eventType",
		"data.event_type",
		"data.state",
	))

	checkID := findNestedString(event,
		"check.id",
		"checkId",
		"check_id",
		"check.ruleId",
		"check.rule_id",
		"checkRuleId",
		"check_rule_id",
		"alert.checkId",
		"alert.check_id",
		"data.check.id",
		"data.checkId",
		"id",
	)

	checkName := findNestedString(event,
		"check.name",
		"checkName",
		"check_name",
		"check.ruleName",
		"check_rule_name",
		"ruleName",
		"title",
	)

	severity := strings.ToLower(findNestedString(event,
		"check.severity",
		"severity",
		"alert.severity",
		"labels.severity",
	))

	summary := findNestedString(event,
		"check.summary",
		"summary",
		"title",
	)

	description := findNestedString(event,
		"check.description",
		"description",
		"message",
	)

	labels := findNestedMap(event,
		"check.labels",
		"labels",
		"alert.labels",
		"data.labels",
	)
	if labels == nil {
		labels = map[string]any{}
	}

	timestamp := extractAlertTimestamp(event)

	return AlertEventPayload{
		EventType:   eventType,
		CheckID:     strings.TrimSpace(checkID),
		CheckName:   strings.TrimSpace(checkName),
		Severity:    strings.TrimSpace(severity),
		Labels:      labels,
		Summary:     strings.TrimSpace(summary),
		Description: strings.TrimSpace(description),
		Timestamp:   timestamp,
		Event:       event,
	}
}

// normalizeAlertEventType maps provider-specific statuses into fired/resolved values.
func normalizeAlertEventType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "fired", "fire", "triggered", "trigger", "open", "opened", "active", "alert", "failing", "failed":
		return "fired"
	case "resolved", "resolve", "recovered", "recovery", "ok", "closed", "clear", "cleared", "normal":
		return "resolved"
	default:
		return ""
	}
}

// findNestedString reads the first non-empty string-like value from candidate paths.
func findNestedString(data map[string]any, paths ...string) string {
	for _, path := range paths {
		value, ok := findNestedValue(data, path)
		if !ok {
			continue
		}

		switch typed := value.(type) {
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed != "" {
				return trimmed
			}
		case json.Number:
			return typed.String()
		case float64:
			if typed == float64(int64(typed)) {
				return strconv.FormatInt(int64(typed), 10)
			}
			return strconv.FormatFloat(typed, 'f', -1, 64)
		case int:
			return strconv.Itoa(typed)
		case int64:
			return strconv.FormatInt(typed, 10)
		}
	}

	return ""
}

// findNestedMap reads the first nested map value from candidate paths.
func findNestedMap(data map[string]any, paths ...string) map[string]any {
	for _, path := range paths {
		value, ok := findNestedValue(data, path)
		if !ok {
			continue
		}

		mapValue, ok := value.(map[string]any)
		if !ok {
			continue
		}

		return mapValue
	}

	return nil
}

// findNestedValue resolves a dot-separated path within a nested map payload.
func findNestedValue(data map[string]any, path string) (any, bool) {
	current := any(data)
	parts := strings.Split(path, ".")

	for _, part := range parts {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}

		value, exists := currentMap[part]
		if !exists {
			return nil, false
		}

		current = value
	}

	return current, true
}

// extractAlertTimestamp finds and normalizes timestamp values from alert payloads.
func extractAlertTimestamp(data map[string]any) string {
	paths := []string{
		"timestamp",
		"time",
		"event.timestamp",
		"eventTime",
		"alert.timestamp",
		"firedAt",
		"resolvedAt",
		"createdAt",
		"updatedAt",
	}

	for _, path := range paths {
		value, ok := findNestedValue(data, path)
		if !ok {
			continue
		}

		if parsed := normalizeTimestampValue(value); parsed != "" {
			return parsed
		}
	}

	return time.Now().UTC().Format(time.RFC3339Nano)
}

// normalizeTimestampValue converts supported timestamp inputs into RFC3339Nano.
func normalizeTimestampValue(value any) string {
	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return ""
		}

		layouts := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02 15:04:05",
		}

		for _, layout := range layouts {
			parsed, err := time.Parse(layout, trimmed)
			if err == nil {
				return parsed.UTC().Format(time.RFC3339Nano)
			}
		}

		if unixInt, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			return normalizeTimestampUnix(unixInt)
		}

		return trimmed
	case float64:
		return normalizeTimestampUnix(int64(typed))
	case int:
		return normalizeTimestampUnix(int64(typed))
	case int64:
		return normalizeTimestampUnix(typed)
	case json.Number:
		if unixInt, err := typed.Int64(); err == nil {
			return normalizeTimestampUnix(unixInt)
		}
	}

	return ""
}

// normalizeTimestampUnix infers unix unit precision and formats to RFC3339Nano.
func normalizeTimestampUnix(value int64) string {
	switch {
	case value >= 1_000_000_000_000_000_000:
		return time.Unix(0, value).UTC().Format(time.RFC3339Nano)
	case value >= 1_000_000_000_000:
		return time.UnixMilli(value).UTC().Format(time.RFC3339Nano)
	default:
		return time.Unix(value, 0).UTC().Format(time.RFC3339Nano)
	}
}
