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

type EmptyBucket struct{}

type EmptyBucketConfiguration struct {
	Region string `json:"region" mapstructure:"region"`
	Bucket string `json:"bucket" mapstructure:"bucket"`
}

func (c *EmptyBucket) Name() string {
	return "aws.s3.emptyBucket"
}

func (c *EmptyBucket) Label() string {
	return "S3 • Empty Bucket"
}

func (c *EmptyBucket) Description() string {
	return "Delete all objects in an S3 bucket"
}

func (c *EmptyBucket) Documentation() string {
	return `The Empty Bucket component deletes all objects in an AWS S3 bucket.

## Configuration

- **Region**: AWS region of the S3 bucket
- **Bucket**: Target S3 bucket to empty`
}

func (c *EmptyBucket) Icon() string {
	return "aws"
}

func (c *EmptyBucket) Color() string {
	return "gray"
}

func (c *EmptyBucket) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *EmptyBucket) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		bucketField(),
	}
}

func (c *EmptyBucket) Setup(ctx core.SetupContext) error {
	var config EmptyBucketConfiguration
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

func (c *EmptyBucket) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *EmptyBucket) Execute(ctx core.ExecutionContext) error {
	var config EmptyBucketConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.Bucket = strings.TrimSpace(config.Bucket)

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	totalDeleted := 0
	continuationToken := ""

	for {
		result, err := client.ListObjectsV2(config.Bucket, continuationToken)
		if err != nil {
			return fmt.Errorf("failed to list objects in S3 bucket: %w", err)
		}

		if len(result.Keys) == 0 {
			break
		}

		if err := client.DeleteObjects(config.Bucket, result.Keys); err != nil {
			return fmt.Errorf("failed to delete objects from S3 bucket: %w", err)
		}

		totalDeleted += len(result.Keys)

		if !result.IsTruncated {
			break
		}

		continuationToken = result.NextContinuationToken
	}

	output := map[string]any{
		"bucket":       config.Bucket,
		"emptied":      true,
		"objectsCount": totalDeleted,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.s3.bucket.emptied",
		[]any{output},
	)
}

func (c *EmptyBucket) Actions() []core.Action {
	return []core.Action{}
}

func (c *EmptyBucket) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *EmptyBucket) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *EmptyBucket) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *EmptyBucket) Cleanup(ctx core.SetupContext) error {
	return nil
}
