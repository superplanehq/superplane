package sns

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	PublishMessageFormatJSON = "json"
	PublishMessageFormatText = "text"
)

type PublishMessage struct{}

type PublishMessageConfiguration struct {
	Region   string  `json:"region" mapstructure:"region"`
	TopicArn string  `json:"topicArn" mapstructure:"topicArn"`
	Format   string  `json:"format" mapstructure:"format"`
	JSON     *any    `json:"json" mapstructure:"json"`
	Text     *string `json:"text" mapstructure:"text"`
}

func (c *PublishMessage) Name() string {
	return "aws.sns.publishMessage"
}

func (c *PublishMessage) Label() string {
	return "SNS • Publish Message"
}

func (c *PublishMessage) Description() string {
	return "Publish a message to an AWS SNS topic"
}

func (c *PublishMessage) Documentation() string {
	return `The Publish Message component sends a message to an AWS SNS topic.

## Use Cases

- **Event fan-out**: Broadcast workflow results to multiple subscribers
- **Notifications**: Send operational updates to users and systems
- **Automation**: Trigger downstream subscribers through SNS delivery`
}

func (c *PublishMessage) Icon() string {
	return "aws"
}

func (c *PublishMessage) Color() string {
	return "gray"
}

func (c *PublishMessage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PublishMessage) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		topicField(),
		{
			Name:     "format",
			Label:    "Message Format",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  PublishMessageFormatJSON,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Value: PublishMessageFormatJSON, Label: "JSON"},
						{Value: PublishMessageFormatText, Label: "Text"},
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
				{
					Field:  "topicArn",
					Values: []string{"*"},
				},
				{
					Field:  "format",
					Values: []string{PublishMessageFormatJSON},
				},
			},
		},
		{
			Name:     "text",
			Label:    "Text Message",
			Type:     configuration.FieldTypeText,
			Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "format",
					Values: []string{PublishMessageFormatText},
				},
			},
		},
	}
}

func (c *PublishMessage) Setup(ctx core.SetupContext) error {
	var config PublishMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode setup configuration: %w", err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return fmt.Errorf("invalid region: %w", err)
	}

	if _, err := requireTopicArn(config.TopicArn); err != nil {
		return fmt.Errorf("invalid topic ARN: %w", err)
	}

	if config.Format == "" {
		return fmt.Errorf("format is required")
	}

	if config.Format == PublishMessageFormatJSON && config.JSON == nil {
		return fmt.Errorf("JSON message is required")
	}

	if config.Format == PublishMessageFormatText && config.Text == nil {
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
		return fmt.Errorf("failed to decode execution configuration: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials from integration: %w", err)
	}

	params, err := c.buildPublishMessageParameters(config)
	if err != nil {
		return fmt.Errorf("failed to build publish message parameters: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, config.Region)
	result, err := client.PublishMessage(*params)
	if err != nil {
		return fmt.Errorf("failed to publish message to topic %q: %w", config.TopicArn, err)
	}

	if err := ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.sns.message.published", []any{result}); err != nil {
		return fmt.Errorf("failed to emit published message payload: %w", err)
	}

	return nil
}

func (c *PublishMessage) Actions() []core.Action {
	return []core.Action{}
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

func (c *PublishMessage) buildPublishMessageParameters(config PublishMessageConfiguration) (*PublishMessageParameters, error) {
	if config.Format == PublishMessageFormatText {
		return &PublishMessageParameters{
			TopicArn: config.TopicArn,
			Message:  *config.Text,
		}, nil
	}

	message, err := json.Marshal(config.JSON)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON message: %w", err)
	}

	return &PublishMessageParameters{
		TopicArn: config.TopicArn,
		Message:  string(message),
	}, nil
}
