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

type DeleteServiceBusTopicComponent struct {
}

type DeleteServiceBusTopicConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	NamespaceName string `json:"namespaceName" mapstructure:"namespaceName"`
	TopicName     string `json:"topicName" mapstructure:"topicName"`
}

func (c *DeleteServiceBusTopicComponent) Name() string {
	return "azure.deleteServiceBusTopic"
}

func (c *DeleteServiceBusTopicComponent) Label() string {
	return "Delete Service Bus Topic"
}

func (c *DeleteServiceBusTopicComponent) Description() string {
	return "Deletes an Azure Service Bus topic from the specified namespace"
}

func (c *DeleteServiceBusTopicComponent) Documentation() string {
	return `
The Delete Service Bus Topic component deletes an existing topic from an Azure Service Bus namespace.

## Use Cases

- **Environment cleanup**: Remove topics when tearing down environments
- **Infrastructure decommissioning**: Delete unused topics as part of cleanup workflows

## How It Works

1. Validates the topic configuration parameters
2. Deletes the topic via the Azure Service Bus ARM API (also deletes all subscriptions and rules)
3. Returns confirmation of deletion

## Configuration

- **Resource Group**: The Azure resource group containing the Service Bus namespace
- **Namespace**: The Service Bus namespace containing the topic
- **Topic Name**: The name of the topic to delete

## Output

Returns deletion confirmation:
- **name**: The name of the deleted topic
- **namespaceName**: The Service Bus namespace
- **resourceGroup**: The resource group
- **deleted**: Always true

## Notes

- If the topic does not exist, the operation completes successfully (idempotent)
- All subscriptions and rules under the topic are also deleted
- This operation is irreversible
`
}

func (c *DeleteServiceBusTopicComponent) Icon() string {
	return "azure"
}

func (c *DeleteServiceBusTopicComponent) Color() string {
	return "blue"
}

func (c *DeleteServiceBusTopicComponent) ExampleOutput() map[string]any {
	return map[string]any{
		"name":          "my-topic",
		"namespaceName": "my-ns",
		"resourceGroup": "my-rg",
		"deleted":       true,
	}
}

func (c *DeleteServiceBusTopicComponent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteServiceBusTopicComponent) Configuration() []configuration.Field {
	return []configuration.Field{
		serviceBusResourceGroupField(),
		serviceBusNamespaceField(),
		serviceBusTopicNameField(),
	}
}

func (c *DeleteServiceBusTopicComponent) Setup(ctx core.SetupContext) error {
	config := DeleteServiceBusTopicConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ResourceGroup == "" {
		return fmt.Errorf("resource group is required")
	}
	if config.NamespaceName == "" {
		return fmt.Errorf("namespace name is required")
	}
	if config.TopicName == "" {
		return fmt.Errorf("topic name is required")
	}

	return nil
}

func (c *DeleteServiceBusTopicComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteServiceBusTopicComponent) Execute(ctx core.ExecutionContext) error {
	config := DeleteServiceBusTopicConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return fmt.Errorf("Azure provider not available: %w", err)
	}

	output, err := DeleteServiceBusTopic(context.Background(), provider,
		azureResourceName(config.ResourceGroup),
		azureResourceName(config.NamespaceName),
		config.TopicName,
		ctx.Logger,
	)
	if err != nil {
		return fmt.Errorf("failed to delete Service Bus topic: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "azure.servicebus.topic.deleted", []any{output})
}

func (c *DeleteServiceBusTopicComponent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteServiceBusTopicComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteServiceBusTopicComponent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for this component")
}

func (c *DeleteServiceBusTopicComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteServiceBusTopicComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}
