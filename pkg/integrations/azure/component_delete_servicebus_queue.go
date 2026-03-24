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

type DeleteServiceBusQueueComponent struct {
}

type DeleteServiceBusQueueConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	NamespaceName string `json:"namespaceName" mapstructure:"namespaceName"`
	QueueName     string `json:"queueName" mapstructure:"queueName"`
}

func (c *DeleteServiceBusQueueComponent) Name() string {
	return "azure.deleteServiceBusQueue"
}

func (c *DeleteServiceBusQueueComponent) Label() string {
	return "Delete Service Bus Queue"
}

func (c *DeleteServiceBusQueueComponent) Description() string {
	return "Deletes an Azure Service Bus queue from the specified namespace"
}

func (c *DeleteServiceBusQueueComponent) Documentation() string {
	return `
The Delete Service Bus Queue component deletes an existing queue from an Azure Service Bus namespace.

## Use Cases

- **Environment cleanup**: Remove queues when tearing down environments
- **Infrastructure decommissioning**: Delete unused queues as part of cleanup workflows

## How It Works

1. Validates the queue configuration parameters
2. Deletes the queue via the Azure Service Bus ARM API
3. Returns confirmation of deletion

## Configuration

- **Resource Group**: The Azure resource group containing the Service Bus namespace
- **Namespace**: The Service Bus namespace containing the queue
- **Queue Name**: The name of the queue to delete

## Output

Returns deletion confirmation:
- **name**: The name of the deleted queue
- **namespaceName**: The Service Bus namespace
- **resourceGroup**: The resource group
- **deleted**: Always true

## Notes

- If the queue does not exist, the operation completes successfully (idempotent)
- All messages in the queue are permanently deleted along with the queue
- This operation is irreversible
`
}

func (c *DeleteServiceBusQueueComponent) Icon() string {
	return "azure"
}

func (c *DeleteServiceBusQueueComponent) Color() string {
	return "blue"
}

func (c *DeleteServiceBusQueueComponent) ExampleOutput() map[string]any {
	return map[string]any{
		"name":          "my-queue",
		"namespaceName": "my-ns",
		"resourceGroup": "my-rg",
		"deleted":       true,
	}
}

func (c *DeleteServiceBusQueueComponent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteServiceBusQueueComponent) Configuration() []configuration.Field {
	return []configuration.Field{
		serviceBusResourceGroupField(),
		serviceBusNamespaceField(),
		serviceBusQueueNameField(),
	}
}

func (c *DeleteServiceBusQueueComponent) Setup(ctx core.SetupContext) error {
	config := DeleteServiceBusQueueConfiguration{}
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

func (c *DeleteServiceBusQueueComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteServiceBusQueueComponent) Execute(ctx core.ExecutionContext) error {
	config := DeleteServiceBusQueueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return fmt.Errorf("Azure provider not available: %w", err)
	}

	output, err := DeleteServiceBusQueue(context.Background(), provider,
		azureResourceName(config.ResourceGroup),
		azureResourceName(config.NamespaceName),
		config.QueueName,
		ctx.Logger,
	)
	if err != nil {
		return fmt.Errorf("failed to delete Service Bus queue: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "azure.servicebus.queue.deleted", []any{output})
}

func (c *DeleteServiceBusQueueComponent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteServiceBusQueueComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteServiceBusQueueComponent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for this component")
}

func (c *DeleteServiceBusQueueComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteServiceBusQueueComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}
