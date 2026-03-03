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

type HeadBucket struct{}

type HeadBucketConfiguration struct {
	Region string `json:"region" mapstructure:"region"`
	Bucket string `json:"bucket" mapstructure:"bucket"`
}

func (c *HeadBucket) Name() string {
	return "aws.s3.headBucket"
}

func (c *HeadBucket) Label() string {
	return "S3 • Head Bucket"
}

func (c *HeadBucket) Description() string {
	return "Check if an S3 bucket exists and you have access"
}

func (c *HeadBucket) Documentation() string {
	return `The Head Bucket component checks whether an S3 bucket exists and you have permission to access it.

## Configuration

- **Region**: AWS region of the S3 bucket
- **Bucket**: Target S3 bucket to check`
}

func (c *HeadBucket) Icon() string {
	return "aws"
}

func (c *HeadBucket) Color() string {
	return "gray"
}

func (c *HeadBucket) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *HeadBucket) Configuration() []configuration.Field {
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

func (c *HeadBucket) Setup(ctx core.SetupContext) error {
	var config HeadBucketConfiguration
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

func (c *HeadBucket) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *HeadBucket) Execute(ctx core.ExecutionContext) error {
	var config HeadBucketConfiguration
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
	headers, err := client.HeadBucket(config.Bucket)
	if err != nil {
		return fmt.Errorf("failed to head S3 bucket: %w", err)
	}

	output := map[string]any{
		"bucket":           config.Bucket,
		"region":           headers.Get("X-Amz-Bucket-Region"),
		"exists":           true,
		"accessPointAlias": headers.Get("X-Amz-Access-Point-Alias") == "true",
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.s3.bucket",
		[]any{output},
	)
}

func (c *HeadBucket) Actions() []core.Action {
	return []core.Action{}
}

func (c *HeadBucket) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *HeadBucket) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *HeadBucket) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *HeadBucket) Cleanup(ctx core.SetupContext) error {
	return nil
}
