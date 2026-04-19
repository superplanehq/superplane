package hetzner

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const DeleteObjectPayloadType = "hetzner.object.deleted"

type DeleteObject struct{}

type DeleteObjectSpec struct {
	Bucket string `json:"bucket" mapstructure:"bucket"`
	Key    string `json:"key" mapstructure:"key"`
}

func (c *DeleteObject) Name() string {
	return "hetzner.deleteObject"
}

func (c *DeleteObject) Label() string {
	return "Delete Object"
}

func (c *DeleteObject) Description() string {
	return "Delete an object from a Hetzner Object Storage bucket"
}

func (c *DeleteObject) Documentation() string {
	return `Deletes an object from a Hetzner Object Storage bucket using the S3-compatible API.

## Requirements

S3 credentials (Access Key ID, Secret Access Key, and Region) must be configured on the Hetzner integration.

## Configuration

- **Bucket**: The bucket containing the object (dropdown or expression).
- **Key**: The object key to delete (supports expressions).

## Output

Emits the bucket and key on the default channel.
`
}

func (c *DeleteObject) Icon() string {
	return "hetzner"
}

func (c *DeleteObject) Color() string {
	return "gray"
}

func (c *DeleteObject) ExampleOutput() map[string]any {
	return exampleOutputDeleteObject()
}

func (c *DeleteObject) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteObject) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "bucket",
			Label:       "Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Bucket containing the object",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "bucket",
				},
			},
		},
		{
			Name:        "key",
			Label:       "Key",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "Object key to delete",
		},
	}
}

func (c *DeleteObject) Setup(ctx core.SetupContext) error {
	spec := DeleteObjectSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(spec.Bucket) == "" {
		return fmt.Errorf("bucket is required")
	}
	if strings.TrimSpace(readStringFromAny(spec.Key)) == "" {
		return fmt.Errorf("key is required")
	}
	return nil
}

func (c *DeleteObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteObject) Execute(ctx core.ExecutionContext) error {
	spec := DeleteObjectSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}
	bucket := strings.TrimSpace(spec.Bucket)
	if bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	key := strings.TrimSpace(readStringFromAny(spec.Key))
	if key == "" {
		return fmt.Errorf("key is required")
	}

	s3, err := NewHetznerS3Client(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	if err := s3.DeleteObject(bucket, key); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}

	payload := map[string]any{"bucket": bucket, "key": key}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteObjectPayloadType, []any{payload})
}

func (c *DeleteObject) Actions() []core.Action                  { return nil }
func (c *DeleteObject) HandleAction(_ core.ActionContext) error { return nil }
func (c *DeleteObject) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (c *DeleteObject) Cancel(_ core.ExecutionContext) error { return nil }
func (c *DeleteObject) Cleanup(_ core.SetupContext) error    { return nil }
