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

type CreateQueue struct{}

type CreateQueueConfiguration struct {
	Region    string `json:"region" mapstructure:"region"`
	QueueName string `json:"queueName" mapstructure:"queueName"`
}

func (c *CreateQueue) Name() string {
	return "aws.sqs.createQueue"
}

func (c *CreateQueue) Label() string {
	return "SQS • Create Queue"
}

func (c *CreateQueue) Description() string {
	return "Create a new SQS queue"
}

func (c *CreateQueue) Documentation() string {
	return `The Create Queue component creates a new AWS SQS queue.

## Configuration

- **Region**: AWS region for the SQS queue
- **Queue Name**: Name of the queue to create`
}

func (c *CreateQueue) Icon() string {
	return "aws"
}

func (c *CreateQueue) Color() string {
	return "gray"
}

func (c *CreateQueue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateQueue) Configuration() []configuration.Field {
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
			Name:     "queueName",
			Label:    "Queue Name",
			Type:     configuration.FieldTypeString,
			Required: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
		},
	}
}

func (c *CreateQueue) Setup(ctx core.SetupContext) error {
	var config CreateQueueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.QueueName = strings.TrimSpace(config.QueueName)

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if config.QueueName == "" {
		return fmt.Errorf("queue name is required")
	}

	return nil
}

func (c *CreateQueue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateQueue) Execute(ctx core.ExecutionContext) error {
	var config CreateQueueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.QueueName = strings.TrimSpace(config.QueueName)

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if config.QueueName == "" {
		return fmt.Errorf("queue name is required")
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	queueURL, err := client.CreateQueue(config.QueueName, map[string]string{})
	if err != nil {
		return fmt.Errorf("failed to create SQS queue: %w", err)
	}

	output := map[string]any{
		"queueUrl":  queueURL,
		"queueName": config.QueueName,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.sqs.queue",
		[]any{output},
	)
}

func (c *CreateQueue) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateQueue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateQueue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateQueue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateQueue) Cleanup(ctx core.SetupContext) error {
	return nil
}
