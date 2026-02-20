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
	return "Delete an empty AWS S3 bucket"
}

func (c *DeleteBucket) Documentation() string {
	return `The Delete Bucket component deletes an AWS S3 bucket. The bucket must be empty before it can be deleted.

## Use Cases

- **Cleanup workflows**: Remove temporary buckets after execution
- **Lifecycle management**: Decommission unused storage resources
- **Rollback automation**: Remove buckets created in failed provisioning runs`
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
		regionField(),
		bucketField(),
	}
}

func (c *DeleteBucket) Setup(ctx core.SetupContext) error {
	var config DeleteBucketConfiguration
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

func (c *DeleteBucket) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteBucket) Execute(ctx core.ExecutionContext) error {
	var config DeleteBucketConfiguration
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
	if err := client.DeleteBucket(bucket); err != nil {
		return fmt.Errorf("failed to delete bucket %q: %w", bucket, err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.s3.bucket.deleted", []any{
		map[string]any{
			"bucket":  bucket,
			"deleted": true,
		},
	})
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
