package hetzner

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const DeleteBucketPayloadType = "hetzner.bucket.deleted"

type DeleteBucket struct{}

type DeleteBucketSpec struct {
	Bucket string `json:"bucket" mapstructure:"bucket"`
}

func (c *DeleteBucket) Name() string {
	return "hetzner.deleteBucket"
}

func (c *DeleteBucket) Label() string {
	return "Delete Bucket"
}

func (c *DeleteBucket) Description() string {
	return "Delete a Hetzner Object Storage bucket"
}

func (c *DeleteBucket) Documentation() string {
	return `Deletes a bucket from Hetzner Object Storage using the S3-compatible API.

## Requirements

S3 credentials (Access Key ID, Secret Access Key, and Region) must be configured on the Hetzner integration.
The bucket must be empty before it can be deleted.

## Configuration

- **Bucket**: The bucket to delete (dropdown or expression).

## Output

Emits the deleted bucket name on the default channel.
`
}

func (c *DeleteBucket) Icon() string {
	return "hetzner"
}

func (c *DeleteBucket) Color() string {
	return "gray"
}

func (c *DeleteBucket) ExampleOutput() map[string]any {
	return exampleOutputDeleteBucket()
}

func (c *DeleteBucket) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteBucket) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "bucket",
			Label:       "Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The bucket to delete",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "bucket",
				},
			},
		},
	}
}

func (c *DeleteBucket) Setup(ctx core.SetupContext) error {
	spec := DeleteBucketSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(spec.Bucket) == "" {
		return fmt.Errorf("bucket is required")
	}
	return nil
}

func (c *DeleteBucket) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteBucket) Execute(ctx core.ExecutionContext) error {
	spec := DeleteBucketSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}
	bucket := strings.TrimSpace(spec.Bucket)
	if bucket == "" {
		return fmt.Errorf("bucket is required")
	}

	s3, err := NewHetznerS3Client(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	if err := s3.DeleteBucket(bucket); err != nil {
		return fmt.Errorf("delete bucket: %w", err)
	}

	payload := map[string]any{"bucket": bucket}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteBucketPayloadType, []any{payload})
}

func (c *DeleteBucket) Actions() []core.Action                  { return nil }
func (c *DeleteBucket) HandleAction(_ core.ActionContext) error { return nil }
func (c *DeleteBucket) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (c *DeleteBucket) Cancel(_ core.ExecutionContext) error { return nil }
func (c *DeleteBucket) Cleanup(_ core.SetupContext) error    { return nil }
