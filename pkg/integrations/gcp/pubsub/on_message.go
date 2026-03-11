package pubsub

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

const (
	OnMessageEmittedEventType = "gcp.pubsub.message"
	OnMessageSubscriptionType = "pubsub.onMessage"
)

type OnMessage struct{}

type OnMessageConfiguration struct {
	Topic        string `json:"topic" mapstructure:"topic"`
	Subscription string `json:"subscription" mapstructure:"subscription"`
}

type OnMessageMetadata struct {
	InternalSubscriptionID string `json:"internalSubscription"`
	Topic                  string `json:"topic"`
	GCPSubName             string `json:"gcpSubName"`
}

func (t *OnMessage) Name() string {
	return "gcp.pubsub.onMessage"
}

func (t *OnMessage) Label() string {
	return "Pub/Sub • On Message"
}

func (t *OnMessage) Description() string {
	return "Trigger a workflow when a message is published to a GCP Pub/Sub topic"
}

func (t *OnMessage) Documentation() string {
	return `The On Message trigger starts a workflow execution when a message is published to a GCP Pub/Sub topic.

**Trigger behavior:** SuperPlane creates a push subscription on the selected topic. Published messages are pushed to SuperPlane and delivered to this trigger.

## Use Cases

- **Event-driven workflows**: React to messages published by your applications
- **Queue processing**: Process tasks published to Pub/Sub topics
- **System integration**: Connect Pub/Sub events to downstream workflow steps

## Setup

**Required GCP setup:** Ensure the **Pub/Sub API** is enabled in your project. The service account used by the integration must have ` + "`roles/pubsub.admin`" + ` to create push subscriptions on your topics.

## Configuration

- **Topic**: Select the Pub/Sub topic to listen to.
- **Subscription (optional)**: Reuse an existing subscription name. Leave empty to let SuperPlane create one.

## Event Data

Each event contains the decoded message payload plus Pub/Sub metadata:

- ` + "`data`" + `: The decoded message body
- ` + "`messageId`" + `: The Pub/Sub message ID
- ` + "`publishTime`" + `: When the message was published
- ` + "`attributes`" + `: Any message attributes`
}

func (t *OnMessage) Icon() string  { return "gcp" }
func (t *OnMessage) Color() string { return "gray" }

func (t *OnMessage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "topic",
			Label:       "Topic",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the Pub/Sub topic to listen to.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeTopic,
					Parameters: []configuration.ParameterRef{},
				},
			},
		},
		{
			Name:        "subscription",
			Label:       "Subscription",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional existing subscription to reuse. If empty, SuperPlane creates one.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "topic", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeSubscription,
					Parameters: []configuration.ParameterRef{
						{Name: "topic", ValueFrom: &configuration.ParameterValueFrom{Field: "topic"}},
					},
				},
			},
		},
	}
}

func (t *OnMessage) Setup(ctx core.TriggerContext) error {
	var config OnMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode trigger configuration: %w", err)
	}

	config.Topic = strings.TrimSpace(config.Topic)
	config.Subscription = strings.TrimSpace(config.Subscription)
	if config.Topic == "" {
		return fmt.Errorf("topic is required")
	}

	if ctx.Integration == nil {
		return fmt.Errorf("connect the GCP integration to this trigger to enable Pub/Sub delivery")
	}

	var metadata OnMessageMetadata
	_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)

	if config.Subscription != "" {
		metadata.GCPSubName = config.Subscription
	} else if metadata.GCPSubName == "" {
		// Assign a stable GCP subscription name on first setup
		metadata.GCPSubName = "sp-pubsub-" + uuid.New().String()
	}
	metadata.Topic = config.Topic

	// Subscribe to integration event routing
	subscriptionID, err := ctx.Integration.Subscribe(map[string]any{
		"type":       OnMessageSubscriptionType,
		"topic":      config.Topic,
		"gcpSubName": metadata.GCPSubName,
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	metadata.InternalSubscriptionID = subscriptionID.String()

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	// Schedule integration action to create the GCP push subscription
	return ctx.Integration.ScheduleActionCall(gcpcommon.ActionNameEnsurePubSubOnMessage, map[string]any{
		"topic":      config.Topic,
		"gcpSubName": metadata.GCPSubName,
	}, 2*time.Second)
}

func (t *OnMessage) Actions() []core.Action { return nil }

func (t *OnMessage) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, fmt.Errorf("unknown action: %s", ctx.Name)
}

func (t *OnMessage) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	return ctx.Events.Emit(OnMessageEmittedEventType, ctx.Message)
}

func (t *OnMessage) Cleanup(ctx core.TriggerContext) error {
	if ctx.Integration == nil {
		return nil
	}

	var metadata OnMessageMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil || metadata.GCPSubName == "" {
		return nil
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil
	}

	_ = DeleteSubscription(context.Background(), client, client.ProjectID(), metadata.GCPSubName)
	return nil
}

func (t *OnMessage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
