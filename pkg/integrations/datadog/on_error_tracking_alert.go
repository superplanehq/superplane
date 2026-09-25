package datadog

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ErrorTrackingAlertPayloadType = "datadog.errorTrackingAlert"

type OnErrorTrackingAlert struct{}

type OnErrorTrackingAlertMetadata struct {
	SubscriptionID string `json:"subscriptionId,omitempty" mapstructure:"subscriptionId"`
}

type ErrorTrackingAlertPayload struct {
	ID              string `json:"id" mapstructure:"id"`
	EventType       string `json:"event_type" mapstructure:"event_type"`
	Title           string `json:"title" mapstructure:"title"`
	Body            string `json:"body" mapstructure:"body"`
	AlertID         string `json:"alert_id" mapstructure:"alert_id"`
	AlertTransition string `json:"alert_transition" mapstructure:"alert_transition"`
	Tags            string `json:"tags" mapstructure:"tags"`
	Link            string `json:"link" mapstructure:"link"`
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

## Event Data

The trigger emits:
- **title**: monitor alert title
- **body**: monitor notification text
- **link**: Datadog link from the alert
- **alert_id**: alerting monitor ID
- **tags**: comma-separated tags from the alert
- **alert_transition**: alert transition, such as Triggered`
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
	return []configuration.Field{}
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
	payload, err := decodeErrorTrackingAlertPayload(ctx.Message)
	if err != nil {
		return err
	}

	if !strings.EqualFold(payload.EventType, ErrorTrackingAlertEventType) {
		return nil
	}

	if !strings.EqualFold(strings.TrimSpace(payload.AlertTransition), AlertTransitionTriggered) {
		return nil
	}

	return ctx.Events.Emit(ErrorTrackingAlertPayloadType, payload)
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
