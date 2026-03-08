package cloudstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	uploadObjectPayloadType   = "gcp.cloudstorage.upload"
	uploadObjectOutputChannel = "default"
)

type UploadObject struct{}

type UploadObjectConfiguration struct {
	Bucket      string `json:"bucket" mapstructure:"bucket"`
	Object      string `json:"object" mapstructure:"object"`
	Content     string `json:"content" mapstructure:"content"`
	ContentType string `json:"contentType" mapstructure:"contentType"`
}

type UploadObjectMetadata struct {
	Bucket string `json:"bucket" mapstructure:"bucket"`
	Object string `json:"object" mapstructure:"object"`
}

func (c *UploadObject) Name() string {
	return "gcp.cloudstorage.uploadObject"
}

func (c *UploadObject) Label() string {
	return "Cloud Storage • Upload Object"
}

func (c *UploadObject) Description() string {
	return "Upload an object to a Google Cloud Storage bucket"
}

func (c *UploadObject) Documentation() string {
	return `Uploads content as an object to a Google Cloud Storage bucket.

## Configuration

- **Bucket** (required): The Cloud Storage bucket to upload to.
- **Object** (required): The destination path within the bucket (e.g. ` + "`data/report.csv`" + `).
- **Content** (required): The content to upload.
- **Content Type**: MIME type for the uploaded object (e.g. ` + "`application/json`" + `, ` + "`text/plain`" + `). Defaults to ` + "`application/octet-stream`" + `.

## Required IAM roles

The service account used by the integration must have ` + "`roles/storage.objectCreator`" + ` (or ` + "`roles/storage.objectAdmin`" + `) on the bucket or project.

## Output

The uploaded object metadata, including:
- ` + "`name`" + `: Object name (path within the bucket).
- ` + "`bucket`" + `: Bucket name.
- ` + "`size`" + `: Object size in bytes.
- ` + "`contentType`" + `: MIME type of the object.
- ` + "`timeCreated`" + `: Timestamp when the object was created.
- ` + "`storageClass`" + `: Storage class of the object.
- ` + "`md5Hash`" + `: MD5 hash of the uploaded content.`
}

func (c *UploadObject) Icon() string  { return "gcp" }
func (c *UploadObject) Color() string { return "gray" }

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
			Description: "Select the Cloud Storage bucket to upload to.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeBucket,
					Parameters: []configuration.ParameterRef{},
				},
			},
		},
		{
			Name:        "object",
			Label:       "Object Path",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Destination path within the bucket (e.g. data/report.csv).",
			Placeholder: "path/to/object.json",
		},
		{
			Name:        "content",
			Label:       "Content",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The content to upload to Cloud Storage.",
		},
		{
			Name:        "contentType",
			Label:       "Content Type",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "MIME type for the uploaded object. Defaults to application/octet-stream.",
			Placeholder: "application/json",
			Default:     "application/octet-stream",
		},
	}
}

func decodeUploadObjectConfiguration(raw any) (UploadObjectConfiguration, error) {
	var config UploadObjectConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return UploadObjectConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.Bucket = strings.TrimSpace(config.Bucket)
	config.Object = strings.TrimSpace(config.Object)
	config.ContentType = strings.TrimSpace(config.ContentType)
	return config, nil
}

func (c *UploadObject) Setup(ctx core.SetupContext) error {
	config, err := decodeUploadObjectConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if config.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	if config.Object == "" {
		return fmt.Errorf("object path is required")
	}

	return ctx.Metadata.Set(UploadObjectMetadata{
		Bucket: config.Bucket,
		Object: config.Object,
	})
}

func (c *UploadObject) Execute(ctx core.ExecutionContext) error {
	config, err := decodeUploadObjectConfiguration(ctx.Configuration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	contentType := config.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	reqCtx := context.Background()
	uploadURL := fmt.Sprintf(
		"%s/b/%s/o?uploadType=media&name=%s",
		uploadBaseURL,
		url.PathEscape(config.Bucket),
		url.QueryEscape(config.Object),
	)

	body := strings.NewReader(config.Content)
	data, err := client.ExecRequest(reqCtx, http.MethodPost, uploadURL, body)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to upload object: %v", err))
	}

	var objectMeta map[string]any
	if err := json.Unmarshal(data, &objectMeta); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse upload response: %v", err))
	}

	output := map[string]any{
		"name":         objectMeta["name"],
		"bucket":       objectMeta["bucket"],
		"size":         objectMeta["size"],
		"contentType":  objectMeta["contentType"],
		"timeCreated":  objectMeta["timeCreated"],
		"updated":      objectMeta["updated"],
		"storageClass": objectMeta["storageClass"],
		"md5Hash":      objectMeta["md5Hash"],
		"generation":   objectMeta["generation"],
		"selfLink":     objectMeta["selfLink"],
	}

	return ctx.ExecutionState.Emit(uploadObjectOutputChannel, uploadObjectPayloadType, []any{output})
}

func (c *UploadObject) Actions() []core.Action                  { return nil }
func (c *UploadObject) HandleAction(_ core.ActionContext) error { return nil }
func (c *UploadObject) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
func (c *UploadObject) Cancel(_ core.ExecutionContext) error { return nil }
func (c *UploadObject) Cleanup(_ core.SetupContext) error    { return nil }
func (c *UploadObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
