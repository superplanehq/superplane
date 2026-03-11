package pubsub

import (
	"context"
	"encoding/base64"
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
	publishMessageOutputChannel = "default"
	publishMessagePayloadType   = "gcp.pubsub.message.published"
)

type PublishMessage struct{}

type PublishMessageConfiguration struct {
	Topic  string  `json:"topic" mapstructure:"topic"`
	Format string  `json:"format" mapstructure:"format"`
	JSON   *any    `json:"json" mapstructure:"json"`
	Text   *string `json:"text" mapstructure:"text"`
}

func (c *PublishMessage) Name() string        { return "gcp.pubsub.publishMessage" }
func (c *PublishMessage) Label() string       { return "Pub/Sub • Publish Message" }
func (c *PublishMessage) Description() string { return "Publish a message to a GCP Pub/Sub topic" }
func (c *PublishMessage) Icon() string        { return "gcp" }
func (c *PublishMessage) Color() string       { return "gray" }

func (c *PublishMessage) Documentation() string {
	return `The Publish Message component sends a message to a GCP Pub/Sub topic.

## Use Cases

- **Event fan-out**: Broadcast workflow results to multiple subscribers
- **Notifications**: Publish operational updates to downstream systems
- **Automation**: Trigger Pub/Sub-based pipelines from workflows`
}

func (c *PublishMessage) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PublishMessage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "topic",
			Label:       "Topic",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the Pub/Sub topic to publish to.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeTopic,
					Parameters: []configuration.ParameterRef{},
				},
			},
		},
		{
			Name:     "format",
			Label:    "Message Format",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "json",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "JSON", Value: "json"},
						{Label: "Text", Value: "text"},
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
				{Field: "format", Values: []string{"json"}},
			},
		},
		{
			Name:     "text",
			Label:    "Text Message",
			Type:     configuration.FieldTypeText,
			Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "format", Values: []string{"text"}},
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
		return fmt.Errorf("format is required")
	}
	return nil
}

func (c *PublishMessage) Execute(ctx core.ExecutionContext) error {
	var config PublishMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}
	config.Topic = strings.TrimSpace(config.Topic)
	if config.Topic == "" {
		return ctx.ExecutionState.Fail("error", "topic is required")
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	data, err := c.buildMessageData(config)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to build message: %v", err))
	}

	projectID := client.ProjectID()
	messageID, err := PublishMessageToTopic(context.Background(), client, projectID, config.Topic, data, nil)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to publish message: %v", err))
	}

	return ctx.ExecutionState.Emit(publishMessageOutputChannel, publishMessagePayloadType, []any{
		map[string]any{
			"messageId": messageID,
			"topic":     config.Topic,
		},
	})
}

func (c *PublishMessage) buildMessageData(config PublishMessageConfiguration) (string, error) {
	var raw []byte
	var err error
	if config.Format == "text" {
		if config.Text == nil {
			return "", fmt.Errorf("text is required for text format")
		}
		raw = []byte(*config.Text)
	} else {
		raw, err = json.Marshal(config.JSON)
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

func (c *PublishMessage) Actions() []core.Action                  { return nil }
func (c *PublishMessage) HandleAction(_ core.ActionContext) error { return nil }
func (c *PublishMessage) Cancel(_ core.ExecutionContext) error    { return nil }
func (c *PublishMessage) Cleanup(_ core.SetupContext) error       { return nil }
func (c *PublishMessage) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *PublishMessage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
