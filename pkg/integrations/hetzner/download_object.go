package hetzner

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const DownloadObjectPayloadType = "hetzner.object.downloaded"

type DownloadObject struct{}

type DownloadObjectSpec struct {
	Bucket string `json:"bucket" mapstructure:"bucket"`
	Key    string `json:"key" mapstructure:"key"`
}

func (c *DownloadObject) Name() string {
	return "hetzner.downloadObject"
}

func (c *DownloadObject) Label() string {
	return "Download Object"
}

func (c *DownloadObject) Description() string {
	return "Download an object from a Hetzner Object Storage bucket"
}

func (c *DownloadObject) Documentation() string {
	return `Downloads an object from a Hetzner Object Storage bucket using the S3-compatible API.

## Requirements

S3 credentials (Access Key ID, Secret Access Key, and Region) must be configured on the Hetzner integration.

## Configuration

- **Bucket**: The source bucket (dropdown or expression).
- **Key**: The object key to download (supports expressions).

## Output

Emits the bucket, key, content as a string, content type, and size in bytes on the default channel.
`
}

func (c *DownloadObject) Icon() string {
	return "hetzner"
}

func (c *DownloadObject) Color() string {
	return "gray"
}

func (c *DownloadObject) ExampleOutput() map[string]any {
	return exampleOutputDownloadObject()
}

func (c *DownloadObject) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DownloadObject) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "bucket",
			Label:       "Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Source bucket",
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
			Description: "Object key to download",
		},
	}
}

func (c *DownloadObject) Setup(ctx core.SetupContext) error {
	spec := DownloadObjectSpec{}
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

func (c *DownloadObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DownloadObject) Execute(ctx core.ExecutionContext) error {
	spec := DownloadObjectSpec{}
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
	obj, err := s3.GetObject(bucket, key)
	if err != nil {
		return fmt.Errorf("download object: %w", err)
	}

	payload := map[string]any{
		"bucket":      bucket,
		"key":         key,
		"content":     string(obj.Body),
		"contentType": obj.ContentType,
		"size":        obj.Size,
	}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DownloadObjectPayloadType, []any{payload})
}

func (c *DownloadObject) Actions() []core.Action                  { return nil }
func (c *DownloadObject) HandleAction(_ core.ActionContext) error { return nil }
func (c *DownloadObject) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (c *DownloadObject) Cancel(_ core.ExecutionContext) error { return nil }
func (c *DownloadObject) Cleanup(_ core.SetupContext) error    { return nil }
