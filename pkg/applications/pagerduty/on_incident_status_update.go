package pagerduty

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type OnIncidentStatusUpdate struct{}

type OnIncidentStatusUpdateConfiguration struct {
	Service string `json:"service"`
}

func (t *OnIncidentStatusUpdate) Name() string {
	return "pagerduty.onIncidentStatusUpdate"
}

func (t *OnIncidentStatusUpdate) Label() string {
	return "On Incident Status Update"
}

func (t *OnIncidentStatusUpdate) Description() string {
	return "Listen to incident status update events"
}

func (t *OnIncidentStatusUpdate) Icon() string {
	return "alert-triangle"
}

func (t *OnIncidentStatusUpdate) Color() string {
	return "gray"
}

func (t *OnIncidentStatusUpdate) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "service",
			Label:       "Service",
			Type:        configuration.FieldTypeAppInstallationResource,
			Required:    true,
			Description: "The PagerDuty service to monitor for incident status updates",
			Placeholder: "Select a service",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "service",
				},
			},
		},
	}
}

func (t *OnIncidentStatusUpdate) Setup(ctx core.TriggerContext) error {
	log.Printf("[OnIncidentStatusUpdate] Setup called")

	metadata := NodeMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		log.Printf("[OnIncidentStatusUpdate] Failed to decode metadata: %v", err)
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	//
	// If metadata is already set, skip setup
	//
	if metadata.Service != nil {
		log.Printf("[OnIncidentStatusUpdate] Metadata already set, skipping setup. Service: %s", metadata.Service.Name)
		return nil
	}

	config := OnIncidentStatusUpdateConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		log.Printf("[OnIncidentStatusUpdate] Failed to decode configuration: %v", err)
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Service == "" {
		log.Printf("[OnIncidentStatusUpdate] Service is required but not provided")
		return fmt.Errorf("service is required")
	}

	log.Printf("[OnIncidentStatusUpdate] Creating client for service: %s", config.Service)

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		log.Printf("[OnIncidentStatusUpdate] Error creating client: %v", err)
		return fmt.Errorf("error creating client: %v", err)
	}

	service, err := client.GetService(config.Service)
	if err != nil {
		log.Printf("[OnIncidentStatusUpdate] Error finding service: %v", err)
		return fmt.Errorf("error finding service: %v", err)
	}

	log.Printf("[OnIncidentStatusUpdate] Found service: %s (%s)", service.Name, service.ID)

	err = ctx.Metadata.Set(NodeMetadata{Service: service})
	if err != nil {
		log.Printf("[OnIncidentStatusUpdate] Error setting node metadata: %v", err)
		return fmt.Errorf("error setting node metadata: %v", err)
	}

	log.Printf("[OnIncidentStatusUpdate] Requesting webhook for event: incident.status_update_published, filter: service_reference/%s", config.Service)

	return ctx.AppInstallation.RequestWebhook(WebhookConfiguration{
		Events: []string{"incident.status_update_published"},
		Filter: WebhookFilter{
			Type: "service_reference",
			ID:   config.Service,
		},
	})
}

func (t *OnIncidentStatusUpdate) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIncidentStatusUpdate) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIncidentStatusUpdate) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	log.Printf("[OnIncidentStatusUpdate] Received webhook request, body length: %d", len(ctx.Body))

	// Verify signature
	signature := ctx.Headers.Get("X-PagerDuty-Signature")
	if signature == "" {
		log.Printf("[OnIncidentStatusUpdate] Missing signature header")
		return http.StatusForbidden, fmt.Errorf("missing signature")
	}

	// Extract version and signature value (format: v1=<signature>)
	parts := strings.SplitN(signature, "=", 2)
	if len(parts) != 2 || parts[0] != "v1" {
		log.Printf("[OnIncidentStatusUpdate] Invalid signature format: %s", signature)
		return http.StatusForbidden, fmt.Errorf("invalid signature format")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		log.Printf("[OnIncidentStatusUpdate] Error getting secret: %v", err)
		return http.StatusInternalServerError, fmt.Errorf("error getting secret: %v", err)
	}

	// Verify signature using HMAC SHA256
	if err := crypto.VerifySignature(secret, ctx.Body, parts[1]); err != nil {
		log.Printf("[OnIncidentStatusUpdate] Invalid signature: %v", err)
		return http.StatusForbidden, fmt.Errorf("invalid signature: %v", err)
	}

	log.Printf("[OnIncidentStatusUpdate] Signature verified successfully")

	// Parse webhook payload
	var webhook Webhook
	err = json.Unmarshal(ctx.Body, &webhook)
	if err != nil {
		log.Printf("[OnIncidentStatusUpdate] Error parsing request body: %v", err)
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	eventType := webhook.Event.EventType
	log.Printf("[OnIncidentStatusUpdate] Received event type: %s", eventType)

	//
	// Only process incident.status_update_published events
	//
	if eventType != "incident.status_update_published" {
		log.Printf("[OnIncidentStatusUpdate] Ignoring event type: %s (expected incident.status_update_published)", eventType)
		return http.StatusOK, nil
	}

	//
	// Extract incident reference from the status update data
	//
	incident := extractIncident(webhook.Event.Data)

	log.Printf("[OnIncidentStatusUpdate] Emitting event: pagerduty.%s", eventType)

	err = ctx.Events.Emit(
		fmt.Sprintf("pagerduty.%s", eventType),
		buildStatusUpdatePayload(webhook.Event.Agent, webhook.Event.Data, incident),
	)

	if err != nil {
		log.Printf("[OnIncidentStatusUpdate] Error emitting event: %v", err)
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	log.Printf("[OnIncidentStatusUpdate] Event emitted successfully")
	return http.StatusOK, nil
}

// extractIncident extracts the incident reference from the status update data.
func extractIncident(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}

	incident, ok := data["incident"].(map[string]any)
	if ok {
		return incident
	}

	return nil
}

// buildStatusUpdatePayload creates the event payload for status update events.
func buildStatusUpdatePayload(agent map[string]any, statusUpdate map[string]any, incident map[string]any) map[string]any {
	payload := map[string]any{}
	if agent != nil {
		payload["agent"] = agent
	}

	if statusUpdate != nil {
		payload["status_update"] = statusUpdate
	}

	if incident != nil {
		payload["incident"] = incident
	}

	return payload
}
