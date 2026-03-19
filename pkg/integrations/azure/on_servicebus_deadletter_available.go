package azure

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// OnServiceBusDeadLetterAvailable fires when dead-letter messages accumulate in a Service Bus
// queue or topic subscription with no active consumers. Requires a Premium tier namespace.
type OnServiceBusDeadLetterAvailable struct {
}

type OnServiceBusDeadLetterAvailableConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	NamespaceName string `json:"namespaceName" mapstructure:"namespaceName"`
	// QueueName filters to a specific queue's dead-letter sub-queue.
	QueueName string `json:"queueName" mapstructure:"queueName"`
}

func (t *OnServiceBusDeadLetterAvailable) Name() string {
	return "azure.onServiceBusDeadLetterAvailable"
}

func (t *OnServiceBusDeadLetterAvailable) Label() string {
	return "On Service Bus Dead-Letter Available"
}

func (t *OnServiceBusDeadLetterAvailable) Description() string {
	return "Triggers when dead-letter messages are available in a Service Bus queue with no active consumers"
}

func (t *OnServiceBusDeadLetterAvailable) Documentation() string {
	return `
The On Service Bus Dead-Letter Available trigger fires when dead-letter messages are present in a
Service Bus queue's dead-letter sub-queue but no active consumers are processing them.

## Use Cases

- **Dead-letter monitoring**: Alert teams when messages fail processing and enter the dead-letter queue
- **Automated remediation**: Trigger reprocessing workflows when dead-letter messages accumulate
- **Compliance**: Ensure dead-letter messages are reviewed and handled within SLA
- **Alerting**: Notify when message processing failures spike

## How It Works

This trigger listens to Azure Event Grid events of type
` + "`Microsoft.ServiceBus.DeadletterMessagesAvailableWithNoListeners`" + `.

Azure fires this event when:
1. One or more messages are in the dead-letter sub-queue, AND
2. There are no active receivers consuming from the dead-letter sub-queue

The trigger provides metadata about which queue has dead-letter messages waiting, but **does not
deliver the actual message content**.

> **Note**: This trigger requires an Azure Service Bus **Premium tier** namespace.
> Standard and Basic tier namespaces do not support Event Grid integration.

## Configuration

- **Resource Group**: Filter events to a specific resource group (optional)
- **Namespace**: The Service Bus namespace to monitor (required)
- **Queue Name** (optional): Filter to a specific queue's dead-letter sub-queue. Leave empty for all queues.

## Event Data

Each trigger event includes the full Azure Event Grid event payload:

- **id**: Unique event ID
- **topic**: The namespace ARM resource ID
- **subject**: The entity path (e.g., ` + "`queues/my-queue`" + `)
- **eventType**: ` + "`Microsoft.ServiceBus.DeadletterMessagesAvailableWithNoListeners`" + `
- **eventTime**: When the event occurred
- **data**:
  - **namespaceName**: Fully-qualified namespace hostname
  - **entityType**: ` + "`\"queue\"`" + ` or ` + "`\"topic\"`" + `
  - **queueName**: The name of the queue with dead-letter messages
  - **deadLetterQueue**: Always ` + "`true`" + ` for this trigger

## Required Azure Permissions

- **Azure Service Bus Data Owner** or **Azure Service Bus Data Receiver** on the namespace
- **EventGrid Contributor** for Event Grid subscription management
`
}

func (t *OnServiceBusDeadLetterAvailable) Icon() string {
	return "azure"
}

func (t *OnServiceBusDeadLetterAvailable) Color() string {
	return "blue"
}

func (t *OnServiceBusDeadLetterAvailable) ExampleData() map[string]any {
	return map[string]any{
		"id":              "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		"topic":           "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.ServiceBus/namespaces/my-ns",
		"subject":         "queues/my-queue",
		"eventType":       "Microsoft.ServiceBus.DeadletterMessagesAvailableWithNoListeners",
		"eventTime":       "2026-03-16T10:00:00Z",
		"dataVersion":     "1",
		"metadataVersion": "1",
		"data": map[string]any{
			"namespaceName":   "my-ns.servicebus.windows.net",
			"requestUri":      "https://my-ns.servicebus.windows.net/my-queue/$deadletterqueue/messages/head",
			"entityType":      "queue",
			"queueName":       "my-queue",
			"deadLetterQueue": true,
		},
	}
}

func (t *OnServiceBusDeadLetterAvailable) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "resourceGroup",
			Label:       "Resource Group",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Filter events to a specific resource group (optional - leave empty for all resource groups)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeResourceGroupDropdown,
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "namespaceName",
			Label:       "Namespace",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The Service Bus namespace to monitor (short name, not FQDN)",
			Placeholder: "my-namespace",
		},
		{
			Name:        "queueName",
			Label:       "Queue Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter to a specific queue's dead-letter sub-queue (optional - leave empty for all queues)",
			Placeholder: "my-queue",
		},
	}
}

func (t *OnServiceBusDeadLetterAvailable) Setup(ctx core.TriggerContext) error {
	config := OnServiceBusDeadLetterAvailableConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.NamespaceName == "" {
		return fmt.Errorf("namespace name is required")
	}

	namespaceName := azureResourceName(config.NamespaceName)

	err := ctx.Integration.RequestWebhook(AzureWebhookConfiguration{
		EventTypes:    []string{EventTypeServiceBusDeadLetterMessages},
		ResourceGroup: config.ResourceGroup,
		TopicContains: fmt.Sprintf("/providers/Microsoft.ServiceBus/namespaces/%s", namespaceName),
	})
	if err != nil {
		return fmt.Errorf("failed to request webhook: %w", err)
	}

	ctx.Logger.Infof("Azure On Service Bus Dead-Letter Available trigger configured for namespace: %s", namespaceName)
	if config.QueueName != "" {
		ctx.Logger.Infof("Filtering events for queue: %s", config.QueueName)
	}

	return nil
}

func (t *OnServiceBusDeadLetterAvailable) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	if err := authenticateServiceBusWebhook(ctx); err != nil {
		ctx.Logger.Warnf("Webhook authentication failed: %v", err)
		return http.StatusUnauthorized, nil, err
	}

	config := OnServiceBusDeadLetterAvailableConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	var events []EventGridEvent
	if err := json.Unmarshal(ctx.Body, &events); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("failed to parse events: %w", err)
	}

	var rawEvents []map[string]any
	if err := json.Unmarshal(ctx.Body, &rawEvents); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("failed to parse raw events: %w", err)
	}

	ctx.Logger.Infof("Received %d Event Grid event(s)", len(events))

	for i, event := range events {
		ctx.Logger.Infof("Event[%d]: id=%s type=%s subject=%s", i, event.ID, event.EventType, event.Subject)

		if event.EventType == EventTypeSubscriptionValidation {
			resp, err := handleServiceBusSubscriptionValidation(ctx, event)
			if err != nil {
				return http.StatusInternalServerError, nil, err
			}
			return http.StatusOK, resp, nil
		}

		if event.EventType == EventTypeServiceBusDeadLetterMessages || event.EventType == EventTypeServiceBusDeadLetterMessagesPeriodic {
			if err := t.handleDeadLetterEvent(ctx, event, rawEvents[i], config); err != nil {
				ctx.Logger.Errorf("Failed to process Service Bus dead-letter event: %v", err)
				continue
			}
		}
	}

	return http.StatusOK, nil, nil
}

func (t *OnServiceBusDeadLetterAvailable) handleDeadLetterEvent(
	ctx core.WebhookRequestContext,
	event EventGridEvent,
	rawEvent map[string]any,
	config OnServiceBusDeadLetterAvailableConfiguration,
) error {
	var data ServiceBusMessageAvailableData
	if err := mapstructure.Decode(event.Data, &data); err != nil {
		return fmt.Errorf("failed to parse event data: %w", err)
	}

	// Filter by queue name if configured
	if config.QueueName != "" && !strings.EqualFold(data.QueueName, config.QueueName) {
		ctx.Logger.Debugf("Skipping dead-letter event for queue %s (filter: %s)", data.QueueName, config.QueueName)
		return nil
	}

	ctx.Logger.Infof("Service Bus dead-letter messages available: namespace=%s queue=%s", data.NamespaceName, data.QueueName)

	if err := ctx.Events.Emit("azure.servicebus.deadletter_available", rawEvent); err != nil {
		return fmt.Errorf("failed to emit event: %w", err)
	}

	return nil
}

func (t *OnServiceBusDeadLetterAvailable) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnServiceBusDeadLetterAvailable) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnServiceBusDeadLetterAvailable) Cleanup(ctx core.TriggerContext) error {
	ctx.Logger.Info("Cleaning up Azure On Service Bus Dead-Letter Available trigger")
	return nil
}
