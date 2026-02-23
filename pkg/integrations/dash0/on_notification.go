package dash0

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnNotification struct{}

type OnNotificationMetadata struct {
	SubscriptionID string `json:"subscriptionId,omitempty" mapstructure:"subscriptionId"`
}

type OnNotificationConfiguration struct {
	Statuses []string `json:"statuses" mapstructure:"statuses"`
}

func (t *OnNotification) Name() string {
	return "dash0.onNotification"
}

func (t *OnNotification) Label() string {
	return "On Notification"
}

func (t *OnNotification) Description() string {
	return "Listen to Dash0 notification webhook events"
}

func (t *OnNotification) Documentation() string {
	return `The On Notification trigger starts a workflow execution when Dash0 sends a notification webhook.

## Setup

1. Configure the Dash0 integration in SuperPlane.
2. Copy the webhook URL shown in the integration configuration.
3. In Dash0, configure notifications to send HTTP POST requests to that URL.

## Event Data

The trigger emits the full JSON payload received from Dash0 as ` + "`dash0.notification`" + `.`
}

func (t *OnNotification) Icon() string {
	return "dash0"
}

func (t *OnNotification) Color() string {
	return "gray"
}

func (t *OnNotification) ExampleData() map[string]any {
	return onNotificationExampleData()
}

func (t *OnNotification) Configuration() []configuration.Field {
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

func (t *OnNotification) Setup(ctx core.TriggerContext) error {
	metadata := OnNotificationMetadata{}
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

func (t *OnNotification) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnNotification) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnNotification) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// no-op, since events are received through the integration
	// and routed to OnIntegrationMessage()
	return http.StatusOK, nil
}

type NotificationEvent struct {
	Type string           `json:"type"`
	Data NotificationData `json:"data"`
}

type NotificationData struct {
	Issue *NotificationIssue `json:"issue"`
}

type NotificationIssue struct {
	ID              string                   `json:"id"`
	IssueIdentifier string                   `json:"issueIdentifier"`
	Start           string                   `json:"start"`
	End             string                   `json:"end"`
	Status          string                   `json:"status"`
	Summary         string                   `json:"summary"`
	URL             string                   `json:"url"`
	Dataset         string                   `json:"dataset"`
	Description     string                   `json:"description"`
	CheckRules      []NotificationCheckRule  `json:"checkrules"`
	Labels          []NotificationIssueLabel `json:"labels"`
}

type NotificationIssueLabel struct {
	Key   string                      `json:"key"`
	Value NotificationIssueLabelValue `json:"value"`
}

type NotificationIssueLabelValue struct {
	StringValue string `json:"stringValue"`
}

type NotificationCheckRule struct {
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

func (t *OnNotification) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	config := OnNotificationConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	ctx.Logger.Infof("Received notification event: %+v", ctx.Message)

	event := NotificationEvent{}
	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return fmt.Errorf("failed to decode notification event: %w", err)
	}

	if event.Type == "test" {
		ctx.Logger.Info("Ignoring test notification event")
		return nil
	}

	if event.Data.Issue == nil {
		ctx.Logger.Info("Ignoring notification event without issue")
		return nil
	}

	issue := event.Data.Issue
	if !slices.Contains(config.Statuses, issue.Status) {
		ctx.Logger.Infof("Ignoring notification event with status %s", issue.Status)
		return nil
	}

	return ctx.Events.Emit("dash0.notification", event.Data)
}

func (t *OnNotification) Cleanup(ctx core.TriggerContext) error {
	return nil
}
