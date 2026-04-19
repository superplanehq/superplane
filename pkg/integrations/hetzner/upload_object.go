package hetzner

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const UploadObjectPayloadType = "hetzner.object.uploaded"

type UploadObject struct{}

type UploadObjectSpec struct {
	Bucket      string `json:"bucket" mapstructure:"bucket"`
	Key         string `json:"key" mapstructure:"key"`
	Content     any    `json:"content" mapstructure:"content"`
	ContentType string `json:"contentType" mapstructure:"contentType"`
}

func (c *UploadObject) Name() string {
	return "hetzner.uploadObject"
}

func (c *UploadObject) Label() string {
	return "Upload Object"
}

func (c *UploadObject) Description() string {
	return "Upload content to a Hetzner Object Storage bucket"
}

func (c *UploadObject) Documentation() string {
	return `Uploads content to an object in a Hetzner Object Storage bucket using the S3-compatible API.

## Requirements

S3 credentials (Access Key ID, Secret Access Key, and Region) must be configured on the Hetzner integration.

## Configuration

- **Bucket**: The target bucket (dropdown or expression).
- **Key**: The object key / path within the bucket (supports expressions). Example: ` + "`artifacts/{{ $.run.id }}.json`" + `
- **Content**: The content to upload (expression). Strings are uploaded as-is; objects/arrays are JSON-serialized automatically.
- **Content Type** (optional): MIME type for the object. Defaults to ` + "`application/octet-stream`" + `.

## Output

Emits the bucket, key, size in bytes, and ETag on the default channel.
`
}

func (c *UploadObject) Icon() string {
	return "hetzner"
}

func (c *UploadObject) Color() string {
	return "gray"
}

func (c *UploadObject) ExampleOutput() map[string]any {
	return exampleOutputUploadObject()
}

func (c *UploadObject) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UploadObject) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "bucket",
			Label:       "Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Target bucket",
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
			Description: "Object key (path within the bucket)",
		},
		{
			Name:        "content",
			Label:       "Content",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "Content to upload. Strings are uploaded as-is; objects/arrays are JSON-serialized.",
		},
		{
			Name:        "contentType",
			Label:       "Content Type",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "application/octet-stream",
			Description: "MIME type for the object (e.g. application/json, text/plain)",
		},
	}
}

func (c *UploadObject) Setup(ctx core.SetupContext) error {
	spec := UploadObjectSpec{}
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

func (c *UploadObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UploadObject) Execute(ctx core.ExecutionContext) error {
	spec := UploadObjectSpec{}
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

	body, err := serializeContent(spec.Content)
	if err != nil {
		return fmt.Errorf("serialize content: %w", err)
	}

	contentType := strings.TrimSpace(spec.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	s3, err := NewHetznerS3Client(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	etag, err := s3.PutObject(bucket, key, contentType, body)
	if err != nil {
		return fmt.Errorf("upload object: %w", err)
	}

	payload := map[string]any{
		"bucket": bucket,
		"key":    key,
		"size":   len(body),
		"etag":   etag,
	}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, UploadObjectPayloadType, []any{payload})
}

// serializeContent converts the content value to bytes.
// Strings are returned as-is; everything else is JSON-serialized.
func serializeContent(content any) ([]byte, error) {
	if content == nil {
		return []byte{}, nil
	}
	if s, ok := content.(string); ok {
		return []byte(s), nil
	}
	return json.Marshal(content)
}

func (c *UploadObject) Actions() []core.Action                  { return nil }
func (c *UploadObject) HandleAction(_ core.ActionContext) error { return nil }
func (c *UploadObject) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (c *UploadObject) Cancel(_ core.ExecutionContext) error { return nil }
func (c *UploadObject) Cleanup(_ core.SetupContext) error    { return nil }
