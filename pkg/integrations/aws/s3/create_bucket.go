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

type CreateBucket struct{}

type CreateBucketConfiguration struct {
	Region     string `json:"region" mapstructure:"region"`
	BucketName string `json:"bucketName" mapstructure:"bucketName"`
}

func (c *CreateBucket) Name() string {
	return "aws.s3.createBucket"
}

func (c *CreateBucket) Label() string {
	return "S3 • Create Bucket"
}

func (c *CreateBucket) Description() string {
	return "Create a new S3 bucket"
}

func (c *CreateBucket) Documentation() string {
	return `The Create Bucket component creates a new AWS S3 bucket.

## Configuration

- **Region**: AWS region for the S3 bucket
- **Bucket Name**: Name of the bucket to create`
}

func (c *CreateBucket) Icon() string {
	return "aws"
}

func (c *CreateBucket) Color() string {
	return "gray"
}

func (c *CreateBucket) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateBucket) Configuration() []configuration.Field {
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
			Name:     "bucketName",
			Label:    "Bucket Name",
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

func (c *CreateBucket) Setup(ctx core.SetupContext) error {
	var config CreateBucketConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.BucketName = strings.TrimSpace(config.BucketName)

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if config.BucketName == "" {
		return fmt.Errorf("bucket name is required")
	}

	return nil
}

func (c *CreateBucket) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateBucket) Execute(ctx core.ExecutionContext) error {
	var config CreateBucketConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.BucketName = strings.TrimSpace(config.BucketName)

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if config.BucketName == "" {
		return fmt.Errorf("bucket name is required")
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	location, err := client.CreateBucket(config.BucketName)
	if err != nil {
		return fmt.Errorf("failed to create S3 bucket: %w", err)
	}

	output := map[string]any{
		"bucketName": config.BucketName,
		"region":     config.Region,
		"location":   location,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.s3.bucket",
		[]any{output},
	)
}

func (c *CreateBucket) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateBucket) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateBucket) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateBucket) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateBucket) Cleanup(ctx core.SetupContext) error {
	return nil
}
