package azure

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetServiceBusQueueComponent struct {
}

type GetServiceBusQueueConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	NamespaceName string `json:"namespaceName" mapstructure:"namespaceName"`
	QueueName     string `json:"queueName" mapstructure:"queueName"`
}

func (c *GetServiceBusQueueComponent) Name() string {
	return "azure.getServiceBusQueue"
}

func (c *GetServiceBusQueueComponent) Label() string {
	return "Get Service Bus Queue"
}

func (c *GetServiceBusQueueComponent) Description() string {
	return "Retrieves details and metrics for an Azure Service Bus queue"
}

func (c *GetServiceBusQueueComponent) Documentation() string {
	return `
The Get Service Bus Queue component retrieves details and metrics for an existing Azure Service Bus queue.

## Use Cases

- **Monitoring**: Check message counts and queue health in automated workflows
- **Conditional logic**: Branch workflows based on queue depth or status
- **Reporting**: Include queue metrics in notifications or dashboards

## How It Works

1. Validates the queue configuration parameters
2. Retrieves queue details via the Azure Service Bus ARM API
3. Returns the queue metadata and current message counts

## Configuration

- **Resource Group**: The Azure resource group containing the Service Bus namespace
- **Namespace**: The Service Bus namespace containing the queue
- **Queue Name**: The name of the queue to retrieve

## Output

Returns the queue information including:
- **id**: The Azure resource ID of the queue
- **name**: The name of the queue
- **namespaceName**: The Service Bus namespace
- **resourceGroup**: The resource group
- **properties**: Queue properties including message counts, size, lock duration, max delivery count, and status
`
}

func (c *GetServiceBusQueueComponent) Icon() string {
	return "azure"
}

func (c *GetServiceBusQueueComponent) Color() string {
	return "blue"
}

func (c *GetServiceBusQueueComponent) ExampleOutput() map[string]any {
	return map[string]any{
		"id":            "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.ServiceBus/namespaces/my-ns/queues/my-queue",
		"name":          "my-queue",
		"namespaceName": "my-ns",
		"resourceGroup": "my-rg",
		"properties": map[string]any{
			"messageCount":           42,
			"activeMessageCount":     40,
			"deadLetterMessageCount": 2,
			"maxSizeInMegabytes":     1024,
			"maxDeliveryCount":       10,
			"status":                 "Active",
		},
	}
}

func (c *GetServiceBusQueueComponent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetServiceBusQueueComponent) Configuration() []configuration.Field {
	return []configuration.Field{
		serviceBusResourceGroupField(),
		serviceBusNamespaceField(),
		serviceBusQueueNameField(),
	}
}

func (c *GetServiceBusQueueComponent) Setup(ctx core.SetupContext) error {
	config := GetServiceBusQueueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ResourceGroup == "" {
		return fmt.Errorf("resource group is required")
	}
	if config.NamespaceName == "" {
		return fmt.Errorf("namespace name is required")
	}
	if config.QueueName == "" {
		return fmt.Errorf("queue name is required")
	}

	return nil
}

func (c *GetServiceBusQueueComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetServiceBusQueueComponent) Execute(ctx core.ExecutionContext) error {
	config := GetServiceBusQueueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return fmt.Errorf("Azure provider not available: %w", err)
	}

	output, err := GetServiceBusQueue(context.Background(), provider,
		azureResourceName(config.ResourceGroup),
		azureResourceName(config.NamespaceName),
		config.QueueName,
		ctx.Logger,
	)
	if err != nil {
		return fmt.Errorf("failed to get Service Bus queue: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "azure.servicebus.queue", []any{output})
}

func (c *GetServiceBusQueueComponent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetServiceBusQueueComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetServiceBusQueueComponent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for this component")
}

func (c *GetServiceBusQueueComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetServiceBusQueueComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}
