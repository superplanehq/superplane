package pubsub

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

const EmittedEventType = "gcp.pubsub.topicMessage"

type OnTopicMessage struct{}

type OnTopicMessageConfiguration struct {
	Topic string `json:"topic" mapstructure:"topic"`
}

type OnTopicMessageMetadata struct {
	Topic          string `json:"topic" mapstructure:"topic"`
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
	PushSubID      string `json:"pushSubId" mapstructure:"pushSubId"`
}

func (t *OnTopicMessage) Name() string {
	return "gcp.pubsub.onTopicMessage"
}

func (t *OnTopicMessage) Label() string {
	return "Pub/Sub • On Topic Message"
}

func (t *OnTopicMessage) Description() string {
	return "Listen to messages published to a Google Cloud Pub/Sub topic"
}

func (t *OnTopicMessage) Documentation() string {
	return `The On Topic Message trigger starts a workflow execution when a message is published to a Google Cloud Pub/Sub topic.

## Use Cases

- **Event-driven automation**: React to messages published by other services
- **Data pipeline triggers**: Start workflows when new data arrives in a topic
- **Cross-service orchestration**: Coordinate workflows across microservices using Pub/Sub

## How it works

During setup, SuperPlane creates a push subscription on the specified topic. When messages are published to the topic, they are delivered to SuperPlane and trigger workflow executions.

## Setup

Ensure the Pub/Sub API is enabled and the integration's service account has ` + "`roles/pubsub.subscriber`" + ` and ` + "`roles/pubsub.viewer`" + ` permissions on the topic.`
}

func (t *OnTopicMessage) Icon() string {
	return "gcp"
}

func (t *OnTopicMessage) Color() string {
	return "gray"
}

func (t *OnTopicMessage) ExampleData() map[string]any {
	return map[string]any{
		"topic":     "my-topic",
		"messageId": "12345678901234",
		"data":      map[string]any{"key": "value"},
	}
}

func (t *OnTopicMessage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "topic",
			Label:       "Topic",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Pub/Sub topic ID to listen to (e.g. my-topic). The project is inferred from the integration.",
			Placeholder: "e.g. my-topic",
		},
	}
}

func (t *OnTopicMessage) Setup(ctx core.TriggerContext) error {
	if ctx.Integration == nil {
		return fmt.Errorf("connect the GCP integration to this trigger to enable Pub/Sub event delivery")
	}

	var config OnTopicMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	topic := strings.TrimSpace(config.Topic)
	if topic == "" {
		return fmt.Errorf("topic is required")
	}

	var metadata OnTopicMessageMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Topic == topic && metadata.SubscriptionID != "" && metadata.PushSubID != "" {
		return nil
	}

	subscriptionID, err := ctx.Integration.Subscribe(TopicSubscriptionPattern(topic))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	pushSubID := "sp-pubsub-" + sanitizeID(subscriptionID.String())

	if err := ctx.Metadata.Set(OnTopicMessageMetadata{
		Topic:          topic,
		SubscriptionID: subscriptionID.String(),
		PushSubID:      pushSubID,
	}); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return ctx.Requests.ScheduleActionCall("provisionPushSubscription", map[string]any{
		"topic":     topic,
		"pushSubId": pushSubID,
	}, 2*time.Second)
}

func (t *OnTopicMessage) Actions() []core.Action {
	return []core.Action{
		{Name: "provisionPushSubscription"},
	}
}

func (t *OnTopicMessage) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	if ctx.Name != "provisionPushSubscription" {
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}

	return t.provisionPushSubscription(ctx)
}

func (t *OnTopicMessage) provisionPushSubscription(ctx core.TriggerActionContext) (map[string]any, error) {
	meta, err := integrationMetadata(ctx.Integration)
	if err != nil {
		return nil, err
	}

	topic, _ := ctx.Parameters["topic"].(string)
	if topic == "" {
		return nil, fmt.Errorf("topic parameter is required")
	}

	pushSubID, _ := ctx.Parameters["pushSubId"].(string)
	if pushSubID == "" {
		return nil, fmt.Errorf("pushSubId parameter is required")
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("create GCP client: %w", err)
	}

	projectID := client.ProjectID()

	secret, err := eventsSecret(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("get events secret: %w", err)
	}

	baseURL := meta.EventsBaseURL
	if baseURL == "" {
		return nil, fmt.Errorf("events base URL not configured; re-sync the GCP integration")
	}

	pushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/events/pubsub?token=%s&topic=%s",
		baseURL, ctx.Integration.ID(), secret, topic)

	reqCtx := context.Background()
	if err := CreatePushSubscription(reqCtx, client, projectID, pushSubID, topic, pushEndpoint); err != nil {
		return nil, fmt.Errorf("create push subscription: %w", err)
	}

	return nil, nil
}

func (t *OnTopicMessage) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	return ctx.Events.Emit(EmittedEventType, ctx.Message)
}

func (t *OnTopicMessage) Cleanup(ctx core.TriggerContext) error {
	var metadata OnTopicMessageMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil || metadata.PushSubID == "" {
		return nil
	}

	if ctx.Integration == nil {
		return nil
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		ctx.Logger.Warnf("failed to create GCP client for push subscription cleanup: %v", err)
		return nil
	}

	if err := DeleteSubscription(context.Background(), client, client.ProjectID(), metadata.PushSubID); err != nil {
		if !gcpcommon.IsNotFoundError(err) {
			ctx.Logger.Warnf("failed to delete push subscription %s: %v", metadata.PushSubID, err)
		}
	}

	return nil
}

func (t *OnTopicMessage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func TopicSubscriptionPattern(topic string) map[string]any {
	return map[string]any{
		"type":  "pubsub.topic",
		"topic": topic,
	}
}

func integrationMetadata(integration core.IntegrationContext) (*gcpcommon.Metadata, error) {
	var m gcpcommon.Metadata
	if err := mapstructure.Decode(integration.GetMetadata(), &m); err != nil {
		return nil, fmt.Errorf("failed to read integration metadata: %w", err)
	}
	if m.ProjectID == "" {
		return nil, fmt.Errorf("integration metadata does not contain a project ID; re-sync the GCP integration")
	}
	return &m, nil
}

func eventsSecret(integration core.IntegrationContext) (string, error) {
	secrets, err := integration.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, s := range secrets {
		if s.Name == "pubsub.events.secret" {
			return string(s.Value), nil
		}
	}

	return "", fmt.Errorf("events secret not found; re-sync the GCP integration")
}

func sanitizeID(s string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		}
	}
	result := b.String()
	if len(result) > 60 {
		result = result[:60]
	}
	return result
}
