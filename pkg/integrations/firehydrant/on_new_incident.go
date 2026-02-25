package firehydrant

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnNewIncident struct{}

type OnNewIncidentConfiguration struct {
	Events []string `json:"events" mapstructure:"events"`
}

func (t *OnNewIncident) Name() string {
	return "firehydrant.onNewIncident"
}

func (t *OnNewIncident) Label() string {
	return "On New Incident"
}

func (t *OnNewIncident) Description() string {
	return "Listen for new incident events in FireHydrant"
}

func (t *OnNewIncident) Documentation() string {
	return `The On New Incident trigger starts a workflow execution when a new incident is created in FireHydrant.

## Use Cases

- **Incident automation**: Automate responses when new incidents are created
- **Notification workflows**: Send notifications to Slack, email, or other channels when an incident is opened
- **Integration workflows**: Create Jira tickets, update status pages, or sync with other systems when incidents are created
- **Escalation workflows**: Trigger escalation processes when new incidents are detected

## Configuration

- **Events**: Select which incident events to listen for (currently focused on new incident creation)

## Event Data

Each incident event includes:
- **event**: Event type identifier
- **incident**: Complete incident information including name, severity, description, status, and more

## Webhook Setup

This trigger automatically sets up a FireHydrant webhook endpoint when configured. The endpoint is managed by SuperPlane and will be cleaned up when the trigger is removed.

FireHydrant verifies webhooks using HMAC-SHA256 signatures sent in the ` + "`fh-signature`" + ` header.`
}

func (t *OnNewIncident) Icon() string {
	return "alert-triangle"
}

func (t *OnNewIncident) Color() string {
	return "gray"
}

func (t *OnNewIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "events",
			Label:    "Events",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"incident_created"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Incident Created", Value: "incident_created"},
					},
				},
			},
		},
	}
}

func (t *OnNewIncident) Setup(ctx core.TriggerContext) error {
	config := OnNewIncidentConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if len(config.Events) == 0 {
		return fmt.Errorf("at least one event type must be chosen")
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Events: config.Events,
	})
}

func (t *OnNewIncident) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnNewIncident) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnNewIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnNewIncidentConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	signature := ctx.Headers.Get("Fh-Signature")
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting secret: %v", err)
	}

	if err := verifyWebhookSignature(signature, ctx.Body, secret); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature: %v", err)
	}

	var payload WebhookPayload
	err = json.Unmarshal(ctx.Body, &payload)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	eventType := payload.Type

	// FireHydrant sends all incident events on the same webhook. We only
	// process events that match the configured event types.
	if !isConfiguredEvent(config.Events, eventType) {
		return http.StatusOK, nil
	}

	err = ctx.Events.Emit(
		fmt.Sprintf("firehydrant.%s", eventType),
		buildNewIncidentPayload(payload),
	)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

// WebhookPayload represents a FireHydrant webhook payload.
type WebhookPayload struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

func isConfiguredEvent(configured []string, eventType string) bool {
	for _, e := range configured {
		if e == eventType {
			return true
		}
	}
	return false
}

func buildNewIncidentPayload(webhook WebhookPayload) map[string]any {
	payload := map[string]any{
		"event": webhook.Type,
	}

	if webhook.Data != nil {
		payload["incident"] = webhook.Data
	}

	return payload
}

func (t *OnNewIncident) Cleanup(ctx core.TriggerContext) error {
	return nil
}
