package servicenow

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIncident struct{}

type OnIncidentConfiguration struct {
	Events []string `json:"events"`
}

func (t *OnIncident) Name() string {
	return "servicenow.onIncident"
}

func (t *OnIncident) Label() string {
	return "On Incident"
}

func (t *OnIncident) Description() string {
	return "Listen to incident events from ServiceNow"
}

func (t *OnIncident) Documentation() string {
	return `The On Incident trigger starts a workflow execution when ServiceNow incident events are received via webhook.

## Use Cases

- **Incident automation**: Automate responses when incidents are created or updated
- **Notification workflows**: Send notifications when new incidents are created
- **Integration workflows**: Sync incidents with external systems
- **Escalation handling**: Handle incident updates automatically

## Configuration

- **Events**: Select which incident events to listen for (insert, update, delete)

## Scoping

Incident scoping (e.g. filtering by assignment group, category, or priority) is configured in the ServiceNow Business Rule conditions. This allows control over which incidents trigger the webhook.

## Required Permissions

Creating the Business Rule in ServiceNow requires the **admin** role. This is a one-time setup step.

## Business Rule Setup

This trigger provides a webhook URL and a ready-to-use Business Rule script:

1. In ServiceNow, navigate to **System Definition > Business Rules** and create a new rule
2. Set the table to ` + "`incident`" + `, set **When** to **after**, and check **insert**, **update**, and/or **delete** as needed
3. Check **Advanced** and paste the generated script into the **Script** field
4. The script uses ` + "`sn_ws.RESTMessageV2`" + ` to send incident data to the webhook URL with the secret for authentication

## Event Data

Each incident event includes the full incident record from ServiceNow, including:
- **sys_id**: Unique identifier
- **number**: Human-readable incident number
- **short_description**: Brief summary
- **state**: Current state
- **urgency**: Urgency level
- **impact**: Impact level
- **assignment_group**: Assigned group
- **assigned_to**: Assigned user`
}

func (t *OnIncident) Icon() string {
	return "servicenow"
}

func (t *OnIncident) Color() string {
	return "gray"
}

func (t *OnIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "events",
			Label:       "Events",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    true,
			Default:     []string{"insert"},
			Description: "Which incident events to listen for",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Insert", Value: "insert"},
						{Label: "Update", Value: "update"},
						{Label: "Delete", Value: "delete"},
					},
				},
			},
		},
	}
}

func (t *OnIncident) Setup(ctx core.TriggerContext) error {
	metadata := NodeMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	config := OnIncidentConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if len(config.Events) == 0 {
		return fmt.Errorf("at least one event type must be chosen")
	}

	if metadata.WebhookURL != "" {
		return nil
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("error setting up webhook: %v", err)
	}

	return ctx.Metadata.Set(NodeMetadata{WebhookURL: webhookURL})
}

func (t *OnIncident) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "resetAuthentication",
			Description:    "Reset/regenerate webhook secret",
			UserAccessible: true,
			Parameters:     []configuration.Field{},
		},
	}
}

func (t *OnIncident) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case "resetAuthentication":
		plainKey, _, err := ctx.Webhook.ResetSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to reset webhook secret: %w", err)
		}

		return map[string]any{"secret": string(plainKey)}, nil
	}

	return nil, fmt.Errorf("action %s not supported", ctx.Name)
}

func (t *OnIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnIncidentConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	secretHeader := ctx.Headers.Get("X-Webhook-Secret")
	if secretHeader == "" {
		return http.StatusForbidden, fmt.Errorf("missing X-Webhook-Secret header")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting secret: %v", err)
	}

	if subtle.ConstantTimeCompare([]byte(secretHeader), secret) != 1 {
		return http.StatusForbidden, fmt.Errorf("invalid webhook secret")
	}

	// Parse the webhook payload.
	// ServiceNow REST Messages may include control characters (\r, \n, tabs)
	// from line-wrapped content templates, which Go's JSON parser rejects.
	// Use a lenient decoder that handles these cases.
	var payload WebhookPayload
	decoder := json.NewDecoder(bytes.NewReader(
		bytes.Map(func(r rune) rune {
			// Strip control characters except those valid in JSON strings
			if r < 0x20 && r != '\t' {
				return -1
			}
			return r
		}, ctx.Body),
	))
	err = decoder.Decode(&payload)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	if payload.EventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing event_type in payload")
	}

	if !slices.Contains(config.Events, payload.EventType) {
		return http.StatusOK, nil
	}

	err = ctx.Events.Emit(
		PayloadTypeIncident+"."+payload.EventType,
		payload.Incident,
	)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

type WebhookPayload struct {
	EventType string         `json:"event_type"`
	Incident  map[string]any `json:"incident"`
}

func (t *OnIncident) Cleanup(ctx core.TriggerContext) error {
	return nil
}
