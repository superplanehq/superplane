package hetzner

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ListObjectsPayloadType = "hetzner.objects.listed"

const defaultMaxKeys = 100

type ListObjects struct{}

type ListObjectsSpec struct {
	Bucket  string `json:"bucket" mapstructure:"bucket"`
	Prefix  any    `json:"prefix" mapstructure:"prefix"`
	MaxKeys int    `json:"maxKeys" mapstructure:"maxKeys"`
}

func (c *ListObjects) Name() string {
	return "hetzner.listObjects"
}

func (c *ListObjects) Label() string {
	return "List Objects"
}

func (c *ListObjects) Description() string {
	return "List objects in a Hetzner Object Storage bucket"
}

func (c *ListObjects) Documentation() string {
	return `Lists objects in a Hetzner Object Storage bucket using the S3-compatible API.

## Requirements

S3 credentials (Access Key ID, Secret Access Key, and Region) must be configured on the Hetzner integration.

## Configuration

- **Bucket**: The bucket to list (dropdown or expression).
- **Prefix** (optional): Filter objects by key prefix (supports expressions). Example: ` + "`artifacts/`" + `
- **Max Keys** (optional): Maximum number of objects to return. Defaults to 100.

## Output

Emits the bucket, prefix, an array of objects (each with key, size, lastModified, etag), count, and a truncated flag on the default channel. When truncated is true, more objects exist beyond maxKeys; narrow the prefix or increase Max Keys to see them.
`
}

func (c *ListObjects) Icon() string {
	return "hetzner"
}

func (c *ListObjects) Color() string {
	return "gray"
}

func (c *ListObjects) ExampleOutput() map[string]any {
	return exampleOutputListObjects()
}

func (c *ListObjects) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ListObjects) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "bucket",
			Label:       "Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Bucket to list objects from",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "bucket",
				},
			},
		},
		{
			Name:        "prefix",
			Label:       "Prefix",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Description: "Filter objects by key prefix (optional)",
		},
		{
			Name:        "maxKeys",
			Label:       "Max Keys",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultMaxKeys,
			Description: "Maximum number of objects to return (default: 100)",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(1000),
				},
			},
		},
	}
}

func (c *ListObjects) Setup(ctx core.SetupContext) error {
	spec := ListObjectsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(spec.Bucket) == "" {
		return fmt.Errorf("bucket is required")
	}
	return nil
}

func (c *ListObjects) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ListObjects) Execute(ctx core.ExecutionContext) error {
	spec := ListObjectsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}
	bucket := strings.TrimSpace(spec.Bucket)
	if bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	prefix := strings.TrimSpace(readStringFromAny(spec.Prefix))
	maxKeys := spec.MaxKeys
	if maxKeys <= 0 {
		maxKeys = defaultMaxKeys
	}

	s3, err := NewHetznerS3Client(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	result, err := s3.ListObjects(bucket, prefix, maxKeys)
	if err != nil {
		return fmt.Errorf("list objects: %w", err)
	}

	objects := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		objects = append(objects, map[string]any{
			"key":          item.Key,
			"size":         item.Size,
			"lastModified": item.LastModified,
			"etag":         item.ETag,
		})
	}

	payload := map[string]any{
		"bucket":    bucket,
		"prefix":    prefix,
		"objects":   objects,
		"count":     len(objects),
		"truncated": result.Truncated,
	}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, ListObjectsPayloadType, []any{payload})
}

func intPtr(v int) *int { return &v }

func (c *ListObjects) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ListObjects) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
func (c *ListObjects) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (c *ListObjects) Cancel(_ core.ExecutionContext) error { return nil }
func (c *ListObjects) Cleanup(_ core.SetupContext) error    { return nil }
