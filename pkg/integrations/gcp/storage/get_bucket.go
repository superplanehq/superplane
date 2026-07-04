package storage

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetBucket struct{}

type GetBucketSpec struct {
	Bucket string `json:"bucket" mapstructure:"bucket"`
}

func (g *GetBucket) Name() string {
	return "gcp.storage.getBucket"
}

func (g *GetBucket) Label() string {
	return "Cloud Storage • Get Bucket"
}

func (g *GetBucket) Description() string {
	return "Fetch a Cloud Storage bucket's configuration and metadata"
}

func (g *GetBucket) Documentation() string {
	return `The Get Bucket component retrieves a Cloud Storage bucket's configuration and metadata.

## Use Cases

- **Enrichment**: Read a bucket's location or storage class to feed a downstream step
- **Existence check**: Confirm a bucket exists before writing to it
- **Auditing**: Capture bucket details as part of a workflow

## Configuration

- **Bucket**: The Cloud Storage bucket to fetch (required)

## Output

Emits a ` + "`gcp.storage.bucket`" + ` payload with the bucket's ` + "`name`" + `, ` + "`location`" + `, ` + "`locationType`" + `, ` + "`storageClass`" + `, ` + "`timeCreated`" + `, ` + "`selfLink`" + `, and a ` + "`consoleUrl`" + ` for opening the bucket in the Cloud Console.

## Important Notes

- Requires the ` + "`roles/storage.admin`" + ` (or ` + "`roles/storage.legacyBucketReader`" + `) IAM role, and the **Cloud Storage API** (` + "`storage.googleapis.com`" + `) enabled.`
}

func (g *GetBucket) Icon() string {
	return "database"
}

func (g *GetBucket) Color() string {
	return "blue"
}

func (g *GetBucket) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetBucket) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "bucket",
			Label:       "Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Cloud Storage bucket to fetch",
			Placeholder: "Select a bucket",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeBucket,
				},
			},
		},
	}
}

func (g *GetBucket) Setup(ctx core.SetupContext) error {
	spec := GetBucketSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if strings.TrimSpace(spec.Bucket) == "" {
		return fmt.Errorf("bucket is required")
	}
	return ctx.Metadata.Set(BucketNodeMetadata{Bucket: strings.TrimSpace(spec.Bucket)})
}

func (g *GetBucket) Execute(ctx core.ExecutionContext) error {
	spec := GetBucketSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}
	bucket := strings.TrimSpace(spec.Bucket)
	if bucket == "" {
		return ctx.ExecutionState.Fail("error", "bucket is required")
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	b, err := getBucket(context.Background(), client, bucket)
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to get bucket", err, roleHintViewer))
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, bucketPayloadType, []any{bucketPayload(b)})
}

func (g *GetBucket) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetBucket) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetBucket) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetBucket) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (g *GetBucket) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetBucket) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
