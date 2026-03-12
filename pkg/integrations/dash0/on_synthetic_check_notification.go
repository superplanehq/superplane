package dash0

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnSyntheticCheckNotification struct{}

type OnSyntheticCheckNotificationMetadata struct {
	SubscriptionID string `json:"subscriptionId,omitempty" mapstructure:"subscriptionId"`
}

type OnSyntheticCheckNotificationConfiguration struct {
	Statuses []string `json:"statuses" mapstructure:"statuses"`
}

func (t *OnSyntheticCheckNotification) Name() string {
	return "dash0.onSyntheticCheckNotification"
}

func (t *OnSyntheticCheckNotification) Label() string {
	return "On Synthetic Check Notification"
}

func (t *OnSyntheticCheckNotification) Description() string {
	return "Listen to Dash0 synthetic check notification webhook events"
}

func (t *OnSyntheticCheckNotification) Documentation() string {
	return `The On Synthetic Check Notification trigger starts a workflow execution when Dash0 sends a synthetic check notification webhook.

## Setup

1. Configure the Dash0 integration in SuperPlane.
2. Copy the webhook URL shown in the integration configuration.
3. In Dash0, configure synthetic check notifications to send HTTP POST requests to that URL.

## Event Data

The trigger emits the full JSON payload received from Dash0 as ` + "`dash0.syntheticCheckNotification`" + `.

## Labels Format

Synthetic check notifications use a tuple-based label format where each label is an array of ` + "`[index, {key, value}]`" + `.
The trigger normalizes these labels into a flat ` + "`{key: value}`" + ` map in the emitted payload for easier downstream consumption.`
}

func (t *OnSyntheticCheckNotification) Icon() string {
	return "dash0"
}

func (t *OnSyntheticCheckNotification) Color() string {
	return "gray"
}

func (t *OnSyntheticCheckNotification) ExampleData() map[string]any {
	return onSyntheticCheckNotificationExampleData()
}

func (t *OnSyntheticCheckNotification) Configuration() []configuration.Field {
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

func (t *OnSyntheticCheckNotification) Setup(ctx core.TriggerContext) error {
	metadata := OnSyntheticCheckNotificationMetadata{}
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

func (t *OnSyntheticCheckNotification) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnSyntheticCheckNotification) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnSyntheticCheckNotification) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// no-op, since events are received through the integration
	// and routed to OnIntegrationMessage()
	return http.StatusOK, nil
}

type SyntheticCheckNotificationEvent struct {
	Type string                         `json:"type"`
	Data SyntheticCheckNotificationData `json:"data"`
}

type SyntheticCheckNotificationData struct {
	Issue *SyntheticCheckNotificationIssue `json:"issue"`
}

// SyntheticCheckNotificationIssue represents the issue in a synthetic check notification.
type SyntheticCheckNotificationIssue struct {
	ID              string                                `json:"id"`
	IssueIdentifier string                                `json:"issueIdentifier"`
	Start           string                                `json:"start"`
	End             string                                `json:"end"`
	Status          string                                `json:"status"`
	Summary         string                                `json:"summary"`
	URL             string                                `json:"url"`
	Dataset         string                                `json:"dataset"`
	Description     string                                `json:"description"`
	CheckRules      []SyntheticCheckNotificationCheckRule `json:"checkrules"`
	Labels          []any                                 `json:"labels" mapstructure:"labels"`
}

type SyntheticCheckNotificationCheckRule struct {
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

func (t *OnSyntheticCheckNotification) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	config := OnSyntheticCheckNotificationConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	ctx.Logger.Infof("Received synthetic check notification event: %+v", ctx.Message)

	event := SyntheticCheckNotificationEvent{}
	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return fmt.Errorf("failed to decode synthetic check notification event: %w", err)
	}

	if event.Type == "test" {
		ctx.Logger.Info("Ignoring test synthetic check notification event")
		return nil
	}
	if event.Type != "synthetic.alert.ongoing" {
		ctx.Logger.Infof("Ignoring unsupported notification event type %s", event.Type)
		return nil
	}

	if event.Data.Issue == nil {
		ctx.Logger.Info("Ignoring synthetic check notification event without issue")
		return nil
	}

	issue := event.Data.Issue
	if !slices.Contains(config.Statuses, issue.Status) {
		ctx.Logger.Infof("Ignoring synthetic check notification event with status %s", issue.Status)
		return nil
	}

	return ctx.Events.Emit("dash0.syntheticCheckNotification", event.Data)
}

func (t *OnSyntheticCheckNotification) Cleanup(ctx core.TriggerContext) error {
	return nil
}
