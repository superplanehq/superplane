package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

const (
	PublishFormatJSON = "json"
	PublishFormatText = "text"

	publishMessagePayloadType   = "gcp.pubsub.message.published"
	publishMessageOutputChannel = "default"
)

type PublishMessage struct{}

type PublishMessageConfiguration struct {
	Topic  string  `json:"topic" mapstructure:"topic"`
	Format string  `json:"format" mapstructure:"format"`
	JSON   *any    `json:"json,omitempty" mapstructure:"json"`
	Text   *string `json:"text,omitempty" mapstructure:"text"`
}

func (c *PublishMessage) Name() string {
	return "gcp.pubsub.publishMessage"
}

func (c *PublishMessage) Label() string {
	return "Pub/Sub • Publish Message"
}

func (c *PublishMessage) Description() string {
	return "Publish a message to a Google Cloud Pub/Sub topic"
}

func (c *PublishMessage) Documentation() string {
	return `The Publish Message component sends a message to a Google Cloud Pub/Sub topic.

## Use Cases

- **Event fan-out**: Broadcast workflow results to multiple subscribers
- **Cross-service communication**: Trigger downstream services through Pub/Sub
- **Data pipelines**: Feed data into streaming or batch processing pipelines

## Output

Emits the published message ID returned by the Pub/Sub API.`
}

func (c *PublishMessage) Icon() string {
	return "gcp"
}

func (c *PublishMessage) Color() string {
	return "gray"
}

func (c *PublishMessage) ExampleOutput() map[string]any {
	return map[string]any{
		"messageId": "12345678901234",
		"topic":     "my-topic",
	}
}

func (c *PublishMessage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PublishMessage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "topic",
			Label:       "Topic",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Pub/Sub topic ID (e.g. my-topic). The project is inferred from the integration.",
			Placeholder: "e.g. my-topic",
		},
		{
			Name:     "format",
			Label:    "Message Format",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  PublishFormatJSON,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Value: PublishFormatJSON, Label: "JSON"},
						{Value: PublishFormatText, Label: "Text"},
					},
				},
			},
		},
		{
			Name:     "json",
			Label:    "JSON Message",
			Type:     configuration.FieldTypeObject,
			Required: false,
			Default:  map[string]any{},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "topic", Values: []string{"*"}},
				{Field: "format", Values: []string{PublishFormatJSON}},
			},
		},
		{
			Name:     "text",
			Label:    "Text Message",
			Type:     configuration.FieldTypeText,
			Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "format", Values: []string{PublishFormatText}},
			},
		},
	}
}

func (c *PublishMessage) Setup(ctx core.SetupContext) error {
	var config PublishMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.Topic) == "" {
		return fmt.Errorf("topic is required")
	}

	if config.Format == "" {
		return fmt.Errorf("message format is required")
	}

	if config.Format == PublishFormatJSON && config.JSON == nil {
		return fmt.Errorf("JSON message is required")
	}

	if config.Format == PublishFormatText && config.Text == nil {
		return fmt.Errorf("text message is required")
	}

	return nil
}

func (c *PublishMessage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PublishMessage) Execute(ctx core.ExecutionContext) error {
	var config PublishMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	topic := strings.TrimSpace(config.Topic)
	if topic == "" {
		return ctx.ExecutionState.Fail("error", "topic is required")
	}

	data, err := c.buildMessageData(config)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to build message data: %v", err))
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	result, err := Publish(context.Background(), client, client.ProjectID(), topic, data)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to publish message to topic %q: %v", topic, err))
	}

	payload := map[string]any{
		"messageId": result,
		"topic":     topic,
	}

	return ctx.ExecutionState.Emit(publishMessageOutputChannel, publishMessagePayloadType, []any{payload})
}

func (c *PublishMessage) Actions() []core.Action {
	return nil
}

func (c *PublishMessage) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *PublishMessage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *PublishMessage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *PublishMessage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *PublishMessage) buildMessageData(config PublishMessageConfiguration) (string, error) {
	if config.Format == PublishFormatText {
		if config.Text == nil {
			return "", fmt.Errorf("text message is required")
		}
		return *config.Text, nil
	}

	if config.JSON == nil {
		return "", fmt.Errorf("JSON message is required")
	}

	data, err := json.Marshal(config.JSON)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON message: %w", err)
	}

	return string(data), nil
}
