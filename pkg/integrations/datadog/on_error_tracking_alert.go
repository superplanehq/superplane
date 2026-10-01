package datadog

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/logging"
)

const ErrorTrackingAlertPayloadType = "datadog.errorTrackingAlert"

type OnErrorTrackingAlert struct{}

type OnErrorTrackingAlertMetadata struct {
	SubscriptionID string `json:"subscriptionId,omitempty" mapstructure:"subscriptionId"`
}

type OnErrorTrackingAlertConfiguration struct {
	Service          string   `json:"service" mapstructure:"service"`
	AlertTransitions []string `json:"alertTransitions" mapstructure:"alertTransitions"`
}

type ErrorTrackingAlertPayload struct {
	ID              string `json:"id" mapstructure:"id"`
	EventType       string `json:"event_type" mapstructure:"event_type"`
	Title           string `json:"title" mapstructure:"title"`
	Body            string `json:"body" mapstructure:"body"`
	EventMessage    string `json:"event_message,omitempty" mapstructure:"event_message"`
	AlertID         string `json:"alert_id" mapstructure:"alert_id"`
	AlertTransition string `json:"alert_transition" mapstructure:"alert_transition"`
	Tags            string `json:"tags" mapstructure:"tags"`
	AlertQuery      string `json:"alert_query,omitempty" mapstructure:"alert_query"`
	AlertScope      string `json:"alert_scope,omitempty" mapstructure:"alert_scope"`
	AggregKey       string `json:"aggreg_key,omitempty" mapstructure:"aggreg_key"`
	Link            string `json:"link" mapstructure:"link"`
	Description     string `json:"description,omitempty" mapstructure:"description"`
	Environment     string `json:"environment,omitempty" mapstructure:"environment"`
	ReceiptID       string `json:"superplaneReceiptId,omitempty" mapstructure:"superplaneReceiptId"`
}

func (t *OnErrorTrackingAlert) Name() string {
	return "datadog.onErrorTrackingAlert"
}

func (t *OnErrorTrackingAlert) Label() string {
	return "On Error Tracking Alert"
}

func (t *OnErrorTrackingAlert) Description() string {
	return "Listen to Datadog Error Tracking monitor alerts"
}

func (t *OnErrorTrackingAlert) Documentation() string {
	return `The On Error Tracking Alert trigger starts a workflow when Datadog sends a Triggered Error Tracking monitor alert to SuperPlane.

## Setup

Connect Datadog in SuperPlane. SuperPlane creates a webhook named ` + "`superplane`" + `. Add ` + "`@webhook-superplane`" + ` to the notification message of an Error Tracking New Issue monitor.

Select a service to keep alerts of that service. Set the Error Tracking monitor query to ` + "`service:<name>`" + `.

Select the alert transitions to keep. Empty keeps Triggered. A Re-Triggered alert can create a second task for the same issue.

## Event Data

The trigger emits:
- **title**: error type and message when SuperPlane can load the issue. Otherwise the monitor title
- **description**: markdown for the work order. SuperPlane loads the Error Tracking issue and includes the error message, file, function, service, versions, impact, ownership, and a sample stack trace when Datadog returns them. With ` + "`apm_read`" + ` or ` + "`logs_read_data`" + `, it also adds the error sample (time, environment, request, user, request ID, and trace link) and up to 25 logs that share the sample trace
- **body**: the same markdown as description, so an existing intake still receives the issue details
- **link**: Datadog link from the alert
- **alert_id**: alerting monitor ID
- **tags**: comma-separated tags from the alert
- **alert_query**: monitor query that fired
- **alert_scope**: tags that triggered the alert
- **alert_transition**: alert transition, such as Triggered
- **environment**: env tag from the alert scope, then from the tags, then from the error sample`
}

func (t *OnErrorTrackingAlert) Icon() string {
	return "chart-bar"
}

func (t *OnErrorTrackingAlert) Color() string {
	return "gray"
}

func (t *OnErrorTrackingAlert) ExampleData() map[string]any {
	return onErrorTrackingAlertExampleData()
}

func (t *OnErrorTrackingAlert) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "service",
			Label:       "Service",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Only trigger for Error Tracking alerts of this service",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeService,
				},
			},
		},
		{
			Name:        "alertTransitions",
			Label:       "Alert transitions",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Default:     []string{AlertTransitionTriggered},
			Description: "Listen for these Error Tracking monitor transitions. Empty keeps Triggered.",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Triggered", Value: AlertTransitionTriggered},
						{Label: "Re-Triggered", Value: AlertTransitionRetriggered},
					},
				},
			},
		},
	}
}

func (t *OnErrorTrackingAlert) Setup(ctx core.TriggerContext) error {
	metadata := OnErrorTrackingAlertMetadata{}
	if ctx.Metadata != nil {
		if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
			return fmt.Errorf("failed to decode metadata: %w", err)
		}
	}

	if metadata.SubscriptionID != "" {
		return nil
	}

	subscriptionID, err := ctx.Integration.Subscribe(SubscriptionConfiguration{})
	if err != nil {
		return fmt.Errorf("failed to subscribe to datadog alerts: %w", err)
	}

	metadata.SubscriptionID = subscriptionID.String()
	return ctx.Metadata.Set(metadata)
}

func (t *OnErrorTrackingAlert) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnErrorTrackingAlert) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnErrorTrackingAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (t *OnErrorTrackingAlert) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	config := OnErrorTrackingAlertConfiguration{}
	if ctx.Configuration != nil {
		if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
			return fmt.Errorf("failed to decode configuration: %w", err)
		}
	}

	payload, err := decodeErrorTrackingAlertPayload(ctx.Message)
	if err != nil {
		return err
	}

	if !strings.EqualFold(payload.EventType, ErrorTrackingAlertEventType) {
		return nil
	}

	if !payloadMatchesAlertTransition(payload, config.AlertTransitions) {
		return nil
	}

	if !payloadMatchesService(payload, config.Service) {
		return nil
	}

	enriched, issue := enrichErrorTrackingAlert(ctx, payload)
	if !issueServiceMatches(issue, config.Service) {
		return nil
	}

	if err := ctx.Events.Emit(ErrorTrackingAlertPayloadType, enriched); err != nil {
		return err
	}

	logDatadogWebhookDelivered(ctx, enriched)
	return nil
}

func logDatadogWebhookDelivered(ctx core.IntegrationMessageContext, payload ErrorTrackingAlertPayload) {
	identity := logging.DatadogWebhookIdentity(ctx.Logger)
	logging.LogDatadogWebhookInfo("Datadog webhook delivered", log.Fields{
		"outcome":           datadogWebhookOutcomeDelivered,
		"event_type":        payload.EventType,
		"alert_transition":  payload.AlertTransition,
		"organization_id":   datadogIdentityField(identity, "organization_id"),
		"organization_name": datadogIdentityField(identity, "organization_name"),
		"integration_id":    datadogIdentityField(identity, "integration_id"),
		"workspace_id":      datadogIdentityField(identity, "workspace_id"),
		"workspace_name":    datadogIdentityField(identity, "workspace_name"),
		"intake_id":         datadogIdentityField(identity, "intake_id"),
		"intake_name":       datadogIdentityField(identity, "intake_name"),
	}, nil)
}

func datadogIdentityField(fields log.Fields, key string) string {
	if fields == nil {
		return ""
	}

	value, ok := fields[key]
	if !ok || value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func (t *OnErrorTrackingAlert) Cleanup(ctx core.TriggerContext) error {
	return nil
}

type SubscriptionConfiguration struct{}

func decodeErrorTrackingAlertPayload(message any) (ErrorTrackingAlertPayload, error) {
	payload := ErrorTrackingAlertPayload{}
	if err := mapstructure.Decode(message, &payload); err != nil {
		return ErrorTrackingAlertPayload{}, fmt.Errorf("failed to decode datadog alert payload: %w", err)
	}
	return payload, nil
}

func issueServiceMatches(issue *ErrorTrackingIssue, service string) bool {
	service = strings.TrimSpace(service)
	if service == "" || issue == nil {
		return true
	}

	actual := strings.TrimSpace(issue.Service)
	if actual == "" {
		return true
	}

	return strings.EqualFold(actual, service)
}

func payloadMatchesService(payload ErrorTrackingAlertPayload, service string) bool {
	service = strings.TrimSpace(service)
	if service == "" {
		return true
	}

	return containsTagToken(payload.AlertQuery, "service", service) ||
		containsTagToken(payload.AlertScope, "service", service) ||
		containsTagToken(payload.Tags, "service", service)
}

func containsServiceToken(text, service string) bool {
	return containsTagToken(text, "service", service)
}

func containsTagToken(text, key, value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}

	lower := strings.ToLower(text)
	needle := strings.ToLower(strings.TrimSpace(key)) + ":" + strings.ToLower(value)
	if strings.HasSuffix(needle, ":") {
		return false
	}

	start := 0
	for {
		index := strings.Index(lower[start:], needle)
		if index < 0 {
			return false
		}
		index += start
		beforeOK := index == 0 || !isServiceNameChar(rune(lower[index-1]))
		after := index + len(needle)
		afterOK := after == len(lower) || !isServiceNameChar(rune(lower[after]))
		if beforeOK && afterOK {
			return true
		}
		start = after
	}
}

func firstTagValue(text, key string) string {
	lower := strings.ToLower(text)
	needle := strings.ToLower(strings.TrimSpace(key)) + ":"
	if needle == ":" {
		return ""
	}

	start := 0
	for {
		index := strings.Index(lower[start:], needle)
		if index < 0 {
			return ""
		}
		index += start
		beforeOK := index == 0 || !isServiceNameChar(rune(lower[index-1]))
		valueStart := index + len(needle)
		if !beforeOK || valueStart >= len(lower) || !isServiceNameChar(rune(lower[valueStart])) {
			start = valueStart
			continue
		}
		valueEnd := valueStart
		for valueEnd < len(lower) && isServiceNameChar(rune(lower[valueEnd])) {
			valueEnd++
		}
		return strings.TrimSpace(text[valueStart:valueEnd])
	}
}

func payloadMatchesAlertTransition(payload ErrorTrackingAlertPayload, transitions []string) bool {
	return slices.ContainsFunc(normalizeAlertTransitions(transitions), func(transition string) bool {
		return strings.EqualFold(strings.TrimSpace(payload.AlertTransition), transition)
	})
}

func normalizeAlertTransitions(transitions []string) []string {
	allowed := make([]string, 0, 2)
	for _, known := range []string{AlertTransitionTriggered, AlertTransitionRetriggered} {
		if slices.ContainsFunc(transitions, func(transition string) bool {
			return strings.EqualFold(strings.TrimSpace(transition), known)
		}) {
			allowed = append(allowed, known)
		}
	}
	if len(allowed) == 0 {
		return []string{AlertTransitionTriggered}
	}
	return allowed
}

func environmentFromAlert(payload ErrorTrackingAlertPayload) string {
	if env := firstTagValue(payload.AlertScope, "env"); env != "" {
		return env
	}
	return firstTagValue(payload.Tags, "env")
}

func isServiceNameChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.'
}
