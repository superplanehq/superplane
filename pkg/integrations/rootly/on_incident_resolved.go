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

type OnIncidentResolved struct{}

type OnIncidentResolvedConfiguration struct {
	SeverityFilter []string `json:"severityFilter"`
	ServiceFilter  []string `json:"serviceFilter"`
	TeamFilter     []string `json:"teamFilter"`
}

func (t *OnIncidentResolved) Name() string {
	return "rootly.onIncidentResolved"
}

func (t *OnIncidentResolved) Label() string {
	return "On Incident Resolved"
}

func (t *OnIncidentResolved) Description() string {
	return "Listen to incident resolved events"
}

func (t *OnIncidentResolved) Documentation() string {
	return `The On Incident Resolved trigger starts a workflow execution when a Rootly incident is resolved.

## Use Cases

- **Close tickets**: Close linked Jira/ServiceNow tickets when the Rootly incident is resolved
- **Status updates**: Post to a status page or Slack when resolution is recorded
- **Post-incident steps**: Trigger retros, follow-ups, or action item syncs

## Configuration

- **Severity Filter** (optional): Only trigger for incidents with specific severity
- **Service Filter** (optional): Only trigger for incidents attached to specific Rootly services
- **Team Filter** (optional): Only trigger for incidents attached to specific Rootly teams

## Event Data

The emitted payload includes the resolved incident data. At minimum, it includes:
- **event**: ` + "`incident.resolved`" + `
- **incident**: A map containing fields like id, sequential_id, title, slug, status, resolution_message, resolved_at, resolved_by, url

## Webhook Setup

This trigger automatically sets up a Rootly webhook endpoint when configured. The endpoint is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnIncidentResolved) Icon() string {
	return "alert-triangle"
}

func (t *OnIncidentResolved) Color() string {
	return "gray"
}

func (t *OnIncidentResolved) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "severityFilter",
			Label:       "Severity Filter",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Only trigger for incidents with these severities",
			Placeholder: "Select severities (optional)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "severity",
					Multi: true,
				},
			},
		},
		{
			Name:        "serviceFilter",
			Label:       "Service Filter",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Only trigger for incidents attached to these services",
			Placeholder: "Select services (optional)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "service",
					Multi: true,
				},
			},
		},
		{
			Name:        "teamFilter",
			Label:       "Team Filter",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Only trigger for incidents attached to these teams",
			Placeholder: "Select teams (optional)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "team",
					Multi: true,
				},
			},
		},
	}
}

func (t *OnIncidentResolved) Setup(ctx core.TriggerContext) error {
	// Only request incident.resolved events; filters are applied in the webhook handler.
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Events: []string{"incident.resolved"},
	})
}

func (t *OnIncidentResolved) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIncidentResolved) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIncidentResolved) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnIncidentResolvedConfiguration{}
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

	// Only process incident.resolved events
	eventType := webhook.Event.Type
	if eventType != "incident.resolved" {
		return http.StatusOK, nil
	}

	incident := webhook.Data
	if incident == nil {
		return http.StatusOK, nil
	}

	if len(config.SeverityFilter) > 0 && !incidentMatchesSeverityFilter(incident, config.SeverityFilter) {
		return http.StatusOK, nil
	}

	if len(config.ServiceFilter) > 0 && !incidentMatchesServicesFilter(incident, config.ServiceFilter) {
		return http.StatusOK, nil
	}

	if len(config.TeamFilter) > 0 && !incidentMatchesTeamsFilter(incident, config.TeamFilter) {
		return http.StatusOK, nil
	}

	err = ctx.Events.Emit(
		"rootly.incident.resolved",
		buildIncidentResolvedPayload(webhook),
	)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func buildIncidentResolvedPayload(webhook WebhookPayload) map[string]any {
	incident := buildResolvedIncidentData(webhook.Data)

	payload := map[string]any{
		"event":     webhook.Event.Type,
		"event_id":  webhook.Event.ID,
		"issued_at": webhook.Event.IssuedAt,
		"incident":  incident,
	}

	return payload
}

func buildResolvedIncidentData(data map[string]any) map[string]any {
	incident := map[string]any{}
	if data != nil {
		// Preserve raw incident fields for flexibility.
		for k, v := range data {
			incident[k] = v
		}
	}

	// Convenience fields expected by workflows.
	setIfNotNil(incident, "id", firstNonNil(
		getValueAtPath(data, "id"),
		getValueAtPath(data, "data", "id"),
		getValueAtPath(data, "incident", "id"),
	))
	setIfNotNil(incident, "sequential_id", firstNonNil(
		getValueAtPath(data, "sequential_id"),
		getValueAtPath(data, "attributes", "sequential_id"),
		getValueAtPath(data, "data", "attributes", "sequential_id"),
	))
	setIfNotNil(incident, "title", firstNonNil(
		getValueAtPath(data, "title"),
		getValueAtPath(data, "attributes", "title"),
		getValueAtPath(data, "data", "attributes", "title"),
	))
	setIfNotNil(incident, "slug", firstNonNil(
		getValueAtPath(data, "slug"),
		getValueAtPath(data, "attributes", "slug"),
		getValueAtPath(data, "data", "attributes", "slug"),
	))
	setIfNotNil(incident, "status", firstNonNil(
		getValueAtPath(data, "status"),
		getValueAtPath(data, "attributes", "status"),
		getValueAtPath(data, "data", "attributes", "status"),
	))
	setIfNotNil(incident, "resolution_message", firstNonNil(
		getValueAtPath(data, "resolution_message"),
		getValueAtPath(data, "attributes", "resolution_message"),
		getValueAtPath(data, "data", "attributes", "resolution_message"),
	))
	setIfNotNil(incident, "resolved_at", firstNonNil(
		getValueAtPath(data, "resolved_at"),
		getValueAtPath(data, "attributes", "resolved_at"),
		getValueAtPath(data, "data", "attributes", "resolved_at"),
	))
	setIfNotNil(incident, "resolved_by", firstNonNil(
		getValueAtPath(data, "resolved_by"),
		getValueAtPath(data, "attributes", "resolved_by"),
		getValueAtPath(data, "data", "attributes", "resolved_by"),
	))
	setIfNotNil(incident, "url", firstNonNil(
		getValueAtPath(data, "url"),
		getValueAtPath(data, "attributes", "url"),
		getValueAtPath(data, "data", "attributes", "url"),
	))

	return incident
}

func incidentMatchesSeverityFilter(incident map[string]any, allowed []string) bool {
	values := collectStringsAtPaths(incident,
		[]string{"severity"},
		[]string{"severity", "id"},
		[]string{"severity", "slug"},
		[]string{"severity", "name"},
		[]string{"attributes", "severity"},
		[]string{"attributes", "severity", "id"},
		[]string{"attributes", "severity", "slug"},
		[]string{"attributes", "severity", "name"},
		[]string{"data", "attributes", "severity"},
		[]string{"data", "attributes", "severity", "id"},
		[]string{"data", "attributes", "severity", "slug"},
		[]string{"data", "attributes", "severity", "name"},
	)
	return anyOverlap(values, allowed)
}

func incidentMatchesServicesFilter(incident map[string]any, allowed []string) bool {
	values := collectIDsFromCollections(incident,
		[]string{"services"},
		[]string{"service"},
		[]string{"attributes", "services"},
		[]string{"data", "attributes", "services"},
	)
	return anyOverlap(values, allowed)
}

func incidentMatchesTeamsFilter(incident map[string]any, allowed []string) bool {
	values := collectIDsFromCollections(incident,
		[]string{"teams"},
		[]string{"team"},
		[]string{"attributes", "teams"},
		[]string{"data", "attributes", "teams"},
	)
	return anyOverlap(values, allowed)
}

func anyOverlap(values []string, allowed []string) bool {
	if len(values) == 0 {
		return false
	}
	for _, v := range values {
		if slices.Contains(allowed, v) {
			return true
		}
	}
	return false
}

func collectStringsAtPaths(root map[string]any, paths ...[]string) []string {
	out := []string{}
	for _, p := range paths {
		v := getValueAtPath(root, p...)
		switch vv := v.(type) {
		case string:
			if vv != "" {
				out = append(out, vv)
			}
		case map[string]any:
			// Allow "id"/"slug"/"name" as fallbacks if caller passed the object itself.
			for _, key := range []string{"id", "slug", "name"} {
				if s, ok := vv[key].(string); ok && s != "" {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

func collectIDsFromCollections(root map[string]any, collectionPaths ...[]string) []string {
	out := []string{}
	for _, p := range collectionPaths {
		v := getValueAtPath(root, p...)
		out = append(out, extractIDs(v)...)
	}
	return out
}

func extractIDs(v any) []string {
	out := []string{}
	switch vv := v.(type) {
	case string:
		if vv != "" {
			out = append(out, vv)
		}
	case []any:
		for _, item := range vv {
			out = append(out, extractIDs(item)...)
		}
	case map[string]any:
		// Common shapes:
		// - {id: "..."}
		// - {data: [{id: "..."}]}
		// - {data: {id: "..."}}
		if s, ok := vv["id"].(string); ok && s != "" {
			out = append(out, s)
		}
		if s, ok := vv["slug"].(string); ok && s != "" {
			out = append(out, s)
		}
		if data, ok := vv["data"]; ok {
			out = append(out, extractIDs(data)...)
		}
	}
	return out
}

func getValueAtPath(root any, path ...string) any {
	if len(path) == 0 {
		return root
	}

	m, ok := root.(map[string]any)
	if !ok || m == nil {
		return nil
	}

	next, ok := m[path[0]]
	if !ok {
		return nil
	}

	return getValueAtPath(next, path[1:]...)
}

func firstNonNil(values ...any) any {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func setIfNotNil(m map[string]any, key string, value any) {
	if value == nil {
		return
	}
	m[key] = value
}
