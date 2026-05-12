package azure

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// OnServiceBusMessageAvailable fires when messages accumulate in a Service Bus queue or topic
// with no active consumers. Requires a Premium tier Service Bus namespace.
type OnServiceBusMessageAvailable struct {
}

type OnServiceBusMessageAvailableConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	NamespaceName string `json:"namespaceName" mapstructure:"namespaceName"`
	// QueueName filters to a specific queue. Leave empty to trigger for all queues in the namespace.
	QueueName string `json:"queueName" mapstructure:"queueName"`
}

func (t *OnServiceBusMessageAvailable) Name() string {
	return "azure.onServiceBusMessageAvailable"
}

func (t *OnServiceBusMessageAvailable) Label() string {
	return "On Service Bus Message Available"
}

func (t *OnServiceBusMessageAvailable) Description() string {
	return "Triggers when messages are available in a Service Bus queue with no active consumers"
}

func (t *OnServiceBusMessageAvailable) Documentation() string {
	return `
The On Service Bus Message Available trigger fires when messages are present in a Service Bus queue
but no active consumers are receiving them. This is useful for autoscaling, alerting, and
orchestrating workflows that should start when work accumulates.

## Use Cases

- **Autoscaling triggers**: Start worker VMs or containers when a queue depth increases
- **Alerting workflows**: Notify on-call teams when messages pile up with no consumers
- **Orchestration**: Kick off processing pipelines when new work arrives
- **Dead-letter monitoring**: React when messages accumulate with no listeners

## How It Works

This trigger listens to Azure Event Grid events of type
` + "`Microsoft.ServiceBus.ActiveMessagesAvailableWithNoListeners`" + `.

Azure fires this event when:
1. One or more messages arrive in the queue, AND
2. There are no active receivers consuming from the queue

The trigger provides metadata about which queue has messages waiting, but **does not deliver
the actual message content**. Use the "Send Service Bus Message" action or other steps in
your workflow to process the messages.

> **Note**: This trigger requires an Azure Service Bus **Premium tier** namespace.
> Standard and Basic tier namespaces do not support Event Grid integration.

## Configuration

- **Resource Group**: Filter events to a specific resource group (optional)
- **Namespace**: The Service Bus namespace to monitor (required)
- **Queue Name** (optional): Filter to a specific queue. Leave empty to trigger for all queues in the namespace.

## Event Data

Each trigger event includes the full Azure Event Grid event payload:

- **id**: Unique event ID
- **topic**: The namespace ARM resource ID
- **subject**: The entity path (e.g., ` + "`queues/my-queue`" + `)
- **eventType**: ` + "`Microsoft.ServiceBus.ActiveMessagesAvailableWithNoListeners`" + `
- **eventTime**: When the event occurred
- **data**:
  - **namespaceName**: Fully-qualified namespace hostname (e.g., ` + "`myns.servicebus.windows.net`" + `)
  - **entityType**: Always ` + "`\"queue\"`" + ` for this trigger
  - **queueName**: The name of the queue with pending messages
  - **requestUri**: The URI to the queue messages endpoint

## Required Azure Permissions

- **Azure Service Bus Data Owner** or **Azure Service Bus Data Receiver** on the namespace
- **EventGrid Contributor** for Event Grid subscription management
`
}

func (t *OnServiceBusMessageAvailable) Icon() string {
	return "azure"
}

func (t *OnServiceBusMessageAvailable) Color() string {
	return "blue"
}

func (t *OnServiceBusMessageAvailable) ExampleData() map[string]any {
	return map[string]any{
		"id":              "96257b6d-17d3-49e2-8369-fb185b29e1b5",
		"topic":           "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.ServiceBus/namespaces/my-ns",
		"subject":         "queues/my-queue",
		"eventType":       "Microsoft.ServiceBus.ActiveMessagesAvailableWithNoListeners",
		"eventTime":       "2026-03-16T10:00:00Z",
		"dataVersion":     "1",
		"metadataVersion": "1",
		"data": map[string]any{
			"namespaceName":   "my-ns.servicebus.windows.net",
			"requestUri":      "https://my-ns.servicebus.windows.net/my-queue/messages/head",
			"entityType":      "queue",
			"queueName":       "my-queue",
			"deadLetterQueue": false,
		},
	}
}

func (t *OnServiceBusMessageAvailable) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "resourceGroup",
			Label:       "Resource Group",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Filter events to a specific resource group (optional – leave empty for all resource groups)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeResourceGroupDropdown,
					UseNameAsValue: true,
				},
			},
		},
		serviceBusTriggerNamespaceField(),
		serviceBusTriggerOptionalQueueNameField(),
	}
}

func (t *OnServiceBusMessageAvailable) Setup(ctx core.TriggerContext) error {
	config := OnServiceBusMessageAvailableConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.NamespaceName == "" {
		return fmt.Errorf("namespace name is required")
	}

	namespaceName := azureResourceName(config.NamespaceName)

	err := ctx.Integration.RequestWebhook(AzureWebhookConfiguration{
		EventTypes:    []string{EventTypeServiceBusActiveMessages, EventTypeServiceBusActiveMessagesPeriodic},
		ResourceGroup: config.ResourceGroup,
		TopicContains: fmt.Sprintf("/providers/Microsoft.ServiceBus/namespaces/%s", namespaceName),
	})
	if err != nil {
		return fmt.Errorf("failed to request webhook: %w", err)
	}

	ctx.Logger.Infof("Azure On Service Bus Message Available trigger configured for namespace: %s", namespaceName)
	if config.QueueName != "" {
		ctx.Logger.Infof("Filtering events for queue: %s", config.QueueName)
	}

	return nil
}

func (t *OnServiceBusMessageAvailable) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	if err := t.authenticateWebhook(ctx); err != nil {
		ctx.Logger.Warnf("Webhook authentication failed: %v", err)
		return http.StatusUnauthorized, nil, err
	}

	config := OnServiceBusMessageAvailableConfiguration{}
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

		if event.EventType == EventTypeServiceBusActiveMessages || event.EventType == EventTypeServiceBusActiveMessagesPeriodic {
			if err := t.handleMessageAvailableEvent(ctx, event, rawEvents[i], config); err != nil {
				ctx.Logger.Errorf("Failed to process Service Bus event: %v", err)
				continue
			}
		}
	}

	return http.StatusOK, nil, nil
}

func (t *OnServiceBusMessageAvailable) handleMessageAvailableEvent(
	ctx core.WebhookRequestContext,
	event EventGridEvent,
	rawEvent map[string]any,
	config OnServiceBusMessageAvailableConfiguration,
) error {
	var data ServiceBusMessageAvailableData
	if err := mapstructure.Decode(event.Data, &data); err != nil {
		return fmt.Errorf("failed to parse event data: %w", err)
	}

	// Filter by queue name if configured
	if config.QueueName != "" && !strings.EqualFold(data.QueueName, config.QueueName) {
		ctx.Logger.Debugf("Skipping event for queue %s (filter: %s)", data.QueueName, config.QueueName)
		return nil
	}

	ctx.Logger.Infof("Service Bus message available: namespace=%s queue=%s", data.NamespaceName, data.QueueName)

	if err := ctx.Events.Emit("azure.servicebus.message_available", rawEvent); err != nil {
		return fmt.Errorf("failed to emit event: %w", err)
	}

	return nil
}

func (t *OnServiceBusMessageAvailable) authenticateWebhook(ctx core.WebhookRequestContext) error {
	return authenticateServiceBusWebhook(ctx)
}

func (t *OnServiceBusMessageAvailable) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnServiceBusMessageAvailable) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnServiceBusMessageAvailable) Cleanup(ctx core.TriggerContext) error {
	ctx.Logger.Info("Cleaning up Azure On Service Bus Message Available trigger")
	return nil
}

// handleServiceBusSubscriptionValidation handles the Event Grid subscription validation handshake.
func handleServiceBusSubscriptionValidation(ctx core.WebhookRequestContext, event EventGridEvent) (*core.WebhookResponseBody, error) {
	var validationData SubscriptionValidationEventData
	if err := mapstructure.Decode(event.Data, &validationData); err != nil {
		return nil, fmt.Errorf("failed to parse validation data: %w", err)
	}

	if validationData.ValidationCode == "" {
		return nil, fmt.Errorf("validation code is empty")
	}

	ctx.Logger.Info("Event Grid subscription validation received")

	body, err := json.Marshal(map[string]string{
		"validationResponse": validationData.ValidationCode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal validation response: %w", err)
	}

	return &core.WebhookResponseBody{Body: body, ContentType: "application/json"}, nil
}

// authenticateServiceBusWebhook verifies the webhook secret if one is configured.
func authenticateServiceBusWebhook(ctx core.WebhookRequestContext) error {
	if ctx.Webhook == nil {
		return nil
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		ctx.Logger.Debugf("Could not retrieve webhook secret: %v", err)
		return nil
	}

	if len(secret) == 0 {
		return nil
	}

	authHeader := ctx.Headers.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		providedSecret := strings.TrimPrefix(authHeader, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(providedSecret), secret) == 1 {
			return nil
		}
		return fmt.Errorf("invalid webhook secret")
	}

	secretHeader := ctx.Headers.Get("X-Webhook-Secret")
	if secretHeader != "" {
		if subtle.ConstantTimeCompare([]byte(secretHeader), secret) == 1 {
			return nil
		}
		return fmt.Errorf("invalid webhook secret")
	}

	return fmt.Errorf("webhook secret required but not provided in Authorization or X-Webhook-Secret header")
}
