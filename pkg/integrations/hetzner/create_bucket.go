package hetzner

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const CreateBucketPayloadType = "hetzner.bucket.created"

type CreateBucket struct{}

type CreateBucketSpec struct {
	Bucket string `json:"bucket" mapstructure:"bucket"`
}

func (c *CreateBucket) Name() string {
	return "hetzner.createBucket"
}

func (c *CreateBucket) Label() string {
	return "Create Bucket"
}

func (c *CreateBucket) Description() string {
	return "Create a new Hetzner Object Storage bucket"
}

func (c *CreateBucket) Documentation() string {
	return `Creates a new bucket in Hetzner Object Storage using the S3-compatible API.

## Requirements

S3 credentials (Access Key ID, Secret Access Key, and Region) must be configured on the Hetzner integration.

## Configuration

- **Bucket**: The name of the bucket to create (supports expressions).

## Output

Emits the bucket name, region, and S3 endpoint on the default channel.
`
}

func (c *CreateBucket) Icon() string {
	return "hetzner"
}

func (c *CreateBucket) Color() string {
	return "gray"
}

func (c *CreateBucket) ExampleOutput() map[string]any {
	return exampleOutputCreateBucket()
}

func (c *CreateBucket) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateBucket) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "bucket",
			Label:       "Bucket",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "Name of the bucket to create",
		},
	}
}

func (c *CreateBucket) Setup(ctx core.SetupContext) error {
	spec := CreateBucketSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(readStringFromAny(spec.Bucket)) == "" {
		return fmt.Errorf("bucket is required")
	}
	return nil
}

func (c *CreateBucket) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateBucket) Execute(ctx core.ExecutionContext) error {
	spec := CreateBucketSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}
	bucket := strings.TrimSpace(readStringFromAny(spec.Bucket))
	if bucket == "" {
		return fmt.Errorf("bucket is required")
	}

	s3, err := NewHetznerS3Client(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	if err := s3.CreateBucket(bucket); err != nil {
		return fmt.Errorf("create bucket: %w", err)
	}

	payload := map[string]any{
		"bucket":   bucket,
		"region":   s3.region,
		"endpoint": s3.endpoint + "/" + bucket,
	}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateBucketPayloadType, []any{payload})
}

func (c *CreateBucket) Actions() []core.Action                  { return nil }
func (c *CreateBucket) HandleAction(_ core.ActionContext) error { return nil }
func (c *CreateBucket) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (c *CreateBucket) Cancel(_ core.ExecutionContext) error { return nil }
func (c *CreateBucket) Cleanup(_ core.SetupContext) error    { return nil }
