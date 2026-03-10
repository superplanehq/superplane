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
	uploadObjectPayloadType   = "gcp.cloudstorage.uploadObject"
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
	return "Upload content to an object in a Google Cloud Storage bucket"
}

func (c *UploadObject) Documentation() string {
	return `Uploads content to a Google Cloud Storage object, creating or overwriting it.

## Configuration

- **Bucket** (required): The Cloud Storage bucket to upload to. Select from the list of buckets in your project.
- **Object** (required): The name (path) of the object to create or overwrite, e.g. ` + "`reports/output.json`" + `.
- **Content** (required): The content to upload as the object body.
- **Content Type**: MIME type for the uploaded object. Defaults to ` + "`application/octet-stream`" + `.

## Required IAM roles

The service account used by the integration must have ` + "`roles/storage.objectCreator`" + ` (or ` + "`roles/storage.admin`" + `) on the bucket or project.

## Output

The created object metadata, including:
- ` + "`bucket`" + `: Bucket name.
- ` + "`name`" + `: Object name.
- ` + "`size`" + `: Object size in bytes.
- ` + "`contentType`" + `: MIME type.
- ` + "`selfLink`" + `: URL for the object metadata.`
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
			Description: "Select the Cloud Storage bucket.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeBucket,
					Parameters: []configuration.ParameterRef{},
				},
			},
		},
		{
			Name:        "object",
			Label:       "Object",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The name (path) of the object to create or overwrite.",
			Placeholder: "e.g. reports/output.json",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bucket", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "bucket", Values: []string{"*"}},
			},
		},
		{
			Name:        "content",
			Label:       "Content",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The content to upload as the object body.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bucket", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "bucket", Values: []string{"*"}},
			},
		},
		{
			Name:        "contentType",
			Label:       "Content Type",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "MIME type for the uploaded object. Defaults to application/octet-stream.",
			Placeholder: "e.g. application/json",
			Default:     "application/octet-stream",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bucket", Values: []string{"*"}},
			},
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
		return fmt.Errorf("object is required")
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

	uploadURL := fmt.Sprintf("%s/b/%s/o?uploadType=media&name=%s&contentEncoding=identity",
		uploadBaseURL,
		url.PathEscape(config.Bucket),
		url.QueryEscape(config.Object),
	)

	data, err := client.PostURL(context.Background(), uploadURL, config.Content)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to upload object: %v", err))
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse upload response: %v", err))
	}

	return ctx.ExecutionState.Emit(uploadObjectOutputChannel, uploadObjectPayloadType, []any{result})
}

func (c *UploadObject) Actions() []core.Action                  { return nil }
func (c *UploadObject) HandleAction(_ core.ActionContext) error { return nil }
func (c *UploadObject) Cancel(_ core.ExecutionContext) error    { return nil }
func (c *UploadObject) Cleanup(_ core.SetupContext) error       { return nil }
func (c *UploadObject) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
func (c *UploadObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
