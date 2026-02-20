package sqs

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type SendMessage struct{}

type SendMessageConfiguration struct {
	Region      string `json:"region" mapstructure:"region"`
	Queue       string `json:"queue" mapstructure:"queue"`
	MessageBody string `json:"messageBody" mapstructure:"messageBody"`
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
			Name:        "messageBody",
			Label:       "Message Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "Message payload",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "queue",
					Values: []string{"*"},
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
	config.Queue = strings.TrimSpace(config.Queue)
	config.MessageBody = strings.TrimSpace(config.MessageBody)

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if config.Queue == "" {
		return fmt.Errorf("queue is required")
	}

	if config.MessageBody == "" {
		return fmt.Errorf("message body is required")
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

	client := NewClient(ctx.HTTP, creds, config.Region)
	messageID, err := client.SendMessage(config.Queue, config.MessageBody)
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

