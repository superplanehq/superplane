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
	Service   string   `json:"service"`
	Events    []string `json:"events"`
	Urgencies []string `json:"urgencies"`
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
	return "gray"
}

func (t *OnIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "events",
			Label:    "Events",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"incident.triggered"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Triggered", Value: "incident.triggered"},
						{Label: "Acknowledged", Value: "incident.acknowledged"},
						{Label: "Resolved", Value: "incident.resolved"},
					},
				},
			},
		},
		{
			Name:        "service",
			Label:       "Service",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. PXXXXXX",
		},
		{
			Name:        "urgencies",
			Label:       "Urgencies",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Default:     []string{"low", "high"},
			Description: "Filter incidents by urgency",
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
	metadata := NodeMetadata{}
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	//
	// If metadata is already set, skip setup
	//
	if metadata.Service != nil {
		return nil
	}

	config := OnIncidentConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if len(config.Events) == 0 {
		return fmt.Errorf("at least one event type must be chosen")
	}

	if config.Service == "" {
		return fmt.Errorf("service is required")
	}

	client, err := NewClient(ctx.AppInstallationContext)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	service, err := client.GetService(config.Service)
	if err != nil {
		return fmt.Errorf("error finding service: %v", err)
	}

	err = ctx.MetadataContext.Set(NodeMetadata{Service: service})
	if err != nil {
		return fmt.Errorf("error setting node metadata: %v", err)
	}

	return ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		Events: config.Events,
		Filter: WebhookFilter{
			Type: "service_reference",
			ID:   config.Service,
		},
	})
}

func (t *OnIncident) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIncident) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
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

	eventType := getEventType(data)

	//
	// Since the webhook may be shared and receive more events than this trigger cares about,
	// we need to filter events by their type here.
	//
	if !slices.Contains(config.Events, eventType) {
		return http.StatusOK, nil
	}

	if !filterByUrgency(data, config.Urgencies) {
		return http.StatusOK, nil
	}

	err = ctx.EventContext.Emit(fmt.Sprintf("pagerduty.%s", eventType), data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func getEventType(data map[string]any) string {
	event, ok := data["event"].(map[string]any)
	if !ok {
		return ""
	}

	eventType, ok := event["event_type"].(string)
	if !ok {
		return ""
	}

	return eventType
}

func filterByUrgency(data map[string]any, allowed []string) bool {
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
