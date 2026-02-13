package sns

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type PublishMessage struct{}

type PublishMessageConfiguration struct {
	Region            string         `json:"region" mapstructure:"region"`
	TopicArn          string         `json:"topicArn" mapstructure:"topicArn"`
	Message           string         `json:"message" mapstructure:"message"`
	Subject           string         `json:"subject" mapstructure:"subject"`
	MessageAttributes map[string]any `json:"messageAttributes" mapstructure:"messageAttributes"`
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
			Name:     "message",
			Label:    "Message",
			Type:     configuration.FieldTypeText,
			Required: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "topicArn",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:        "subject",
			Label:       "Subject",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional subject for supported protocols",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "message",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:        "messageAttributes",
			Label:       "Message Attributes",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "Optional message attributes as key-value pairs",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "message",
					Values: []string{"*"},
				},
			},
		},
	}
}

func (c *PublishMessage) Setup(ctx core.SetupContext) error {
	var config PublishMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: failed to decode setup configuration: %w", c.Name(), err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return fmt.Errorf("%s: invalid region: %w", c.Name(), err)
	}

	if _, err := requireTopicArn(config.TopicArn); err != nil {
		return fmt.Errorf("%s: invalid topic ARN: %w", c.Name(), err)
	}

	if _, err := requireMessage(config.Message); err != nil {
		return fmt.Errorf("%s: invalid message: %w", c.Name(), err)
	}

	return nil
}

func (c *PublishMessage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PublishMessage) Execute(ctx core.ExecutionContext) error {
	var config PublishMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: failed to decode execution configuration: %w", c.Name(), err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return fmt.Errorf("%s: invalid region: %w", c.Name(), err)
	}

	topicArn, err := requireTopicArn(config.TopicArn)
	if err != nil {
		return fmt.Errorf("%s: invalid topic ARN: %w", c.Name(), err)
	}

	message, err := requireMessage(config.Message)
	if err != nil {
		return fmt.Errorf("%s: invalid message: %w", c.Name(), err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("%s: failed to load AWS credentials from integration: %w", c.Name(), err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	result, err := client.PublishMessage(PublishMessageParameters{
		TopicArn:          topicArn,
		Message:           message,
		Subject:           config.Subject,
		MessageAttributes: mapAnyToStringMap(config.MessageAttributes),
	})
	if err != nil {
		return fmt.Errorf("%s: failed to publish message to topic %q: %w", c.Name(), topicArn, err)
	}

	if err := ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.sns.message.published", []any{result}); err != nil {
		return fmt.Errorf("%s: failed to emit published message payload: %w", c.Name(), err)
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
