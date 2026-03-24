package logfire

import (
	"encoding/json"
	"fmt"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"net/http"
	"strings"
	"time"
)

type OnAlertReceived struct{}

type onAlertReceivedWebhookConfiguration struct {
	EventType string `json:"eventType"`
	Resource  string `json:"resource"`
}

const (
	onAlertReceivedEventType = "alert.received"
	onAlertReceivedResource  = "alerts"
)

func (t *OnAlertReceived) Name() string {
	return "logfire.onAlertReceived"
}

func (t *OnAlertReceived) Label() string {
	return "On Alert Received"
}

func (t *OnAlertReceived) Description() string {
	return "Trigger when a Logfire alert is received via webhook"
}

func (t *OnAlertReceived) Documentation() string {
	return `The On Alert Received trigger starts a workflow execution when Logfire sends an alert payload to your SuperPlane webhook URL.

## Configuration

This trigger does not require any additional fields.

## Webhook setup

After you save this trigger, SuperPlane provides a webhook URL. Add that URL as a Logfire notification webhook target so alert events are sent to this workflow.
`
}

func (t *OnAlertReceived) Icon() string {
	return "flame"
}

func (t *OnAlertReceived) Color() string {
	return "gray"
}

func (t *OnAlertReceived) ExampleData() map[string]any {
	return map[string]any{
		"alertId":   "alt_123",
		"alertName": "Latency spike",
		"eventType": "firing",
		"severity":  "warning",
		"message":   "p95 latency exceeded threshold",
		"url":       "https://logfire-us.pydantic.dev",
		"timestamp": "2026-03-23T12:00:00Z",
	}
}

func (t *OnAlertReceived) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *OnAlertReceived) Setup(ctx core.TriggerContext) error {
	return ctx.Integration.RequestWebhook(onAlertReceivedWebhookConfiguration{
		EventType: onAlertReceivedEventType,
		Resource:  onAlertReceivedResource,
	})
}

func (t *OnAlertReceived) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnAlertReceived) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnAlertReceived) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	payload := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("failed to parse request body: %w", err)
	}
	if payload == nil {
		payload = map[string]any{}
	}
	for key, value := range normalizeSlackAlertPayload(payload) {
		payload[key] = value
	}
	copyStringField(payload, "alert_id", "alertId")
	copyStringField(payload, "alert_name", "alertName")
	copyStringField(payload, "event_type", "eventType")
	if err := ctx.Events.Emit("logfire.alert.received", payload); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to emit alert event: %w", err)
	}
	return http.StatusOK, nil, nil
}

func copyStringField(payload map[string]any, from, to string) {
	value, ok := payload[from].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return
	}
	payload[to] = value
}

func (t *OnAlertReceived) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func normalizeSlackAlertPayload(payload map[string]any) map[string]any {
	out := map[string]any{
		"eventType": "alert",
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	}
	raw := payload
	if data, ok := payload["data"].(map[string]any); ok && data != nil {
		raw = data
	}
	text, _ := raw["text"].(string)
	if text == "" {
		return out
	}
	if start := strings.Index(text, "<"); start >= 0 {
		if endOffset := strings.Index(text[start:], ">"); endOffset > 0 {
			link := text[start+1 : start+endOffset]
			parts := strings.SplitN(link, "|", 2)
			out["url"] = strings.TrimSpace(parts[0])
			if len(parts) == 2 {
				out["alertName"] = strings.TrimSpace(parts[1])
			}
		}
	}
	if alertURL, ok := out["url"].(string); ok && alertURL != "" {
		if alertsIndex := strings.Index(alertURL, "/alerts/"); alertsIndex >= 0 {
			alertID := alertURL[alertsIndex+len("/alerts/"):]
			if queryIndex := strings.Index(alertID, "?"); queryIndex >= 0 {
				alertID = alertID[:queryIndex]
			}
			if strings.TrimSpace(alertID) != "" {
				out["alertId"] = strings.TrimSpace(alertID)
			}
		}
	}
	switch {
	case strings.Contains(text, ":no_entry:"):
		out["severity"] = "critical"
	case strings.Contains(text, ":warning:"):
		out["severity"] = "warning"
	case strings.Contains(text, ":white_check_mark:"):
		out["severity"] = "info"
	}
	if attachments, ok := raw["attachments"].([]any); ok && len(attachments) > 0 {
		if first, ok := attachments[0].(map[string]any); ok {
			if msg, _ := first["text"].(string); strings.TrimSpace(msg) != "" {
				out["message"] = strings.TrimSpace(msg)
			} else if fallback, _ := first["fallback"].(string); strings.TrimSpace(fallback) != "" {
				out["message"] = strings.TrimSpace(fallback)
			}
			if ts, ok := first["ts"].(float64); ok {
				seconds := int64(ts)
				nanoseconds := int64((ts - float64(seconds)) * 1e9)
				out["timestamp"] = time.Unix(seconds, nanoseconds).UTC().Format(time.RFC3339Nano)
			}
		}
	}
	return out
}
