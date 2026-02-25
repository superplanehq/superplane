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

type PurgeQueue struct{}

type PurgeQueueConfiguration struct {
	Region string `json:"region" mapstructure:"region"`
	Queue  string `json:"queue" mapstructure:"queue"`
}

func (c *PurgeQueue) Name() string {
	return "aws.sqs.purgeQueue"
}

func (c *PurgeQueue) Label() string {
	return "SQS • Purge Queue"
}

func (c *PurgeQueue) Description() string {
	return "Purge all messages from an SQS queue"
}

func (c *PurgeQueue) Documentation() string {
	return `The Purge Queue component removes all messages from an AWS SQS queue.

## Configuration

- **Region**: AWS region of the SQS queue
- **Queue**: Target SQS queue to purge`
}

func (c *PurgeQueue) Icon() string {
	return "aws"
}

func (c *PurgeQueue) Color() string {
	return "gray"
}

func (c *PurgeQueue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PurgeQueue) Configuration() []configuration.Field {
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
	}
}

func (c *PurgeQueue) Setup(ctx core.SetupContext) error {
	var config PurgeQueueConfiguration
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

	return nil
}

func (c *PurgeQueue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PurgeQueue) Execute(ctx core.ExecutionContext) error {
	var config PurgeQueueConfiguration
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
	if err := client.PurgeQueue(config.Queue); err != nil {
		return fmt.Errorf("failed to purge SQS queue: %w", err)
	}

	output := map[string]any{
		"queueUrl": config.Queue,
		"purged":   true,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.sqs.queue.purged",
		[]any{output},
	)
}

func (c *PurgeQueue) Actions() []core.Action {
	return []core.Action{}
}

func (c *PurgeQueue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *PurgeQueue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *PurgeQueue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *PurgeQueue) Cleanup(ctx core.SetupContext) error {
	return nil
}
