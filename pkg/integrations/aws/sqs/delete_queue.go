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

type DeleteQueue struct{}

type DeleteQueueConfiguration struct {
	Region string `json:"region" mapstructure:"region"`
	Queue  string `json:"queue" mapstructure:"queue"`
}

func (c *DeleteQueue) Name() string {
	return "aws.sqs.deleteQueue"
}

func (c *DeleteQueue) Label() string {
	return "SQS • Delete Queue"
}

func (c *DeleteQueue) Description() string {
	return "Delete an SQS queue"
}

func (c *DeleteQueue) Documentation() string {
	return `The Delete Queue component deletes an AWS SQS queue.

## Configuration

- **Region**: AWS region of the SQS queue
- **Queue**: Target SQS queue to delete`
}

func (c *DeleteQueue) Icon() string {
	return "aws"
}

func (c *DeleteQueue) Color() string {
	return "gray"
}

func (c *DeleteQueue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteQueue) Configuration() []configuration.Field {
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

func (c *DeleteQueue) Setup(ctx core.SetupContext) error {
	var config DeleteQueueConfiguration
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

func (c *DeleteQueue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteQueue) Execute(ctx core.ExecutionContext) error {
	var config DeleteQueueConfiguration
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
	if err := client.DeleteQueue(config.Queue); err != nil {
		return fmt.Errorf("failed to delete SQS queue: %w", err)
	}

	output := map[string]any{
		"queueUrl": config.Queue,
		"deleted":  true,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.sqs.queue.deleted",
		[]any{output},
	)
}

func (c *DeleteQueue) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteQueue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteQueue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DeleteQueue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteQueue) Cleanup(ctx core.SetupContext) error {
	return nil
}
