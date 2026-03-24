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

type CreateServiceBusTopicComponent struct {
}

type CreateServiceBusTopicConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	NamespaceName string `json:"namespaceName" mapstructure:"namespaceName"`
	TopicName     string `json:"topicName" mapstructure:"topicName"`
}

func (c *CreateServiceBusTopicComponent) Name() string {
	return "azure.createServiceBusTopic"
}

func (c *CreateServiceBusTopicComponent) Label() string {
	return "Create Service Bus Topic"
}

func (c *CreateServiceBusTopicComponent) Description() string {
	return "Creates a new Azure Service Bus topic in the specified namespace"
}

func (c *CreateServiceBusTopicComponent) Documentation() string {
	return `
The Create Service Bus Topic component creates a new topic in an Azure Service Bus namespace.

## Use Cases

- **Pub/sub setup**: Provision topics for publish-subscribe messaging patterns
- **Environment provisioning**: Create topics for new environments or services
- **Dynamic topic creation**: Create topics on demand as part of scaling workflows

## How It Works

1. Validates the topic configuration parameters
2. Creates the topic via the Azure Service Bus ARM API
3. Returns the created topic details

## Configuration

- **Resource Group**: The Azure resource group containing the Service Bus namespace
- **Namespace**: The Service Bus namespace to create the topic in
- **Topic Name**: The name of the topic to create

## Output

Returns the created topic information:
- **id**: The Azure resource ID of the topic
- **name**: The name of the topic
- **namespaceName**: The Service Bus namespace
- **resourceGroup**: The resource group

## Notes

- If a topic with the same name already exists, it will be updated (idempotent operation)
- Topic is created with default settings
- Subscriptions to the topic must be created separately
`
}

func (c *CreateServiceBusTopicComponent) Icon() string {
	return "azure"
}

func (c *CreateServiceBusTopicComponent) Color() string {
	return "blue"
}

func (c *CreateServiceBusTopicComponent) ExampleOutput() map[string]any {
	return map[string]any{
		"id":            "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.ServiceBus/namespaces/my-ns/topics/my-topic",
		"name":          "my-topic",
		"namespaceName": "my-ns",
		"resourceGroup": "my-rg",
	}
}

func (c *CreateServiceBusTopicComponent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateServiceBusTopicComponent) Configuration() []configuration.Field {
	return []configuration.Field{
		serviceBusResourceGroupField(),
		serviceBusNamespaceField(),
		{
			Name:        "topicName",
			Label:       "Topic Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The name of the topic to create",
			Placeholder: "my-topic",
		},
	}
}

func (c *CreateServiceBusTopicComponent) Setup(ctx core.SetupContext) error {
	config := CreateServiceBusTopicConfiguration{}
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

func (c *CreateServiceBusTopicComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateServiceBusTopicComponent) Execute(ctx core.ExecutionContext) error {
	config := CreateServiceBusTopicConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return fmt.Errorf("Azure provider not available: %w", err)
	}

	output, err := CreateServiceBusTopic(context.Background(), provider,
		azureResourceName(config.ResourceGroup),
		azureResourceName(config.NamespaceName),
		config.TopicName,
		ctx.Logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create Service Bus topic: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "azure.servicebus.topic", []any{output})
}

func (c *CreateServiceBusTopicComponent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateServiceBusTopicComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateServiceBusTopicComponent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for this component")
}

func (c *CreateServiceBusTopicComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateServiceBusTopicComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}
