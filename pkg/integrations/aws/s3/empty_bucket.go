package s3

import (
	"fmt"
	"net/http"

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
	return "Delete all objects in an AWS S3 bucket"
}

func (c *EmptyBucket) Documentation() string {
	return `The Empty Bucket component deletes all objects in an S3 bucket. This is useful as a prerequisite before deleting the bucket itself.

## Use Cases

- **Bucket cleanup**: Remove all objects before deleting a bucket
- **Environment reset**: Clear bucket contents for fresh test runs
- **Data purge**: Remove all data from a bucket as part of compliance workflows`
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
		return fmt.Errorf("failed to decode setup configuration: %w", err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return fmt.Errorf("invalid region: %w", err)
	}

	if _, err := requireBucket(config.Bucket); err != nil {
		return err
	}

	return nil
}

func (c *EmptyBucket) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *EmptyBucket) Execute(ctx core.ExecutionContext) error {
	var config EmptyBucketConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode execution configuration: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials from integration: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, config.Region)
	totalDeleted := 0

	for {
		objects, err := client.ListObjects(config.Bucket)
		if err != nil {
			return fmt.Errorf("failed to list objects in bucket %q: %w", config.Bucket, err)
		}

		if len(objects) == 0 {
			break
		}

		keys := make([]string, len(objects))
		for i, obj := range objects {
			keys[i] = obj.Key
		}

		result, err := client.DeleteObjects(config.Bucket, keys)
		if err != nil {
			return fmt.Errorf("failed to delete objects from bucket %q: %w", config.Bucket, err)
		}

		if len(result.Errors) > 0 {
			return fmt.Errorf("failed to delete %d objects from bucket %q: %s", len(result.Errors), config.Bucket, result.Errors[0].Message)
		}

		totalDeleted += len(result.Deleted)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.s3.bucket.emptied", []any{
		&EmptyBucketResult{
			Bucket:       config.Bucket,
			DeletedCount: totalDeleted,
		},
	})
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
