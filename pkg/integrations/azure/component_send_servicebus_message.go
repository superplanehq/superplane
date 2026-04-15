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

type SendServiceBusMessageComponent struct {
}

type SendServiceBusMessageConfiguration struct {
	NamespaceName string `json:"namespaceName" mapstructure:"namespaceName"`
	QueueName     string `json:"queueName" mapstructure:"queueName"`
	Body          string `json:"body" mapstructure:"body"`
	ContentType   string `json:"contentType" mapstructure:"contentType"`
}

func (c *SendServiceBusMessageComponent) Name() string {
	return "azure.sendServiceBusMessage"
}

func (c *SendServiceBusMessageComponent) Label() string {
	return "Send Service Bus Message"
}

func (c *SendServiceBusMessageComponent) Description() string {
	return "Sends a message to an Azure Service Bus queue"
}

func (c *SendServiceBusMessageComponent) Documentation() string {
	return `
The Send Service Bus Message component sends a message to an Azure Service Bus queue.

## Use Cases

- **Event publishing**: Send events to downstream consumers via Service Bus queues
- **Task queuing**: Queue work items for background processing
- **System integration**: Bridge workflows to systems that consume Service Bus messages
- **Notifications**: Send structured notifications to queue consumers

## How It Works

1. Validates the message configuration parameters
2. Authenticates with the Service Bus data plane using Azure AD (no connection strings needed)
3. Posts the message to the specified queue
4. Returns confirmation of delivery

## Configuration

- **Namespace**: The Service Bus namespace containing the queue
- **Queue Name**: The name of the queue to send the message to
- **Body**: The message body content
- **Content Type** (optional): The MIME type of the message body (e.g., ` + "`application/json`" + `, ` + "`text/plain`" + `)

## Output

Returns send confirmation:
- **queue**: The name of the queue
- **namespaceName**: The Service Bus namespace
- **sent**: Always true

## Notes

- Authentication uses Azure AD OAuth2 (Service Bus Data Sender role required)
- The namespace is the short name (e.g., ` + "`my-namespace`" + `), not the FQDN
- Message is sent with at-least-once delivery guarantee
`
}

func (c *SendServiceBusMessageComponent) Icon() string {
	return "azure"
}

func (c *SendServiceBusMessageComponent) Color() string {
	return "blue"
}

func (c *SendServiceBusMessageComponent) ExampleOutput() map[string]any {
	return map[string]any{
		"queue":         "my-queue",
		"namespaceName": "my-ns",
		"sent":          true,
	}
}

func (c *SendServiceBusMessageComponent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *SendServiceBusMessageComponent) Configuration() []configuration.Field {
	return []configuration.Field{
		serviceBusNamespaceFieldStandalone(),
		serviceBusQueueNameField(),
		{
			Name:        "body",
			Label:       "Message Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The content of the message to send",
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

func (c *SendServiceBusMessageComponent) Setup(ctx core.SetupContext) error {
	config := SendServiceBusMessageConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.NamespaceName == "" {
		return fmt.Errorf("namespace name is required")
	}
	if config.QueueName == "" {
		return fmt.Errorf("queue name is required")
	}
	if config.Body == "" {
		return fmt.Errorf("message body is required")
	}

	return nil
}

func (c *SendServiceBusMessageComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SendServiceBusMessageComponent) Execute(ctx core.ExecutionContext) error {
	config := SendServiceBusMessageConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return fmt.Errorf("Azure provider not available: %w", err)
	}

	output, err := SendServiceBusMessage(context.Background(), provider,
		azureResourceName(config.NamespaceName),
		config.QueueName,
		config.Body,
		config.ContentType,
		ctx.Logger,
	)
	if err != nil {
		return fmt.Errorf("failed to send Service Bus message: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "azure.servicebus.message.sent", []any{output})
}

func (c *SendServiceBusMessageComponent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SendServiceBusMessageComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *SendServiceBusMessageComponent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for this component")
}

func (c *SendServiceBusMessageComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *SendServiceBusMessageComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}
