package rootly

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIncidentCreated struct{}

type OnIncidentCreatedConfiguration struct {
	SeverityFilter []string `json:"severityFilter"`
	ServiceFilter  []string `json:"serviceFilter"`
	TeamFilter     []string `json:"teamFilter"`
}

func (t *OnIncidentCreated) Name() string {
	return "rootly.onIncidentCreated"
}

func (t *OnIncidentCreated) Label() string {
	return "On Incident Created"
}

func (t *OnIncidentCreated) Description() string {
	return "Listen to incident created events"
}

func (t *OnIncidentCreated) Documentation() string {
	return `The On Incident Created trigger starts a workflow execution when a Rootly incident is created.

## Use Cases

- **Incident automation**: Automate responses when new incidents are created
- **Notification workflows**: Send notifications when incidents are created
- **Integration workflows**: Sync new incidents with external systems

## Configuration

- **Severity Filter** (optional): Only trigger for incidents with specific severity
- **Service Filter** (optional): Only trigger for incidents attached to specific Rootly services
- **Team Filter** (optional): Only trigger for incidents attached to specific Rootly teams

## Event Data

Each incident created event includes:
- **event**: incident.created
- **incident**: Complete incident information including title, summary, severity, status

## Webhook Setup

This trigger automatically sets up a Rootly webhook endpoint when configured. The endpoint is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnIncidentCreated) Icon() string {
	return "alert-triangle"
}

func (t *OnIncidentCreated) Color() string {
	return "gray"
}

func (t *OnIncidentCreated) Configuration() []configuration.Field {
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

func (t *OnIncidentCreated) Setup(ctx core.TriggerContext) error {
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Events: []string{"incident.created"},
	})
}

func (t *OnIncidentCreated) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIncidentCreated) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIncidentCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnIncidentCreatedConfiguration{}
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

	// Only process incident.created events
	if webhook.Event.Type != "incident.created" {
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
		"rootly.incident.created",
		buildIncidentPayload(webhook),
	)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

// WebhookPayload represents the Rootly webhook payload
type WebhookPayload struct {
	Event WebhookEvent   `json:"event"`
	Data  map[string]any `json:"data"`
}

// WebhookEvent represents the event metadata in a Rootly webhook
type WebhookEvent struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	IssuedAt string `json:"issued_at"`
}

func buildIncidentPayload(webhook WebhookPayload) map[string]any {
	payload := map[string]any{
		"event":     webhook.Event.Type,
		"event_id":  webhook.Event.ID,
		"issued_at": webhook.Event.IssuedAt,
	}

	if webhook.Data != nil {
		payload["incident"] = webhook.Data
	}

	return payload
}