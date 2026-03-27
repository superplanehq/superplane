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

type CopyObject struct{}

type CopyObjectSpec struct {
	SourceBucket        string `mapstructure:"sourceBucket"`
	SourceFilePath      string `mapstructure:"sourceFilePath"`
	DestinationBucket   string `mapstructure:"destinationBucket"`
	DestinationFilePath string `mapstructure:"destinationFilePath"`
	ACL                 string `mapstructure:"acl"`
	DeleteSource        bool   `mapstructure:"deleteSource"`
}

func (c *CopyObject) Name() string {
	return "digitalocean.copyObject"
}

func (c *CopyObject) Label() string {
	return "Copy Object"
}

func (c *CopyObject) Description() string {
	return "Copy or move an object within DigitalOcean Spaces Object Storage"
}

func (c *CopyObject) Documentation() string {
	return `The Copy Object component copies an object from one location to another within DigitalOcean Spaces. When **Delete Source** is enabled, the source object is deleted after a successful copy, effectively moving the object.

## Prerequisites

Spaces Access Key ID and Secret Access Key must be configured in the DigitalOcean integration settings.

## Configuration

- **Source Bucket**: The bucket containing the object to copy
- **Source File Path**: The path to the source object (e.g. reports/daily.csv). Supports expressions.
- **Destination Bucket**: The bucket to copy the object into (can be the same bucket)
- **Destination File Path**: The path for the copied object (e.g. archive/2026/daily.csv). Supports expressions.
- **Visibility**: Access control for the destination object — Private (default) or Public
- **Delete Source**: When enabled, the source object is deleted after a successful copy (move operation)

## Output

- **sourceBucket**: The source bucket name
- **sourceFilePath**: The source object path
- **destinationBucket**: The destination bucket name
- **destinationFilePath**: The destination object path
- **endpoint**: The full Spaces URL of the copied object
- **eTag**: MD5 hash of the copied object
- **moved**: ` + "`true`" + ` if the source was deleted, ` + "`false`" + ` otherwise

## Notes

- Both buckets must be in the same region
- Copying an object to the same path overwrites it
- Metadata and tags are copied from the source object by default

## Use Cases

- **Archiving**: Move processed files to an archive bucket or folder
- **Promotion**: Copy an artifact from a staging bucket to production
- **Backup**: Duplicate a file before modifying it
- **Renaming**: Move a file to a new path within the same bucket`
}

func (c *CopyObject) Icon() string {
	return "copy"
}

func (c *CopyObject) Color() string {
	return "blue"
}

func (c *CopyObject) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CopyObject) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "sourceBucket",
			Label:       "Source Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The bucket containing the object to copy",
			Placeholder: "Select a bucket",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "spaces_bucket",
				},
			},
		},
		{
			Name:        "sourceFilePath",
			Label:       "Source File Path",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Full path including file name and extension. Use / to organize into folders (e.g. reports/2024/daily.csv)",
			Placeholder: "reports/daily.csv",
		},
		{
			Name:        "destinationBucket",
			Label:       "Destination Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The bucket to copy the object into",
			Placeholder: "Select a bucket",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "spaces_bucket",
				},
			},
		},
		{
			Name:        "destinationFilePath",
			Label:       "Destination File Path",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Full path including file name and extension. Use / to organize into folders (e.g. archive/2026/daily.csv)",
			Placeholder: "archive/2026/daily.csv",
		},
		{
			Name:        "acl",
			Label:       "Visibility",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "private",
			Description: "Controls who can access the copied object",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Private", Value: "private"},
						{Label: "Public", Value: "public-read"},
					},
				},
			},
		},
		{
			Name:        "deleteSource",
			Label:       "Delete Source",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     "false",
			Description: "Delete the source object after a successful copy (move operation)",
		},
	}
}

func (c *CopyObject) Setup(ctx core.SetupContext) error {
	accessKey, _ := ctx.Integration.GetConfig("spacesAccessKey")
	if strings.TrimSpace(string(accessKey)) == "" {
		return errors.New("spaces access key is missing — add it in the DigitalOcean integration settings")
	}

	secretKey, _ := ctx.Integration.GetConfig("spacesSecretKey")
	if strings.TrimSpace(string(secretKey)) == "" {
		return errors.New("spaces secret key is missing — add it in the DigitalOcean integration settings")
	}

	spec := CopyObjectSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.SourceBucket) == "" {
		return errors.New("sourceBucket is required")
	}

	if _, _, err := parseBucket(spec.SourceBucket); err != nil {
		return fmt.Errorf("invalid sourceBucket: %w", err)
	}

	if strings.TrimSpace(spec.SourceFilePath) == "" {
		return errors.New("sourceFilePath is required")
	}

	if strings.TrimSpace(spec.DestinationBucket) == "" {
		return errors.New("destinationBucket is required")
	}

	if _, _, err := parseBucket(spec.DestinationBucket); err != nil {
		return fmt.Errorf("invalid destinationBucket: %w", err)
	}

	if strings.TrimSpace(spec.DestinationFilePath) == "" {
		return errors.New("destinationFilePath is required")
	}

	return nil
}

func (c *CopyObject) Execute(ctx core.ExecutionContext) error {
	spec := CopyObjectSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	sourceRegion, sourceBucketName, err := parseBucket(spec.SourceBucket)
	if err != nil {
		return fmt.Errorf("invalid source bucket: %v", err)
	}

	destRegion, destBucketName, err := parseBucket(spec.DestinationBucket)
	if err != nil {
		return fmt.Errorf("invalid destination bucket: %v", err)
	}

	destClient, err := NewSpacesClient(ctx.HTTP, ctx.Integration, destRegion)
	if err != nil {
		return fmt.Errorf("error creating spaces client: %v", err)
	}

	eTag, err := destClient.CopyObject(sourceBucketName, spec.SourceFilePath, destBucketName, spec.DestinationFilePath, spec.ACL)
	if err != nil {
		return fmt.Errorf("failed to copy object: %v", err)
	}

	if spec.DeleteSource {
		sourceClient, err := NewSpacesClient(ctx.HTTP, ctx.Integration, sourceRegion)
		if err != nil {
			return fmt.Errorf("object copied but failed to create source client for deletion: %v", err)
		}

		if err := sourceClient.DeleteObject(sourceBucketName, spec.SourceFilePath); err != nil {
			return fmt.Errorf("object copied but failed to delete source: %v", err)
		}
	}

	endpoint := fmt.Sprintf("https://%s.%s.digitaloceanspaces.com/%s", destBucketName, destRegion, spec.DestinationFilePath)

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.spaces.object.copied",
		[]any{map[string]any{
			"sourceBucket":        sourceBucketName,
			"sourceFilePath":      spec.SourceFilePath,
			"destinationBucket":   destBucketName,
			"destinationFilePath": spec.DestinationFilePath,
			"endpoint":            endpoint,
			"eTag":                eTag,
			"moved":               spec.DeleteSource,
		}},
	)
}

func (c *CopyObject) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CopyObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CopyObject) Actions() []core.Action {
	return []core.Action{}
}

func (c *CopyObject) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CopyObject) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CopyObject) Cleanup(ctx core.SetupContext) error {
	return nil
}
