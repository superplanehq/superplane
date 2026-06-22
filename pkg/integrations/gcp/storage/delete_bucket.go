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
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

type DeleteBucket struct{}

type DeleteBucketSpec struct {
	Bucket string `json:"bucket" mapstructure:"bucket"`
}

func (d *DeleteBucket) Name() string {
	return "gcp.storage.deleteBucket"
}

func (d *DeleteBucket) Label() string {
	return "Cloud Storage • Delete Bucket"
}

func (d *DeleteBucket) Description() string {
	return "Delete a Cloud Storage bucket"
}

func (d *DeleteBucket) Documentation() string {
	return `The Delete Bucket component permanently deletes a Cloud Storage bucket.

## Use Cases

- **Teardown**: Remove a bucket when decommissioning an environment
- **Ephemeral cleanup**: Delete a preview/test bucket when it is no longer needed

## Configuration

- **Bucket**: The Cloud Storage bucket to delete (required)

## Output

Emits a ` + "`gcp.storage.bucket`" + ` payload with the bucket ` + "`name`" + ` and ` + "`deleted: true`" + `.

## Important Notes

- **The bucket must be empty before it can be deleted.** Cloud Storage rejects the request if it still contains objects (including noncurrent object versions).
- **This permanently deletes the bucket — it is irreversible.** Deleting a bucket frees its globally unique name for reuse by anyone.
- If the bucket does not exist, the component treats the deletion as already complete (idempotent).
- Requires the ` + "`roles/storage.admin`" + ` IAM role, and the **Cloud Storage API** (` + "`storage.googleapis.com`" + `) enabled.`
}

func (d *DeleteBucket) Icon() string {
	return "trash-2"
}

func (d *DeleteBucket) Color() string {
	return "red"
}

func (d *DeleteBucket) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteBucket) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "bucket",
			Label:       "Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Cloud Storage bucket to delete",
			Placeholder: "Select a bucket",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeBucket,
				},
			},
		},
	}
}

func (d *DeleteBucket) Setup(ctx core.SetupContext) error {
	spec := DeleteBucketSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if strings.TrimSpace(spec.Bucket) == "" {
		return fmt.Errorf("bucket is required")
	}
	return ctx.Metadata.Set(BucketNodeMetadata{Bucket: strings.TrimSpace(spec.Bucket)})
}

func (d *DeleteBucket) Execute(ctx core.ExecutionContext) error {
	spec := DeleteBucketSpec{}
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

	if err := deleteBucket(context.Background(), client, bucket); err != nil {
		if gcpcommon.IsNotFoundError(err) {
			// The bucket is already gone — confirm the deletion instead of failing.
			// This keeps the component idempotent when the bucket was removed out
			// of band.
			return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, bucketPayloadType, []any{
				map[string]any{"name": bucket, "deleted": true},
			})
		}
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to delete bucket", err, roleHintAdmin))
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, bucketPayloadType, []any{
		map[string]any{"name": bucket, "deleted": true},
	})
}

func (d *DeleteBucket) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteBucket) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteBucket) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteBucket) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (d *DeleteBucket) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeleteBucket) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
