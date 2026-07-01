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

// locationOptions lists the most common Cloud Storage locations: the three
// multi-regions, then a representative set of regional locations. Cloud Storage
// has no per-project "list locations" API, so these are the well-known location
// values from the platform.
var locationOptions = []configuration.FieldOption{
	{Label: "US (multi-region)", Value: "US"},
	{Label: "EU (multi-region)", Value: "EU"},
	{Label: "ASIA (multi-region)", Value: "ASIA"},
	{Label: "us-central1 (Iowa)", Value: "US-CENTRAL1"},
	{Label: "us-east1 (South Carolina)", Value: "US-EAST1"},
	{Label: "us-east4 (Northern Virginia)", Value: "US-EAST4"},
	{Label: "us-west1 (Oregon)", Value: "US-WEST1"},
	{Label: "us-west2 (Los Angeles)", Value: "US-WEST2"},
	{Label: "europe-west1 (Belgium)", Value: "EUROPE-WEST1"},
	{Label: "europe-west2 (London)", Value: "EUROPE-WEST2"},
	{Label: "europe-west3 (Frankfurt)", Value: "EUROPE-WEST3"},
	{Label: "europe-north1 (Finland)", Value: "EUROPE-NORTH1"},
	{Label: "asia-east1 (Taiwan)", Value: "ASIA-EAST1"},
	{Label: "asia-northeast1 (Tokyo)", Value: "ASIA-NORTHEAST1"},
	{Label: "asia-south1 (Mumbai)", Value: "ASIA-SOUTH1"},
	{Label: "asia-southeast1 (Singapore)", Value: "ASIA-SOUTHEAST1"},
	{Label: "australia-southeast1 (Sydney)", Value: "AUSTRALIA-SOUTHEAST1"},
	{Label: "southamerica-east1 (São Paulo)", Value: "SOUTHAMERICA-EAST1"},
}

var storageClassOptions = []configuration.FieldOption{
	{Label: "Standard", Value: "STANDARD"},
	{Label: "Nearline", Value: "NEARLINE"},
	{Label: "Coldline", Value: "COLDLINE"},
	{Label: "Archive", Value: "ARCHIVE"},
}

type CreateBucket struct{}

type CreateBucketSpec struct {
	Name                     string       `json:"name" mapstructure:"name"`
	Location                 string       `json:"location" mapstructure:"location"`
	StorageClass             string       `json:"storageClass" mapstructure:"storageClass"`
	UniformBucketLevelAccess *bool        `json:"uniformBucketLevelAccess" mapstructure:"uniformBucketLevelAccess"`
	Versioning               *bool        `json:"versioning" mapstructure:"versioning"`
	Labels                   []labelEntry `json:"labels" mapstructure:"labels"`
}

type labelEntry struct {
	Key   string `json:"key" mapstructure:"key"`
	Value string `json:"value" mapstructure:"value"`
}

func (c *CreateBucket) Name() string {
	return "gcp.storage.createBucket"
}

func (c *CreateBucket) Label() string {
	return "Cloud Storage • Create Bucket"
}

func (c *CreateBucket) Description() string {
	return "Create a Cloud Storage bucket"
}

func (c *CreateBucket) Documentation() string {
	return `The Create Bucket component creates a new Cloud Storage bucket.

## Use Cases

- **Environment setup**: Provision a bucket for artifacts, backups, or static assets as part of an environment
- **Ephemeral environments**: Create a dedicated bucket for a preview or test run
- **Infrastructure automation**: Provision storage as part of a broader workflow

## Configuration

- **Name**: The globally unique bucket name (required)
- **Location**: The location to create the bucket in — a multi-region (` + "`US`" + `, ` + "`EU`" + `, ` + "`ASIA`" + `) or a specific region (required)
- **Storage Class**: The default storage class for objects (Standard, Nearline, Coldline, Archive)
- **Uniform Bucket-Level Access**: Use IAM-only access control instead of per-object ACLs (recommended, default on)
- **Object Versioning**: Keep noncurrent versions of objects when they are overwritten or deleted (optional)
- **Labels**: Key-value labels applied to the bucket (optional)

## Output

Emits a ` + "`gcp.storage.bucket`" + ` payload with the bucket's ` + "`name`" + `, ` + "`location`" + `, ` + "`locationType`" + `, ` + "`storageClass`" + `, ` + "`timeCreated`" + `, ` + "`selfLink`" + `, and a ` + "`consoleUrl`" + ` for opening the bucket in the Cloud Console.

## Important Notes

- **Bucket names are globally unique across all of Cloud Storage** and must follow the [naming guidelines](https://cloud.google.com/storage/docs/buckets#naming) (3–63 characters, lowercase letters, numbers, hyphens, underscores, and dots).
- A bucket's location is permanent and cannot be changed after creation.
- Requires the ` + "`roles/storage.admin`" + ` IAM role, and the **Cloud Storage API** (` + "`storage.googleapis.com`" + `) enabled.`
}

func (c *CreateBucket) Icon() string {
	return "database"
}

func (c *CreateBucket) Color() string {
	return "blue"
}

func (c *CreateBucket) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateBucket) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The globally unique name for the new bucket",
			Placeholder: "my-bucket",
		},
		{
			Name:        "location",
			Label:       "Location",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "US",
			Description: "The location to create the bucket in (multi-region or region). This is permanent.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: locationOptions},
			},
		},
		{
			Name:        "storageClass",
			Label:       "Storage Class",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "STANDARD",
			Description: "The default storage class for objects in the bucket",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: storageClassOptions},
			},
		},
		{
			Name:        "uniformBucketLevelAccess",
			Label:       "Uniform Bucket-Level Access",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Use IAM exclusively for access control instead of per-object ACLs (recommended)",
		},
		{
			Name:        "versioning",
			Label:       "Object Versioning",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Keep noncurrent versions of objects when they are overwritten or deleted",
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Key-value labels for the bucket (billing, environment, team)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "key",
								Label:       "Key",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Label key (e.g. env, team, cost-center)",
								Placeholder: "e.g. env",
							},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Description: "Label value",
								Placeholder: "e.g. production",
							},
						},
					},
				},
			},
		},
	}
}

// labelsMap converts label entries to the labels map, skipping blank keys.
func labelsMap(entries []labelEntry) map[string]string {
	labels := make(map[string]string, len(entries))
	for _, e := range entries {
		key := strings.TrimSpace(e.Key)
		if key == "" {
			continue
		}
		labels[key] = strings.TrimSpace(e.Value)
	}
	return labels
}

func (c *CreateBucket) Setup(ctx core.SetupContext) error {
	spec := CreateBucketSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if strings.TrimSpace(spec.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(spec.Location) == "" {
		return fmt.Errorf("location is required")
	}
	return ctx.Metadata.Set(BucketNodeMetadata{Bucket: strings.TrimSpace(spec.Name)})
}

func (c *CreateBucket) Execute(ctx core.ExecutionContext) error {
	spec := CreateBucketSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return ctx.ExecutionState.Fail("error", "name is required")
	}
	location := strings.TrimSpace(spec.Location)
	if location == "" {
		return ctx.ExecutionState.Fail("error", "location is required")
	}

	body := map[string]any{
		"name":     name,
		"location": location,
	}
	if storageClass := strings.TrimSpace(spec.StorageClass); storageClass != "" {
		body["storageClass"] = storageClass
	}
	// Uniform bucket-level access defaults to on, matching the recommended setup.
	if spec.UniformBucketLevelAccess == nil || *spec.UniformBucketLevelAccess {
		body["iamConfiguration"] = map[string]any{
			"uniformBucketLevelAccess": map[string]any{"enabled": true},
		}
	}
	if spec.Versioning != nil {
		body["versioning"] = map[string]any{"enabled": *spec.Versioning}
	}
	if labels := labelsMap(spec.Labels); len(labels) > 0 {
		body["labels"] = labels
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	bucket, err := createBucket(context.Background(), client, client.ProjectID(), body)
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to create bucket", err, roleHintAdmin))
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, bucketPayloadType, []any{bucketPayload(bucket)})
}

func (c *CreateBucket) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateBucket) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateBucket) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateBucket) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateBucket) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateBucket) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
