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
	getObjectPayloadType   = "gcp.cloudstorage.getObject"
	getObjectOutputChannel = "default"
)

type GetObject struct{}

type GetObjectConfiguration struct {
	Bucket string `json:"bucket" mapstructure:"bucket"`
	Object string `json:"object" mapstructure:"object"`
}

type GetObjectMetadata struct {
	Bucket string `json:"bucket" mapstructure:"bucket"`
	Object string `json:"object" mapstructure:"object"`
}

func (c *GetObject) Name() string {
	return "gcp.cloudstorage.getObject"
}

func (c *GetObject) Label() string {
	return "Cloud Storage • Get Object"
}

func (c *GetObject) Description() string {
	return "Retrieve metadata for an object in a Google Cloud Storage bucket"
}

func (c *GetObject) Documentation() string {
	return `Retrieves metadata for a specific object in a Google Cloud Storage bucket.

## Configuration

- **Bucket** (required): The Cloud Storage bucket containing the object. Select from the list of buckets in your project.
- **Object** (required): The name (path) of the object to retrieve, e.g. ` + "`folder/file.txt`" + `.

## Required IAM roles

The service account used by the integration must have ` + "`roles/storage.objectViewer`" + ` (or ` + "`roles/storage.admin`" + `) on the bucket or project.

## Output

The object metadata, including:
- ` + "`bucket`" + `: Bucket name.
- ` + "`name`" + `: Object name.
- ` + "`size`" + `: Object size in bytes.
- ` + "`contentType`" + `: MIME type.
- ` + "`updated`" + `: Last modification time.
- ` + "`md5Hash`" + `: MD5 hash of the object data.
- ` + "`selfLink`" + `: URL for the object metadata.`
}

func (c *GetObject) Icon() string  { return "gcp" }
func (c *GetObject) Color() string { return "gray" }

func (c *GetObject) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetObject) Configuration() []configuration.Field {
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
			Description: "The name (path) of the object to retrieve.",
			Placeholder: "e.g. folder/file.txt",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bucket", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "bucket", Values: []string{"*"}},
			},
		},
	}
}

func decodeGetObjectConfiguration(raw any) (GetObjectConfiguration, error) {
	var config GetObjectConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return GetObjectConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.Bucket = strings.TrimSpace(config.Bucket)
	config.Object = strings.TrimSpace(config.Object)
	return config, nil
}

func (c *GetObject) Setup(ctx core.SetupContext) error {
	config, err := decodeGetObjectConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if config.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	if config.Object == "" {
		return fmt.Errorf("object is required")
	}

	return ctx.Metadata.Set(GetObjectMetadata{
		Bucket: config.Bucket,
		Object: config.Object,
	})
}

func (c *GetObject) Execute(ctx core.ExecutionContext) error {
	config, err := decodeGetObjectConfiguration(ctx.Configuration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	objectURL := fmt.Sprintf("%s/b/%s/o/%s",
		storageBaseURL,
		url.PathEscape(config.Bucket),
		url.PathEscape(config.Object),
	)

	data, err := client.GetURL(context.Background(), objectURL)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get object: %v", err))
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse object metadata: %v", err))
	}

	return ctx.ExecutionState.Emit(getObjectOutputChannel, getObjectPayloadType, []any{result})
}

func (c *GetObject) Actions() []core.Action                  { return nil }
func (c *GetObject) HandleAction(_ core.ActionContext) error { return nil }
func (c *GetObject) Cancel(_ core.ExecutionContext) error    { return nil }
func (c *GetObject) Cleanup(_ core.SetupContext) error       { return nil }
func (c *GetObject) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
func (c *GetObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
