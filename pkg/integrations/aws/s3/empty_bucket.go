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

const deleteObjectsBatchSize = 1000

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
	return `The Empty Bucket component deletes all objects in an AWS S3 bucket, leaving the bucket itself intact.

## Use Cases

- **Pre-deletion cleanup**: Empty a bucket before deleting it
- **Environment reset**: Clear all objects from a staging or test bucket
- **Data lifecycle management**: Purge old data from storage buckets`
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
		return fmt.Errorf("invalid bucket: %w", err)
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

	region, err := requireRegion(config.Region)
	if err != nil {
		return fmt.Errorf("invalid region: %w", err)
	}

	bucket, err := requireBucket(config.Bucket)
	if err != nil {
		return fmt.Errorf("invalid bucket: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials from integration: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	totalDeleted, err := emptyBucket(client, bucket)
	if err != nil {
		return fmt.Errorf("failed to empty bucket %q: %w", bucket, err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.s3.bucket.emptied", []any{
		map[string]any{
			"bucket":       bucket,
			"objectsCount": totalDeleted,
		},
	})
}

func emptyBucket(client *Client, bucket string) (int, error) {
	totalDeleted := 0

	for {
		objects, err := client.ListObjects(bucket)
		if err != nil {
			return totalDeleted, fmt.Errorf("failed to list objects: %w", err)
		}

		if len(objects) == 0 {
			return totalDeleted, nil
		}

		for i := 0; i < len(objects); i += deleteObjectsBatchSize {
			end := i + deleteObjectsBatchSize
			if end > len(objects) {
				end = len(objects)
			}

			batch := objects[i:end]
			keys := make([]string, len(batch))
			for j, obj := range batch {
				keys[j] = obj.Key
			}

			if err := client.DeleteObjects(bucket, keys); err != nil {
				return totalDeleted, fmt.Errorf("failed to delete objects batch: %w", err)
			}

			totalDeleted += len(keys)
		}
	}
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
