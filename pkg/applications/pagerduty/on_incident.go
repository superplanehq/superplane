package pagerduty

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

type OnIncident struct{}

type OnIncidentConfiguration struct {
	Events    []string `json:"events"`
	ServiceID string   `json:"serviceId"`
	TeamID    string   `json:"teamId"`
	Urgency   []string `json:"urgency"`
}

type OnIncidentMetadata struct{
	WebhookRegistered bool `json:"webhookRegistered"`
}

func (t *OnIncident) Name() string {
	return "pagerduty.onIncident"
}

func (t *OnIncident) Label() string {
	return "On Incident"
}

func (t *OnIncident) Description() string {
	return "Listen to incident events"
}

func (t *OnIncident) Icon() string {
	return "alert-triangle"
}

func (t *OnIncident) Color() string {
	return "red"
}

func (t *OnIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "events",
			Label:    "Events",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"triggered"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Triggered", Value: "triggered"},
						{Label: "Acknowledged", Value: "acknowledged"},
						{Label: "Resolved", Value: "resolved"},
						{Label: "Unacknowledged", Value: "unacknowledged"},
						{Label: "Reopened", Value: "reopened"},
						{Label: "Reassigned", Value: "reassigned"},
						{Label: "Delegated", Value: "delegated"},
						{Label: "Escalated", Value: "escalated"},
						{Label: "Incident Type Changed", Value: "incident_type.changed"},
						{Label: "Priority Updated", Value: "priority_updated"},
						{Label: "Service Updated", Value: "service_updated"},
					},
				},
			},
		},
		{
			Name:        "serviceId",
			Label:       "Service ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional: filter incidents by service ID",
			Placeholder: "e.g. PXXXXXX",
		},
		{
			Name:        "teamId",
			Label:       "Team ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional: filter incidents by team ID",
			Placeholder: "e.g. PXXXXXX",
		},
		{
			Name:     "urgency",
			Label:    "Urgency Filter",
			Type:     configuration.FieldTypeMultiSelect,
			Required: false,
			Description: "Filter incidents by urgency. Leave empty to receive all urgencies.",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "High", Value: "high"},
						{Label: "Low", Value: "low"},
					},
				},
			},
		},
	}
}

func (t *OnIncident) Setup(ctx core.TriggerContext) error {
	var metadata OnIncidentMetadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// If webhook already registered, nothing to do
	if metadata.WebhookRegistered {
		return nil
	}

	config := OnIncidentConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Build full event names
	fullEventNames := make([]string, len(config.Events))
	for i, event := range config.Events {
		fullEventNames[i] = fmt.Sprintf("incident.%s", event)
	}

	// Request webhook from app installation
	err = ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		Events:    fullEventNames,
		ServiceID: config.ServiceID,
		TeamID:    config.TeamID,
	})
	if err != nil {
		return err
	}

	// Mark as registered
	metadata.WebhookRegistered = true
	return ctx.MetadataContext.Set(metadata)
}

func (t *OnIncident) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIncident) HandleAction(ctx core.TriggerActionContext) error {
	return nil
}

func (t *OnIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnIncidentConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Verify signature
	signature := ctx.Headers.Get("X-PagerDuty-Signature")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing signature")
	}

	// Extract version and signature value (format: v1=<signature>)
	parts := strings.SplitN(signature, "=", 2)
	if len(parts) != 2 || parts[0] != "v1" {
		return http.StatusForbidden, fmt.Errorf("invalid signature format")
	}

	secret, err := ctx.WebhookContext.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting secret: %v", err)
	}

	// Verify signature using HMAC SHA256
	if err := crypto.VerifySignature(secret, ctx.Body, parts[1]); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature: %v", err)
	}

	// Parse webhook payload
	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// Filter by event type (webhook may be shared and receive more events than this trigger cares about)
	if !whitelistedEvent(data, config.Events, "incident") {
		return http.StatusOK, nil
	}

	// Filter by urgency if configured
	if !filterByUrgency(data, config.Urgency) {
		return http.StatusOK, nil
	}

	// Emit event
	err = ctx.EventContext.Emit("pagerduty.incident", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

// whitelistedEvent checks if the event type in the payload matches the allowed events
func whitelistedEvent(data map[string]any, allowed []string, eventPrefix string) bool {
	event, ok := data["event"].(map[string]any)
	if !ok {
		return false
	}

	eventType, ok := event["event_type"].(string)
	if !ok {
		return false
	}

	// Extract sub-event (e.g., "incident.triggered" → "triggered")
	subEvent := strings.TrimPrefix(eventType, eventPrefix+".")

	return slices.Contains(allowed, subEvent)
}

// filterByUrgency checks if the incident urgency matches the allowed urgencies
func filterByUrgency(data map[string]any, allowed []string) bool {
	if len(allowed) == 0 {
		return true // No filter means allow all
	}

	event, ok := data["event"].(map[string]any)
	if !ok {
		return false
	}

	dataPayload, ok := event["data"].(map[string]any)
	if !ok {
		return false
	}

	urgency, ok := dataPayload["urgency"].(string)
	if !ok {
		return false
	}

	return slices.Contains(allowed, urgency)
}
