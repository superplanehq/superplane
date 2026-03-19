package azure

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	sbEntityTypeQueue = "queue"
	sbEntityTypeTopic = "topic"

	sbReceivePollInterval  = 1 * time.Second
	sbReceiveRetryInterval = 5 * time.Second
)

type OnServiceBusMessageReceived struct{}

type OnServiceBusMessageReceivedConfiguration struct {
	NamespaceName    string `json:"namespaceName" mapstructure:"namespaceName"`
	EntityType       string `json:"entityType" mapstructure:"entityType"`
	QueueName        string `json:"queueName" mapstructure:"queueName"`
	TopicName        string `json:"topicName" mapstructure:"topicName"`
	SubscriptionName string `json:"subscriptionName" mapstructure:"subscriptionName"`
}

func (t *OnServiceBusMessageReceived) Name() string {
	return "azure.onServiceBusMessageReceived"
}

func (t *OnServiceBusMessageReceived) Label() string {
	return "On Service Bus Message Received"
}

func (t *OnServiceBusMessageReceived) Description() string {
	return "Triggers when a message is received from an Azure Service Bus queue or topic subscription"
}

func (t *OnServiceBusMessageReceived) Documentation() string {
	return `
The On Service Bus Message Received trigger polls an Azure Service Bus queue or topic subscription
and fires a workflow execution for each message received. Messages are consumed (deleted) after delivery.

## Use Cases

- **Queue processing**: Trigger workflows for each message published to a queue
- **Event-driven pipelines**: React to messages published to a topic subscription
- **Task distribution**: Fan out work to parallel workflow executions per message

## How It Works

1. SuperPlane polls the queue or topic subscription using long-polling (55-second timeout)
2. When a message arrives, it is received and deleted (receive-and-delete mode)
3. The message body and metadata are passed to the workflow as trigger event data
4. Polling resumes immediately for the next message

## Configuration

- **Namespace**: The Service Bus namespace (short name, not FQDN)
- **Entity Type**: Queue or topic subscription
- **Queue Name**: The queue to receive from (when entity type is queue)
- **Topic Name**: The topic containing the subscription (when entity type is topic)
- **Subscription Name**: The subscription to receive from (when entity type is topic)

## Event Data

- **body**: The raw message body
- **messageId**: The Service Bus message ID
- **contentType**: The content type of the message
- **namespaceName**: The Service Bus namespace
- **entityPath**: The queue name or topic/subscription path
- **properties**: Broker properties (MessageId, ContentType, etc.)

## Notes

- Uses receive-and-delete mode: messages are consumed atomically
- Works with Standard and Premium tier namespaces (no Event Grid required)
- Requires **Azure Service Bus Data Receiver** or **Data Owner** role
`
}

func (t *OnServiceBusMessageReceived) Icon() string  { return "azure" }
func (t *OnServiceBusMessageReceived) Color() string { return "blue" }

func (t *OnServiceBusMessageReceived) ExampleData() map[string]any {
	return map[string]any{
		"body":          `{"event": "order_created", "orderId": "12345"}`,
		"messageId":     "abc123",
		"contentType":   "application/json",
		"namespaceName": "my-namespace",
		"entityPath":    "my-queue",
		"properties": map[string]any{
			"MessageId":   "abc123",
			"ContentType": "application/json",
		},
	}
}

func (t *OnServiceBusMessageReceived) Configuration() []configuration.Field {
	// queueName dropdown: visible when entityType=queue and cascades from namespaceName
	queueNameField := serviceBusTriggerOptionalQueueNameField()
	queueNameField.Description = "The queue to receive messages from"
	queueNameField.VisibilityConditions = []configuration.VisibilityCondition{
		{Field: "entityType", Values: []string{sbEntityTypeQueue}},
	}

	return []configuration.Field{
		serviceBusNamespaceFieldStandalone(),
		{
			Name:     "entityType",
			Label:    "Entity Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  sbEntityTypeQueue,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Queue", Value: sbEntityTypeQueue},
						{Label: "Topic Subscription", Value: sbEntityTypeTopic},
					},
				},
			},
		},
		queueNameField,
		serviceBusTriggerOptionalTopicNameField(),
		serviceBusTriggerSubscriptionNameField(),
	}
}

func (t *OnServiceBusMessageReceived) Setup(ctx core.TriggerContext) error {
	config := OnServiceBusMessageReceivedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateOnServiceBusMessageReceivedConfig(config); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, sbReceivePollInterval)
}

func (t *OnServiceBusMessageReceived) Actions() []core.Action {
	return []core.Action{
		{Name: "poll", Description: "Poll for a new Service Bus message"},
	}
}

func (t *OnServiceBusMessageReceived) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	config := OnServiceBusMessageReceivedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return nil, scheduleRetry(ctx, fmt.Errorf("failed to decode configuration: %w", err))
	}

	entityPath, err := buildEntityPath(config)
	if err != nil {
		return nil, scheduleRetry(ctx, err)
	}

	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return nil, scheduleRetry(ctx, fmt.Errorf("Azure provider not available: %w", err))
	}

	namespaceName := azureResourceName(config.NamespaceName)
	sbClient := newServiceBusDataClient(provider.getClient(), namespaceName)

	pollCtx, cancel := context.WithTimeout(context.Background(), 65*time.Second)
	defer cancel()

	msg, err := sbClient.receiveMessage(pollCtx, entityPath)
	if err != nil {
		return nil, scheduleRetry(ctx, fmt.Errorf("failed to receive message: %w", err))
	}

	// No message available — poll again immediately.
	if msg == nil {
		return nil, ctx.Requests.ScheduleActionCall("poll", map[string]any{}, sbReceivePollInterval)
	}

	ctx.Logger.Infof("Received Service Bus message: id=%s entity=%s", msg.MessageID, entityPath)

	event := map[string]any{
		"body":          msg.Body,
		"messageId":     msg.MessageID,
		"contentType":   msg.ContentType,
		"namespaceName": namespaceName,
		"entityPath":    entityPath,
		"properties":    msg.Properties,
	}

	if err := ctx.Events.Emit("azure.servicebus.message.received", event); err != nil {
		return nil, scheduleRetry(ctx, fmt.Errorf("failed to emit event: %w", err))
	}

	return nil, ctx.Requests.ScheduleActionCall("poll", map[string]any{}, sbReceivePollInterval)
}

func (t *OnServiceBusMessageReceived) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	return nil
}

func (t *OnServiceBusMessageReceived) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (t *OnServiceBusMessageReceived) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func validateOnServiceBusMessageReceivedConfig(config OnServiceBusMessageReceivedConfiguration) error {
	if config.NamespaceName == "" {
		return fmt.Errorf("namespace name is required")
	}

	switch config.EntityType {
	case sbEntityTypeQueue:
		if config.QueueName == "" {
			return fmt.Errorf("queue name is required")
		}
	case sbEntityTypeTopic:
		if config.TopicName == "" {
			return fmt.Errorf("topic name is required")
		}
		if config.SubscriptionName == "" {
			return fmt.Errorf("subscription name is required")
		}
	default:
		return fmt.Errorf("entity type must be %q or %q", sbEntityTypeQueue, sbEntityTypeTopic)
	}

	return nil
}

func buildEntityPath(config OnServiceBusMessageReceivedConfiguration) (string, error) {
	switch config.EntityType {
	case sbEntityTypeQueue:
		return config.QueueName, nil
	case sbEntityTypeTopic:
		return fmt.Sprintf("%s/subscriptions/%s", config.TopicName, config.SubscriptionName), nil
	default:
		return "", fmt.Errorf("unknown entity type: %s", config.EntityType)
	}
}

func scheduleRetry(ctx core.TriggerActionContext, err error) error {
	ctx.Logger.Warnf("Service Bus poll error (retrying in %s): %v", sbReceiveRetryInterval, err)
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, sbReceiveRetryInterval)
}
