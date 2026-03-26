package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteObject struct{}

type DeleteObjectSpec struct {
	Bucket   string `mapstructure:"bucket"`
	FilePath string `mapstructure:"filePath"`
}

func (d *DeleteObject) Name() string {
	return "digitalocean.deleteObject"
}

func (d *DeleteObject) Label() string {
	return "Delete Object"
}

func (d *DeleteObject) Description() string {
	return "Delete an object from DigitalOcean Spaces Object Storage"
}

func (d *DeleteObject) Documentation() string {
	return `The Delete Object component permanently removes an object from a DigitalOcean Spaces bucket.

## Prerequisites

Spaces Access Key ID and Secret Access Key must be configured in the DigitalOcean integration settings.

## Configuration

- **Bucket**: The Spaces bucket containing the object. The dropdown lists all buckets — the region is determined automatically.
- **File Path**: The full path to the object within the bucket (e.g. reports/daily.csv). Supports expressions to reference a path from an upstream component.

## Output

- **bucket**: The bucket name
- **filePath**: The path of the deleted object
- **deleted**: Always ` + "`true`" + ` on success

## Notes

- This operation is **permanent** and cannot be undone
- The operation succeeds even if the object does not exist (idempotent)

## Use Cases

- **Cleanup**: Remove temporary or processed files after a workflow completes
- **Rotation**: Delete old reports or artifacts as part of a file rotation policy
- **Pipeline teardown**: Remove objects created during a pipeline run`
}

func (d *DeleteObject) Icon() string {
	return "trash-2"
}

func (d *DeleteObject) Color() string {
	return "red"
}

func (d *DeleteObject) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteObject) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "bucket",
			Label:       "Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Spaces bucket containing the object",
			Placeholder: "Select a bucket",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "spaces_bucket",
				},
			},
		},
		{
			Name:        "filePath",
			Label:       "File Path",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Full path including file name and extension. Use / to organize into folders (e.g. reports/2024/daily.csv)",
			Placeholder: "reports/daily.csv",
		},
	}
}

func (d *DeleteObject) Setup(ctx core.SetupContext) error {
	accessKey, _ := ctx.Integration.GetConfig("spacesAccessKey")
	if strings.TrimSpace(string(accessKey)) == "" {
		return errors.New("spaces access key is missing — add it in the DigitalOcean integration settings")
	}

	secretKey, _ := ctx.Integration.GetConfig("spacesSecretKey")
	if strings.TrimSpace(string(secretKey)) == "" {
		return errors.New("spaces secret key is missing — add it in the DigitalOcean integration settings")
	}

	spec := DeleteObjectSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.Bucket) == "" {
		return errors.New("bucket is required")
	}

	if _, _, err := parseBucket(spec.Bucket); err != nil {
		return err
	}

	if strings.TrimSpace(spec.FilePath) == "" {
		return errors.New("filePath is required")
	}

	return nil
}

func (d *DeleteObject) Execute(ctx core.ExecutionContext) error {
	spec := DeleteObjectSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	region, bucketName, err := parseBucket(spec.Bucket)
	if err != nil {
		return fmt.Errorf("invalid bucket configuration: %v", err)
	}

	client, err := NewSpacesClient(ctx.HTTP, ctx.Integration, region)
	if err != nil {
		return fmt.Errorf("error creating spaces client: %v", err)
	}

	if err := client.DeleteObject(bucketName, spec.FilePath); err != nil {
		return fmt.Errorf("failed to delete object: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.spaces.object.deleted",
		[]any{map[string]any{
			"bucket":   bucketName,
			"filePath": spec.FilePath,
			"deleted":  true,
		}},
	)
}

func (d *DeleteObject) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteObject) Actions() []core.Action {
	return []core.Action{}
}

func (d *DeleteObject) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (d *DeleteObject) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteObject) Cleanup(ctx core.SetupContext) error {
	return nil
}
