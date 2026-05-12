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

type CreateServiceBusQueueComponent struct {
}

type CreateServiceBusQueueConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	NamespaceName string `json:"namespaceName" mapstructure:"namespaceName"`
	QueueName     string `json:"queueName" mapstructure:"queueName"`
}

func (c *CreateServiceBusQueueComponent) Name() string {
	return "azure.createServiceBusQueue"
}

func (c *CreateServiceBusQueueComponent) Label() string {
	return "Create Service Bus Queue"
}

func (c *CreateServiceBusQueueComponent) Description() string {
	return "Creates a new Azure Service Bus queue in the specified namespace"
}

func (c *CreateServiceBusQueueComponent) Documentation() string {
	return `
The Create Service Bus Queue component creates a new queue in an Azure Service Bus namespace.

## Use Cases

- **Message pipeline setup**: Provision queues as part of infrastructure automation workflows
- **Environment provisioning**: Create queues for new environments or tenants
- **Dynamic queue creation**: Create queues on demand as part of scaling workflows

## How It Works

1. Validates the queue configuration parameters
2. Creates the queue via the Azure Service Bus ARM API
3. Returns the created queue details

## Configuration

- **Resource Group**: The Azure resource group containing the Service Bus namespace
- **Namespace**: The Service Bus namespace to create the queue in
- **Queue Name**: The name of the queue to create

## Output

Returns the created queue information:
- **id**: The Azure resource ID of the queue
- **name**: The name of the queue
- **namespaceName**: The Service Bus namespace
- **resourceGroup**: The resource group

## Notes

- If a queue with the same name already exists, it will be updated (idempotent operation)
- Queue is created with default settings (can be customized via the Azure portal after creation)
`
}

func (c *CreateServiceBusQueueComponent) Icon() string {
	return "azure"
}

func (c *CreateServiceBusQueueComponent) Color() string {
	return "blue"
}

func (c *CreateServiceBusQueueComponent) ExampleOutput() map[string]any {
	return map[string]any{
		"id":            "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.ServiceBus/namespaces/my-ns/queues/my-queue",
		"name":          "my-queue",
		"namespaceName": "my-ns",
		"resourceGroup": "my-rg",
	}
}

func (c *CreateServiceBusQueueComponent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateServiceBusQueueComponent) Configuration() []configuration.Field {
	return []configuration.Field{
		serviceBusResourceGroupField(),
		serviceBusNamespaceField(),
		{
			Name:        "queueName",
			Label:       "Queue Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The name of the queue to create",
			Placeholder: "my-queue",
		},
	}
}

func (c *CreateServiceBusQueueComponent) Setup(ctx core.SetupContext) error {
	config := CreateServiceBusQueueConfiguration{}
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

func (c *CreateServiceBusQueueComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateServiceBusQueueComponent) Execute(ctx core.ExecutionContext) error {
	config := CreateServiceBusQueueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return fmt.Errorf("Azure provider not available: %w", err)
	}

	output, err := CreateServiceBusQueue(context.Background(), provider,
		azureResourceName(config.ResourceGroup),
		azureResourceName(config.NamespaceName),
		config.QueueName,
		ctx.Logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create Service Bus queue: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "azure.servicebus.queue", []any{output})
}

func (c *CreateServiceBusQueueComponent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateServiceBusQueueComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateServiceBusQueueComponent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for this component")
}

func (c *CreateServiceBusQueueComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateServiceBusQueueComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}
