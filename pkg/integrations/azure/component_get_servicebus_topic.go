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

type GetServiceBusTopicComponent struct {
}

type GetServiceBusTopicConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	NamespaceName string `json:"namespaceName" mapstructure:"namespaceName"`
	TopicName     string `json:"topicName" mapstructure:"topicName"`
}

func (c *GetServiceBusTopicComponent) Name() string {
	return "azure.getServiceBusTopic"
}

func (c *GetServiceBusTopicComponent) Label() string {
	return "Get Service Bus Topic"
}

func (c *GetServiceBusTopicComponent) Description() string {
	return "Retrieves details and metrics for an Azure Service Bus topic"
}

func (c *GetServiceBusTopicComponent) Documentation() string {
	return `
The Get Service Bus Topic component retrieves details and metrics for an existing Azure Service Bus topic.

## Use Cases

- **Monitoring**: Check subscription counts and message metrics in automated workflows
- **Conditional logic**: Branch workflows based on topic status or subscription count
- **Reporting**: Include topic metrics in notifications or dashboards

## How It Works

1. Validates the topic configuration parameters
2. Retrieves topic details via the Azure Service Bus ARM API
3. Returns the topic metadata and current metrics

## Configuration

- **Resource Group**: The Azure resource group containing the Service Bus namespace
- **Namespace**: The Service Bus namespace containing the topic
- **Topic Name**: The name of the topic to retrieve

## Output

Returns the topic information including:
- **id**: The Azure resource ID of the topic
- **name**: The name of the topic
- **namespaceName**: The Service Bus namespace
- **resourceGroup**: The resource group
- **properties**: Topic properties including subscription count, message counts, size, and status
`
}

func (c *GetServiceBusTopicComponent) Icon() string {
	return "azure"
}

func (c *GetServiceBusTopicComponent) Color() string {
	return "blue"
}

func (c *GetServiceBusTopicComponent) ExampleOutput() map[string]any {
	return map[string]any{
		"id":            "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.ServiceBus/namespaces/my-ns/topics/my-topic",
		"name":          "my-topic",
		"namespaceName": "my-ns",
		"resourceGroup": "my-rg",
		"properties": map[string]any{
			"subscriptionCount":      3,
			"sizeInBytes":            0,
			"maxSizeInMegabytes":     1024,
			"activeMessageCount":     0,
			"deadLetterMessageCount": 0,
			"status":                 "Active",
		},
	}
}

func (c *GetServiceBusTopicComponent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetServiceBusTopicComponent) Configuration() []configuration.Field {
	return []configuration.Field{
		serviceBusResourceGroupField(),
		serviceBusNamespaceField(),
		serviceBusTopicNameField(),
	}
}

func (c *GetServiceBusTopicComponent) Setup(ctx core.SetupContext) error {
	config := GetServiceBusTopicConfiguration{}
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

func (c *GetServiceBusTopicComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetServiceBusTopicComponent) Execute(ctx core.ExecutionContext) error {
	config := GetServiceBusTopicConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return fmt.Errorf("Azure provider not available: %w", err)
	}

	output, err := GetServiceBusTopic(context.Background(), provider,
		azureResourceName(config.ResourceGroup),
		azureResourceName(config.NamespaceName),
		config.TopicName,
		ctx.Logger,
	)
	if err != nil {
		return fmt.Errorf("failed to get Service Bus topic: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "azure.servicebus.topic", []any{output})
}

func (c *GetServiceBusTopicComponent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetServiceBusTopicComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetServiceBusTopicComponent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for this component")
}

func (c *GetServiceBusTopicComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetServiceBusTopicComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}
