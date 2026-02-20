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
	return "Check if an AWS S3 bucket exists and is accessible"
}

func (c *HeadBucket) Documentation() string {
	return `The Head Bucket component checks whether an S3 bucket exists and is accessible using the current credentials.

## Use Cases

- **Validation**: Verify bucket existence before performing operations
- **Health checks**: Confirm access to required storage resources
- **Pre-flight checks**: Validate infrastructure prerequisites in workflows`
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
		regionField(),
		bucketField(),
	}
}

func (c *HeadBucket) Setup(ctx core.SetupContext) error {
	var config HeadBucketConfiguration
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

func (c *HeadBucket) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *HeadBucket) Execute(ctx core.ExecutionContext) error {
	var config HeadBucketConfiguration
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
	result, err := client.HeadBucket(bucket)
	if err != nil {
		return fmt.Errorf("failed to head bucket %q: %w", bucket, err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.s3.bucket.head", []any{result})
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
