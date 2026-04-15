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

type PublishServiceBusMessageComponent struct {
}

type PublishServiceBusMessageConfiguration struct {
	NamespaceName string `json:"namespaceName" mapstructure:"namespaceName"`
	TopicName     string `json:"topicName" mapstructure:"topicName"`
	Body          string `json:"body" mapstructure:"body"`
	ContentType   string `json:"contentType" mapstructure:"contentType"`
}

func (c *PublishServiceBusMessageComponent) Name() string {
	return "azure.publishServiceBusMessage"
}

func (c *PublishServiceBusMessageComponent) Label() string {
	return "Publish Service Bus Message"
}

func (c *PublishServiceBusMessageComponent) Description() string {
	return "Publishes a message to an Azure Service Bus topic"
}

func (c *PublishServiceBusMessageComponent) Documentation() string {
	return `
The Publish Service Bus Message component publishes a message to an Azure Service Bus topic.
All subscriptions on the topic will receive a copy of the message (fan-out).

## Use Cases

- **Event broadcasting**: Publish events to multiple consumers via topic subscriptions
- **System notifications**: Broadcast state changes to multiple downstream services
- **Decoupled workflows**: Fan out work to parallel processing pipelines
- **Audit logging**: Publish events to a topic with an audit subscription

## How It Works

1. Validates the message configuration parameters
2. Authenticates with the Service Bus data plane using Azure AD (no connection strings needed)
3. Posts the message to the specified topic
4. The topic routes the message to all matching subscriptions
5. Returns confirmation of delivery

## Configuration

- **Namespace**: The Service Bus namespace containing the topic
- **Topic Name**: The name of the topic to publish to
- **Body**: The message body content
- **Content Type** (optional): The MIME type of the message body (e.g., ` + "`application/json`" + `, ` + "`text/plain`" + `)

## Output

Returns publish confirmation:
- **topic**: The name of the topic
- **namespaceName**: The Service Bus namespace
- **published**: Always true

## Notes

- Authentication uses Azure AD OAuth2 (Service Bus Data Sender role required)
- The namespace is the short name (e.g., ` + "`my-namespace`" + `), not the FQDN
- Message is delivered to all active topic subscriptions
`
}

func (c *PublishServiceBusMessageComponent) Icon() string {
	return "azure"
}

func (c *PublishServiceBusMessageComponent) Color() string {
	return "blue"
}

func (c *PublishServiceBusMessageComponent) ExampleOutput() map[string]any {
	return map[string]any{
		"topic":         "my-topic",
		"namespaceName": "my-ns",
		"published":     true,
	}
}

func (c *PublishServiceBusMessageComponent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PublishServiceBusMessageComponent) Configuration() []configuration.Field {
	return []configuration.Field{
		serviceBusNamespaceFieldStandalone(),
		serviceBusTopicNameField(),
		{
			Name:        "body",
			Label:       "Message Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The content of the message to publish",
			Placeholder: `{"key": "value"}`,
		},
		{
			Name:        "contentType",
			Label:       "Content Type",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "application/json",
			Description: "The MIME type of the message body",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "application/json", Value: "application/json"},
						{Label: "text/plain", Value: "text/plain"},
						{Label: "application/xml", Value: "application/xml"},
						{Label: "application/octet-stream", Value: "application/octet-stream"},
					},
				},
			},
		},
	}
}

func (c *PublishServiceBusMessageComponent) Setup(ctx core.SetupContext) error {
	config := PublishServiceBusMessageConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.NamespaceName == "" {
		return fmt.Errorf("namespace name is required")
	}
	if config.TopicName == "" {
		return fmt.Errorf("topic name is required")
	}
	if config.Body == "" {
		return fmt.Errorf("message body is required")
	}

	return nil
}

func (c *PublishServiceBusMessageComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PublishServiceBusMessageComponent) Execute(ctx core.ExecutionContext) error {
	config := PublishServiceBusMessageConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return fmt.Errorf("Azure provider not available: %w", err)
	}

	output, err := PublishServiceBusMessage(context.Background(), provider,
		azureResourceName(config.NamespaceName),
		config.TopicName,
		config.Body,
		config.ContentType,
		ctx.Logger,
	)
	if err != nil {
		return fmt.Errorf("failed to publish Service Bus message: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "azure.servicebus.message.published", []any{output})
}

func (c *PublishServiceBusMessageComponent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *PublishServiceBusMessageComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *PublishServiceBusMessageComponent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for this component")
}

func (c *PublishServiceBusMessageComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *PublishServiceBusMessageComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}
