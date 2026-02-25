package sqs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	SendMessageFormatJSON = "json"
	SendMessageFormatXML  = "xml"
	SendMessageFormatText = "text"
)

type SendMessage struct{}

type SendMessageConfiguration struct {
	Region string  `json:"region" mapstructure:"region"`
	Queue  string  `json:"queue" mapstructure:"queue"`
	Format string  `json:"format" mapstructure:"format"`
	JSON   *any    `json:"json" mapstructure:"json"`
	XML    *string `json:"xml" mapstructure:"xml"`
	Text   *string `json:"text" mapstructure:"text"`
}

func (c *SendMessage) Name() string {
	return "aws.sqs.sendMessage"
}

func (c *SendMessage) Label() string {
	return "SQS • Send Message"
}

func (c *SendMessage) Description() string {
	return "Send a message to an SQS queue"
}

func (c *SendMessage) Documentation() string {
	return `The Send Message component publishes a message to an AWS SQS queue.

## Configuration

- **Region**: AWS region of the SQS queue
- **Queue**: Target SQS queue
- **Message Body**: The message payload to send`
}

func (c *SendMessage) Icon() string {
	return "aws"
}

func (c *SendMessage) Color() string {
	return "gray"
}

func (c *SendMessage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *SendMessage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-east-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: common.AllRegions,
				},
			},
		},
		{
			Name:        "queue",
			Label:       "Queue",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Target SQS queue",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "sqs.queue",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
		},
		{
			Name:     "format",
			Label:    "Message Format",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  SendMessageFormatJSON,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Value: SendMessageFormatJSON, Label: "JSON"},
						{Value: SendMessageFormatXML, Label: "XML"},
						{Value: SendMessageFormatText, Label: "Text"},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "queue",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:     "json",
			Label:    "JSON Message",
			Type:     configuration.FieldTypeObject,
			Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "queue",
					Values: []string{"*"},
				},
				{
					Field:  "format",
					Values: []string{SendMessageFormatJSON},
				},
			},
		},
		{
			Name:     "xml",
			Label:    "XML Message",
			Type:     configuration.FieldTypeXML,
			Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "queue",
					Values: []string{"*"},
				},
				{
					Field:  "format",
					Values: []string{SendMessageFormatXML},
				},
			},
		},
		{
			Name:     "text",
			Label:    "Text",
			Type:     configuration.FieldTypeText,
			Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "queue",
					Values: []string{"*"},
				},
				{
					Field:  "format",
					Values: []string{SendMessageFormatText},
				},
			},
		},
	}
}

func (c *SendMessage) Setup(ctx core.SetupContext) error {
	var config SendMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	config.Queue = strings.TrimSpace(config.Queue)
	if config.Queue == "" {
		return fmt.Errorf("queue is required")
	}

	if config.Format == "" {
		return fmt.Errorf("format is required")
	}

	if config.Format == SendMessageFormatJSON && config.JSON == nil {
		return fmt.Errorf("JSON message is required")
	}

	if config.Format == SendMessageFormatXML && config.XML == nil {
		return fmt.Errorf("XML message is required")
	}

	if config.Format == SendMessageFormatText && config.Text == nil {
		return fmt.Errorf("text message is required")
	}

	return nil
}

func (c *SendMessage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SendMessage) Execute(ctx core.ExecutionContext) error {
	var config SendMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.Queue = strings.TrimSpace(config.Queue)

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if config.Queue == "" {
		return fmt.Errorf("queue is required")
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	messageBody, err := c.buildMessageBody(config)
	if err != nil {
		return fmt.Errorf("failed to build message body: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	messageID, err := client.SendMessage(config.Queue, messageBody)
	if err != nil {
		return fmt.Errorf("failed to send SQS message: %w", err)
	}

	output := map[string]any{
		"queueUrl":  config.Queue,
		"messageId": messageID,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.sqs.message",
		[]any{output},
	)
}

func (c *SendMessage) buildMessageBody(config SendMessageConfiguration) (string, error) {
	if config.Format == SendMessageFormatText {
		if config.Text == nil || *config.Text == "" {
			return "", fmt.Errorf("text message is required")
		}

		return *config.Text, nil
	}

	if config.Format == SendMessageFormatXML {
		if config.XML == nil || *config.XML == "" {
			return "", fmt.Errorf("XML message is required")
		}

		return *config.XML, nil
	}

	if config.JSON == nil {
		return "", fmt.Errorf("JSON message is required")
	}

	message, err := json.Marshal(config.JSON)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON message: %w", err)
	}

	return string(message), nil
}

func (c *SendMessage) Actions() []core.Action {
	return []core.Action{}
}

func (c *SendMessage) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *SendMessage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *SendMessage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SendMessage) Cleanup(ctx core.SetupContext) error {
	return nil
}
