package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	alertActionCreate      = "Create"
	alertActionAcknowledge = "Acknowledge"
	alertActionClose       = "Close"
	alertActionDelete      = "Delete"

	// AlertEventPayloadType is the event type emitted for every matching alert webhook.
	AlertEventPayloadType = "jira.alert"
)

type OnAlert struct{}

type OnAlertConfiguration struct {
	Team   string   `json:"team" mapstructure:"team"`
	Events []string `json:"events" mapstructure:"events"`
}

type OnAlertMetadata struct {
	Team *OpsTeam `json:"team,omitempty" mapstructure:"team,omitempty"`
}

// AlertEvent is a JSM Ops alert, as sent in alert action webhook payloads.
type AlertEvent struct {
	AlertID  string   `json:"alertId,omitempty"`
	TinyID   string   `json:"tinyId,omitempty"`
	Alias    string   `json:"alias,omitempty"`
	Message  string   `json:"message,omitempty"`
	Priority string   `json:"priority,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Source   string   `json:"source,omitempty"`
	Username string   `json:"username,omitempty"`
}

// AlertWebhookPayload is the shape JSM Ops sends for alert action webhooks.
type AlertWebhookPayload struct {
	Action string      `json:"action"`
	Alert  *AlertEvent `json:"alert"`
}

// OnAlertEvent is the event SuperPlane emits for each matching alert webhook.
type OnAlertEvent struct {
	Action string      `json:"action"`
	Alert  *AlertEvent `json:"alert"`
}

func (t *OnAlert) Name() string {
	return "jira.onAlert"
}

func (t *OnAlert) Label() string {
	return "On Alert"
}

func (t *OnAlert) Description() string {
	return "Listen to alert created, acknowledged, closed, or deleted events in JSM Ops"
}

func (t *OnAlert) Documentation() string {
	return `The On Alert trigger starts a workflow execution when an alert changes state in Jira Service Management Operations (Ops).

## Use Cases

- **Escalation automation**: Notify a channel or page someone when a new alert fires
- **Incident linkage**: Open a Jira incident when a high-priority alert is created
- **Reporting**: Track alert volume and acknowledgement time

## Configuration

- **Team** (optional): JSM Operations team to scope alerts to; leave empty to listen for alerts from every team
- **Events**: Which alert events to listen for (Created, Acknowledged, Closed, Deleted)

## Webhook Setup

This is provisioned automatically as a dedicated outgoing Webhook integration in JSM Ops for this
trigger - unlike ` + "`jira.onIssue`" + `, JSM Ops has no "one URL per connection" limit, so each
` + "`jira.onAlert`" + ` trigger gets its own registration, removed automatically when the trigger is deleted.

**Requires a Premium or Enterprise Jira Service Management plan** - Atlassian gates outgoing
webhook integrations to those tiers, and requires the ` + "`enableOpsFeatures`" + ` option on this connection
to be turned on (Settings tab), since it needs the JSM Ops OAuth scopes.

## Output

Emits one event per matching alert webhook with:
- **action**: ` + "`created`" + `, ` + "`acknowledged`" + `, ` + "`closed`" + `, or ` + "`deleted`" + `
- **alert**: The alert (alertId, tinyId, alias, message, priority, tags, source, username)`
}

func (t *OnAlert) Icon() string {
	return "jira"
}

func (t *OnAlert) Color() string {
	return "red"
}

func (t *OnAlert) ExampleData() map[string]any {
	return onAlertExampleData()
}

func (t *OnAlert) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "team",
			Label:       "Team",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "JSM Operations team to scope alerts to - leave empty to listen for alerts from every team",
			Placeholder: "All teams",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "team",
				},
			},
		},
		{
			Name:        "events",
			Label:       "Events",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    true,
			Default:     []string{"created"},
			Description: "Which alert events to listen for",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "created"},
						{Label: "Acknowledged", Value: "acknowledged"},
						{Label: "Closed", Value: "closed"},
						{Label: "Deleted", Value: "deleted"},
					},
				},
			},
		},
	}
}

func (t *OnAlert) Setup(ctx core.TriggerContext) error {
	config := OnAlertConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if len(config.Events) == 0 {
		return fmt.Errorf("at least one event must be selected")
	}

	metadata := OnAlertMetadata{}
	if config.Team != "" {
		team, err := requireOpsTeam(ctx.HTTP, ctx.Integration, config.Team)
		if err != nil {
			return err
		}
		metadata.Team = team
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{Kind: webhookKindAlert, TeamID: config.Team})
}

func (t *OnAlert) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnAlert) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	config := OnAlertConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	payload := AlertWebhookPayload{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %w", err)
	}

	action, ok := alertEventAction(payload.Action)
	if !ok {
		ctx.Logger.Infof("Ignoring event - unsupported action %q", payload.Action)
		return http.StatusOK, nil, nil
	}

	if !slices.Contains(config.Events, action) {
		ctx.Logger.Infof("Ignoring event - action %q is not configured", action)
		return http.StatusOK, nil, nil
	}

	if payload.Alert == nil {
		ctx.Logger.Info("Ignoring event - missing alert")
		return http.StatusOK, nil, nil
	}

	event := OnAlertEvent{Action: action, Alert: payload.Alert}
	if err := ctx.Events.Emit(AlertEventPayloadType, event); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %w", err)
	}

	return http.StatusOK, nil, nil
}

// Cleanup does nothing: the dedicated JSM Ops integration requested via RequestWebhook is torn
// down by the platform's webhook cleanup worker (through JiraWebhookHandler.Cleanup) once this
// trigger is removed.
func (t *OnAlert) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// alertEventAction maps a native JSM Ops alert action to this trigger's short action name.
func alertEventAction(action string) (string, bool) {
	switch action {
	case alertActionCreate:
		return "created", true
	case alertActionAcknowledge:
		return "acknowledged", true
	case alertActionClose:
		return "closed", true
	case alertActionDelete:
		return "deleted", true
	default:
		return "", false
	}
}
