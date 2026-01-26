package datadog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type OnMonitorAlert struct{}

type OnMonitorAlertConfiguration struct {
	AlertTransitions []string `json:"alertTransitions"`
	Tags             string   `json:"tags"`
}

func (t *OnMonitorAlert) Name() string {
	return "datadog.onMonitorAlert"
}

func (t *OnMonitorAlert) Label() string {
	return "On Monitor Alert"
}

func (t *OnMonitorAlert) Description() string {
	return "Listen to Datadog monitor alert webhooks"
}

func (t *OnMonitorAlert) Icon() string {
	return "chart-bar"
}

func (t *OnMonitorAlert) Color() string {
	return "gray"
}

func (t *OnMonitorAlert) Documentation() string {
	return `The On Monitor Alert trigger starts a workflow execution when Datadog monitor alerts fire.

## Use Cases

- **Automated incident response**: Trigger runbooks when alerts fire
- **Alert routing**: Route alerts to different workflows based on tags or severity
- **Metric-based automation**: Execute actions when thresholds are breached

## Webhook Configuration

Configure a webhook in Datadog to send alerts to the SuperPlane webhook URL. Include the X-Superplane-Signature-256 header with an HMAC-SHA256 signature.

## Event Payload

The trigger emits an event containing:
- ` + "`id`" + `: The alert event ID
- ` + "`event_type`" + `: Type of event
- ` + "`alert_type`" + `: Alert severity
- ` + "`alert_transition`" + `: State change (Triggered, Recovered, etc.)
- ` + "`monitor_id`" + `: The monitor's numeric ID
- ` + "`monitor_name`" + `: Human-readable monitor name
- ` + "`title`" + `: Alert title
- ` + "`body`" + `: Alert message body
- ` + "`date`" + `: Unix timestamp of the alert
- ` + "`tags`" + `: Array of tags from the monitor
`
}

func (t *OnMonitorAlert) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "alertTransitions",
			Label:    "Alert Transitions",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"Triggered"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Triggered", Value: "Triggered"},
						{Label: "Recovered", Value: "Recovered"},
						{Label: "Re-Triggered", Value: "Re-Triggered"},
						{Label: "No Data", Value: "No Data"},
						{Label: "Warn", Value: "Warn"},
						{Label: "Recovered from Warn", Value: "Recovered from Warn"},
					},
				},
			},
		},
		{
			Name:        "tags",
			Label:       "Tags Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Comma-separated list of tags to filter alerts (e.g., env:prod,service:web)",
			Placeholder: "env:prod,service:web",
		},
	}
}

func (t *OnMonitorAlert) Setup(ctx core.TriggerContext) error {
	config := OnMonitorAlertConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if len(config.AlertTransitions) == 0 {
		return fmt.Errorf("at least one alert transition must be chosen")
	}

	return nil
}

func (t *OnMonitorAlert) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnMonitorAlert) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnMonitorAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	code, err := verifyWebhookSecret(ctx)
	if err != nil {
		return code, err
	}

	config := OnMonitorAlertConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	var payload MonitorAlertPayload
	err = json.Unmarshal(ctx.Body, &payload)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// Filter by alert transition
	if !slices.Contains(config.AlertTransitions, payload.AlertTransition) {
		return http.StatusOK, nil
	}

	// Filter by tags if configured
	if config.Tags != "" && !matchesTags(payload.Tags, config.Tags) {
		return http.StatusOK, nil
	}

	err = ctx.Events.Emit(
		"datadog.monitor.alert",
		buildMonitorAlertPayload(payload),
	)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

// MonitorAlertPayload represents the webhook payload sent by Datadog when a monitor alert fires.
type MonitorAlertPayload struct {
	ID              string   `json:"id"`
	EventType       string   `json:"event_type"`
	AlertType       string   `json:"alert_type"`
	AlertTransition string   `json:"alert_transition"`
	Hostname        string   `json:"hostname"`
	MonitorID       int64    `json:"monitor_id"`
	MonitorName     string   `json:"monitor_name"`
	Priority        string   `json:"priority"`
	Tags            []string `json:"tags"`
	Title           string   `json:"title"`
	Date            int64    `json:"date"`
	Body            string   `json:"body"`
	Org             *OrgInfo `json:"org"`
}

type OrgInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func buildMonitorAlertPayload(payload MonitorAlertPayload) map[string]any {
	result := map[string]any{
		"id":               payload.ID,
		"event_type":       payload.EventType,
		"alert_type":       payload.AlertType,
		"alert_transition": payload.AlertTransition,
		"monitor_id":       payload.MonitorID,
		"monitor_name":     payload.MonitorName,
		"title":            payload.Title,
		"body":             payload.Body,
		"date":             payload.Date,
		"tags":             payload.Tags,
	}

	if payload.Hostname != "" {
		result["hostname"] = payload.Hostname
	}

	if payload.Priority != "" {
		result["priority"] = payload.Priority
	}

	if payload.Org != nil {
		result["org"] = map[string]any{
			"id":   payload.Org.ID,
			"name": payload.Org.Name,
		}
	}

	return result
}

func verifyWebhookSecret(ctx core.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get("X-Superplane-Signature-256")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing X-Superplane-Signature-256 header")
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature format")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting webhook secret: %v", err)
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	return http.StatusOK, nil
}

func matchesTags(payloadTags []string, filterTags string) bool {
	if len(payloadTags) == 0 {
		return false
	}

	for _, filterTag := range parseTags(filterTags) {
		if slices.Contains(payloadTags, filterTag) {
			return true
		}
	}

	return false
}

func parseTags(tags string) []string {
	var result []string
	for _, tag := range strings.Split(tags, ",") {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
