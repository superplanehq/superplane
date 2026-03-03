package s3

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

type DeleteBucket struct{}

type DeleteBucketConfiguration struct {
	Region string `json:"region" mapstructure:"region"`
	Bucket string `json:"bucket" mapstructure:"bucket"`
}

func (c *DeleteBucket) Name() string {
	return "aws.s3.deleteBucket"
}

func (c *DeleteBucket) Label() string {
	return "S3 • Delete Bucket"
}

func (c *DeleteBucket) Description() string {
	return "Delete an S3 bucket (bucket must be empty)"
}

func (c *DeleteBucket) Documentation() string {
	return `The Delete Bucket component deletes an AWS S3 bucket. The bucket must be empty before it can be deleted.

## Configuration

- **Region**: AWS region of the S3 bucket
- **Bucket**: Target S3 bucket to delete`
}

func (c *DeleteBucket) Icon() string {
	return "aws"
}

func (c *DeleteBucket) Color() string {
	return "gray"
}

func (c *DeleteBucket) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteBucket) Configuration() []configuration.Field {
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
			Name:        "bucket",
			Label:       "Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Target S3 bucket",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "s3.bucket",
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

func (c *DeleteBucket) Setup(ctx core.SetupContext) error {
	var config DeleteBucketConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.Bucket = strings.TrimSpace(config.Bucket)

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if config.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}

	return nil
}

func (c *DeleteBucket) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteBucket) Execute(ctx core.ExecutionContext) error {
	var config DeleteBucketConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.Bucket = strings.TrimSpace(config.Bucket)

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if config.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	if err := client.DeleteBucket(config.Bucket); err != nil {
		return fmt.Errorf("failed to delete S3 bucket: %w", err)
	}

	output := map[string]any{
		"bucket":  config.Bucket,
		"deleted": true,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.s3.bucket.deleted",
		[]any{output},
	)
}

func (c *DeleteBucket) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteBucket) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteBucket) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DeleteBucket) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteBucket) Cleanup(ctx core.SetupContext) error {
	return nil
}
