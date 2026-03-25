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

// parseBucket splits a bucket value of the form "region/bucket-name"
// into its region and bucket name components.
func parseBucket(bucket string) (region, name string, err error) {
	parts := strings.SplitN(bucket, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid bucket value %q: expected format region/bucket-name", bucket)
	}
	return parts[0], parts[1], nil
}

const (
	channelFound    = "found"
	channelNotFound = "notFound"
)

type GetObject struct{}

type GetObjectSpec struct {
	Bucket      string `mapstructure:"bucket"`
	FilePath    string `mapstructure:"filePath"`
	IncludeBody bool   `mapstructure:"includeBody"`
}

func (g *GetObject) Name() string {
	return "digitalocean.getObject"
}

func (g *GetObject) Label() string {
	return "Get Object"
}

func (g *GetObject) Description() string {
	return "Retrieve an object and its metadata from DigitalOcean Spaces Object Storage"
}

func (g *GetObject) Documentation() string {
	return `The Get Object component retrieves an object and its metadata from a DigitalOcean Spaces bucket.

## Prerequisites

Spaces Access Key ID and Secret Access Key must be configured in the DigitalOcean integration settings. The integration works without them for other components (Droplets, DNS, etc.), but they are required for any Spaces operation.

## Configuration

- **Bucket**: The Spaces bucket to read from. The dropdown lists all buckets across all regions — the region is determined automatically.
- **File Path**: The path to the object within the bucket (e.g. reports/daily.csv)
- **Include Body**: Download the object content. Only supported for text-based content types (JSON, YAML, CSV, plain text, etc.)

## Output Channels

- **Found**: The object exists — metadata, tags, and optional body are returned
- **Not Found**: The object does not exist in the bucket (404)

## Output

- **bucket**: The bucket name
- **filePath**: The path to the object within the bucket
- **endpoint**: The full Spaces URL of the object
- **contentType**: MIME type of the object
- **size**: Human-readable file size (e.g. 1.23 MiB)
- **lastModified**: When the object was last modified (RFC1123 format)
- **eTag**: MD5 hash of the object content — changes when the file changes
- **metadata**: Custom metadata key-value pairs set on the object (x-amz-meta-* headers)
- **tags**: Key-value tags applied to the object
- **body**: Object content as a string (only present for text-based content types when Include Body is enabled)

## Use Cases

- **Config reading**: Fetch a JSON or YAML config file and use its contents in downstream workflow steps
- **File existence check**: Verify a file exists before triggering a process — route to notFound if it is missing
- **Change detection**: Compare ETag or LastModified with a previously stored value to detect updates
- **Backup verification**: Confirm a backup file was written today by checking LastModified
- **Tag-based routing**: Read object tags to drive workflow logic (e.g. status=ready)`
}

func (g *GetObject) Icon() string {
	return "file"
}

func (g *GetObject) Color() string {
	return "blue"
}

func (g *GetObject) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: channelFound, Label: "Found", Description: "The object exists"},
		{Name: channelNotFound, Label: "Not Found", Description: "The object does not exist"},
	}
}

func (g *GetObject) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "bucket",
			Label:       "Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Spaces bucket to read from",
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
			Description: "The path to the object within the bucket",
			Placeholder: "reports/daily.csv",
		},
		{
			Name:        "includeBody",
			Label:       "Include Body",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     "false",
			Description: "Download the object content. Only available for text-based content types (JSON, YAML, plain text, etc.).",
		},
	}
}

func (g *GetObject) Setup(ctx core.SetupContext) error {
	// Check integration credentials first — this way the node immediately
	// shows a configuration error if Spaces keys are missing, even before
	// the user has filled in the rest of the node configuration.
	accessKey, _ := ctx.Integration.GetConfig("spacesAccessKey")
	if strings.TrimSpace(string(accessKey)) == "" {
		return errors.New("spaces access key is missing — add it in the DigitalOcean integration settings")
	}

	secretKey, _ := ctx.Integration.GetConfig("spacesSecretKey")
	if strings.TrimSpace(string(secretKey)) == "" {
		return errors.New("spaces secret key is missing — add it in the DigitalOcean integration settings")
	}

	spec := GetObjectSpec{}
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

func (g *GetObject) Execute(ctx core.ExecutionContext) error {
	spec := GetObjectSpec{}
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

	result, err := client.GetObject(bucketName, spec.FilePath, spec.IncludeBody)
	if err != nil {
		return fmt.Errorf("failed to get object: %v", err)
	}

	if result.NotFound {
		return ctx.ExecutionState.Emit(
			channelNotFound,
			"digitalocean.spaces.object.not_found",
			[]any{map[string]any{
				"bucket":   bucketName,
				"filePath": spec.FilePath,
			}},
		)
	}

	tags, err := client.GetObjectTagging(bucketName, spec.FilePath)
	if err != nil {
		return fmt.Errorf("failed to get object tags: %v", err)
	}

	endpoint := fmt.Sprintf("https://%s.%s.digitaloceanspaces.com/%s", bucketName, region, spec.FilePath)

	output := map[string]any{
		"bucket":       bucketName,
		"filePath":     spec.FilePath,
		"endpoint":     endpoint,
		"contentType":  result.ContentType,
		"size":         FormatSize(result.ContentLength),
		"lastModified": result.LastModified,
		"eTag":         result.ETag,
		"metadata":     result.Metadata,
		"tags":         tags,
	}

	if spec.IncludeBody && result.Body != nil && isReadableContentType(result.ContentType) {
		output["body"] = string(result.Body)
	}

	return ctx.ExecutionState.Emit(
		channelFound,
		"digitalocean.spaces.object.fetched",
		[]any{output},
	)
}

func (g *GetObject) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetObject) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetObject) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (g *GetObject) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetObject) Cleanup(ctx core.SetupContext) error {
	return nil
}
