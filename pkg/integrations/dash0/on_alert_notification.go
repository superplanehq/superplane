package dash0

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnAlertNotification struct{}

type OnAlertNotificationMetadata struct {
	SubscriptionID string `json:"subscriptionId,omitempty" mapstructure:"subscriptionId"`
}

type OnAlertNotificationConfiguration struct {
	Statuses []string `json:"statuses" mapstructure:"statuses"`
}

func (t *OnAlertNotification) Name() string {
	return "dash0.onAlertNotification"
}

func (t *OnAlertNotification) Label() string {
	return "On Alert Notification"
}

func (t *OnAlertNotification) Description() string {
	return "Listen to Dash0 alert notification webhook events"
}

func (t *OnAlertNotification) Documentation() string {
	return `The On Alert Notification trigger starts a workflow execution when Dash0 sends an alert notification webhook.

## Setup

1. Configure the Dash0 integration in SuperPlane.
2. Copy the webhook URL shown in the integration configuration.
3. In Dash0, configure alert notifications to send HTTP POST requests to that URL.

## Event Data

The trigger emits the full JSON payload received from Dash0 as ` + "`dash0.alertNotification`" + `.`
}

func (t *OnAlertNotification) Icon() string {
	return "dash0"
}

func (t *OnAlertNotification) Color() string {
	return "gray"
}

func (t *OnAlertNotification) ExampleData() map[string]any {
	return onAlertNotificationExampleData()
}

func (t *OnAlertNotification) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "statuses",
			Label:    "Statuses",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"critical", "degraded"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Critical", Value: "critical"},
						{Label: "Degraded", Value: "degraded"},
						{Label: "Closed", Value: "closed"},
					},
				},
			},
		},
	}
}

func (t *OnAlertNotification) Setup(ctx core.TriggerContext) error {
	metadata := OnAlertNotificationMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.SubscriptionID != "" {
		return nil
	}

	//
	// NOTE: we don't include anything in the subscription itself for now.
	// All the filters are applied as part of OnIntegrationMessage().
	//
	subscriptionID, err := ctx.Integration.Subscribe(SubscriptionConfiguration{})
	if err != nil {
		return fmt.Errorf("failed to subscribe to dash0 notifications: %w", err)
	}

	metadata.SubscriptionID = subscriptionID.String()
	return ctx.Metadata.Set(metadata)
}

func (t *OnAlertNotification) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnAlertNotification) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnAlertNotification) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// no-op, since events are received through the integration
	// and routed to OnIntegrationMessage()
	return http.StatusOK, nil
}

type AlertNotificationEvent struct {
	Type string                `json:"type"`
	Data AlertNotificationData `json:"data"`
}

type AlertNotificationData struct {
	Issue *AlertNotificationIssue `json:"issue"`
}

type AlertNotificationIssue struct {
	ID              string                        `json:"id"`
	IssueIdentifier string                        `json:"issueIdentifier"`
	Start           string                        `json:"start"`
	End             string                        `json:"end"`
	Status          string                        `json:"status"`
	Summary         string                        `json:"summary"`
	URL             string                        `json:"url"`
	Dataset         string                        `json:"dataset"`
	Description     string                        `json:"description"`
	CheckRules      []AlertNotificationCheckRule  `json:"checkrules"`
	Labels          []AlertNotificationIssueLabel `json:"labels"`
}

type AlertNotificationIssueLabel struct {
	Key   string                           `json:"key"`
	Value AlertNotificationIssueLabelValue `json:"value"`
}

type AlertNotificationIssueLabelValue struct {
	StringValue string `json:"stringValue"`
}

type AlertNotificationCheckRule struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	For           string         `json:"for"`
	KeepFiringFor string         `json:"keepFiringFor"`
	Interval      string         `json:"interval"`
	Description   string         `json:"description"`
	URL           string         `json:"url"`
	Expression    string         `json:"expression"`
	Annotations   map[string]any `json:"annotations"`
	Labels        map[string]any `json:"labels"`
	Thresholds    map[string]any `json:"thresholds"`
}

func (t *OnAlertNotification) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	config := OnAlertNotificationConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	ctx.Logger.Infof("Received alert notification event: %+v", ctx.Message)

	event := AlertNotificationEvent{}
	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return fmt.Errorf("failed to decode alert notification event: %w", err)
	}

	if event.Type == "test" {
		ctx.Logger.Info("Ignoring test alert notification event")
		return nil
	}
	if event.Type != "alert.ongoing" {
		ctx.Logger.Infof("Ignoring unsupported notification event type %s", event.Type)
		return nil
	}

	if event.Data.Issue == nil {
		ctx.Logger.Info("Ignoring alert notification event without issue")
		return nil
	}

	issue := event.Data.Issue
	if !slices.Contains(config.Statuses, issue.Status) {
		ctx.Logger.Infof("Ignoring alert notification event with status %s", issue.Status)
		return nil
	}

	return ctx.Events.Emit("dash0.alertNotification", event.Data)
}

func (t *OnAlertNotification) Cleanup(ctx core.TriggerContext) error {
	return nil
}
