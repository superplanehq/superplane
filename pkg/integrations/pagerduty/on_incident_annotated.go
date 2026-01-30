package pagerduty

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type OnIncidentAnnotated struct{}

type OnIncidentAnnotatedConfiguration struct {
	Service       string `json:"service"`
	ContentFilter string `json:"contentFilter"`
}

func (t *OnIncidentAnnotated) Name() string {
	return "pagerduty.onIncidentAnnotated"
}

func (t *OnIncidentAnnotated) Label() string {
	return "On Incident Annotated"
}

func (t *OnIncidentAnnotated) Description() string {
	return "Listen to incident annotation events"
}

func (t *OnIncidentAnnotated) Documentation() string {
	return `The On Incident Annotated trigger starts a workflow execution when a note is added to a PagerDuty incident.

## Use Cases

- **Note tracking**: Track when notes are added to incidents
- **Collaboration workflows**: Trigger actions based on incident annotations
- **Audit logging**: Log all notes added to incidents
- **Integration sync**: Sync notes with external ticketing systems

## Configuration

- **Service**: Select the PagerDuty service to monitor for incident annotations

## Event Data

Each annotation event includes:
- **agent**: Information about who added the note
- **incident**: Complete incident information

## Webhook Setup

This trigger automatically sets up a PagerDuty webhook subscription when configured. The subscription is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnIncidentAnnotated) Icon() string {
	return "message-square"
}

func (t *OnIncidentAnnotated) Color() string {
	return "gray"
}

func (t *OnIncidentAnnotated) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "service",
			Label:       "Service",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The PagerDuty service to monitor for incident annotations",
			Placeholder: "Select a service",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "service",
				},
			},
		},
		{
			Name:        "contentFilter",
			Label:       "Content Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., /trigger",
			Description: "Optional regex pattern to filter notes by content",
		},
	}
}

func (t *OnIncidentAnnotated) Setup(ctx core.TriggerContext) error {
	metadata := NodeMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	//
	// If metadata is already set, skip setup
	//
	if metadata.Service != nil {
		return nil
	}

	config := OnIncidentAnnotatedConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Service == "" {
		return fmt.Errorf("service is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	service, err := client.GetService(config.Service)
	if err != nil {
		return fmt.Errorf("error finding service: %v", err)
	}

	err = ctx.Metadata.Set(NodeMetadata{Service: service})
	if err != nil {
		return fmt.Errorf("error setting node metadata: %v", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Events: []string{"incident.annotated"},
		Filter: WebhookFilter{
			Type: "service_reference",
			ID:   config.Service,
		},
	})
}

func (t *OnIncidentAnnotated) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIncidentAnnotated) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIncidentAnnotated) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	log.Printf("[OnIncidentAnnotated] Received webhook request, body length: %d", len(ctx.Body))

	// Verify signature
	signature := ctx.Headers.Get("X-PagerDuty-Signature")
	if signature == "" {
		log.Printf("[OnIncidentAnnotated] Missing signature header")
		return http.StatusForbidden, fmt.Errorf("missing signature")
	}

	// Extract version and signature value (format: v1=<signature>)
	parts := strings.SplitN(signature, "=", 2)
	if len(parts) != 2 || parts[0] != "v1" {
		log.Printf("[OnIncidentAnnotated] Invalid signature format: %s", signature)
		return http.StatusForbidden, fmt.Errorf("invalid signature format")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		log.Printf("[OnIncidentAnnotated] Error getting secret: %v", err)
		return http.StatusInternalServerError, fmt.Errorf("error getting secret: %v", err)
	}

	// Verify signature using HMAC SHA256
	if err := crypto.VerifySignature(secret, ctx.Body, parts[1]); err != nil {
		log.Printf("[OnIncidentAnnotated] Invalid signature: %v", err)
		return http.StatusForbidden, fmt.Errorf("invalid signature: %v", err)
	}

	log.Printf("[OnIncidentAnnotated] Signature verified successfully")

	// Parse webhook payload
	var webhook Webhook
	err = json.Unmarshal(ctx.Body, &webhook)
	if err != nil {
		log.Printf("[OnIncidentAnnotated] Error parsing request body: %v", err)
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	eventType := webhook.Event.EventType
	log.Printf("[OnIncidentAnnotated] Received event type: %s", eventType)

	//
	// Only process incident.annotated events
	//
	if eventType != "incident.annotated" {
		log.Printf("[OnIncidentAnnotated] Ignoring event type: %s (expected incident.annotated)", eventType)
		return http.StatusOK, nil
	}

	// Parse configuration
	config := OnIncidentAnnotatedConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		log.Printf("[OnIncidentAnnotated] Error decoding configuration: %v", err)
		return http.StatusInternalServerError, fmt.Errorf("error decoding configuration: %v", err)
	}

	// Extract annotation content from the log_entries or directly from data
	annotationContent := extractAnnotationContent(webhook.Event.Data)
	log.Printf("[OnIncidentAnnotated] Annotation content: %s", annotationContent)

	// Apply content filter if configured
	if config.ContentFilter != "" {
		matched, err := regexp.MatchString(config.ContentFilter, annotationContent)
		if err != nil {
			log.Printf("[OnIncidentAnnotated] Error matching content filter: %v", err)
			return http.StatusInternalServerError, fmt.Errorf("invalid content filter regex: %v", err)
		}
		if !matched {
			log.Printf("[OnIncidentAnnotated] Content does not match filter, skipping event")
			return http.StatusOK, nil
		}
		log.Printf("[OnIncidentAnnotated] Content matches filter")
	}

	// Extract incident from the event data
	incident := extractAnnotatedIncident(webhook.Event.Data)

	log.Printf("[OnIncidentAnnotated] Emitting event: pagerduty.%s", eventType)

	err = ctx.Events.Emit(
		fmt.Sprintf("pagerduty.%s", eventType),
		buildAnnotatedPayload(webhook.Event.Agent, incident, annotationContent),
	)

	if err != nil {
		log.Printf("[OnIncidentAnnotated] Error emitting event: %v", err)
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	log.Printf("[OnIncidentAnnotated] Event emitted successfully")
	return http.StatusOK, nil
}

// extractAnnotationContent extracts the note content from the webhook data.
// PagerDuty V3 webhooks include the annotation in log_entries or as a channel.
func extractAnnotationContent(data map[string]any) string {
	if data == nil {
		return ""
	}

	// Try to get content from log_entries (PagerDuty V3 structure)
	if logEntries, ok := data["log_entries"].([]any); ok && len(logEntries) > 0 {
		if entry, ok := logEntries[0].(map[string]any); ok {
			if channel, ok := entry["channel"].(map[string]any); ok {
				if content, ok := channel["content"].(string); ok {
					return content
				}
			}
		}
	}

	// Try to get content directly from channel (alternative structure)
	if channel, ok := data["channel"].(map[string]any); ok {
		if content, ok := channel["content"].(string); ok {
			return content
		}
	}

	// Try to get content from note field
	if note, ok := data["note"].(map[string]any); ok {
		if content, ok := note["content"].(string); ok {
			return content
		}
	}

	// Try direct content field
	if content, ok := data["content"].(string); ok {
		return content
	}

	return ""
}

// extractAnnotatedIncident extracts the incident from the annotation event data.
func extractAnnotatedIncident(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}

	// The incident is typically at the root of event.data for incident.annotated
	// But check if it's nested under "incident" key first
	if incident, ok := data["incident"].(map[string]any); ok {
		return incident
	}

	// If the data itself looks like an incident (has id, type, status), return it
	if _, hasID := data["id"]; hasID {
		if dataType, ok := data["type"].(string); ok && dataType == "incident" {
			return data
		}
	}

	return data
}

// buildAnnotatedPayload creates the event payload for annotation events.
func buildAnnotatedPayload(agent map[string]any, incident map[string]any, annotationContent string) map[string]any {
	payload := map[string]any{}
	if agent != nil {
		payload["agent"] = agent
	}

	if incident != nil {
		payload["incident"] = incident
	}

	payload["annotation"] = map[string]any{
		"content": annotationContent,
	}

	return payload
}
