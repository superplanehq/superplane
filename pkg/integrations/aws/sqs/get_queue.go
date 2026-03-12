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

type GetQueue struct{}

type GetQueueConfiguration struct {
	Region string `json:"region" mapstructure:"region"`
	Queue  string `json:"queue" mapstructure:"queue"`
}

func (c *GetQueue) Name() string {
	return "aws.sqs.getQueue"
}

func (c *GetQueue) Label() string {
	return "SQS • Get Queue"
}

func (c *GetQueue) Description() string {
	return "Get metadata and attributes for an SQS queue"
}

func (c *GetQueue) Documentation() string {
	return `The Get Queue component retrieves metadata and attributes for an AWS SQS queue.

## Configuration

- **Region**: AWS region of the SQS queue
- **Queue**: Target SQS queue`
}

func (c *GetQueue) Icon() string {
	return "aws"
}

func (c *GetQueue) Color() string {
	return "gray"
}

func (c *GetQueue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetQueue) Configuration() []configuration.Field {
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

func (c *GetQueue) Setup(ctx core.SetupContext) error {
	var config GetQueueConfiguration
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

func (c *GetQueue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetQueue) Execute(ctx core.ExecutionContext) error {
	var config GetQueueConfiguration
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
	attributes, err := client.GetQueueAttributes(config.Queue)
	if err != nil {
		return fmt.Errorf("failed to get SQS queue attributes: %w", err)
	}

	output := map[string]any{
		"queueUrl":   config.Queue,
		"attributes": attributes,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.sqs.queue",
		[]any{output},
	)
}

func (c *GetQueue) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetQueue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetQueue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetQueue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetQueue) Cleanup(ctx core.SetupContext) error {
	return nil
}
