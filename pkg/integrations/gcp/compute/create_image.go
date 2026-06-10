package compute

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	compute "google.golang.org/api/compute/v1"
)

const (
	ImageSourceDisk     = "disk"
	ImageSourceSnapshot = "snapshot"
	ImageSourceImage    = "image"
)

type CreateImage struct{}

type CreateImageSpec struct {
	Name             string       `mapstructure:"name"`
	SourceType       string       `mapstructure:"sourceType"`
	Region           string       `mapstructure:"region"`
	Zone             string       `mapstructure:"zone"`
	SourceDisk       string       `mapstructure:"sourceDisk"`
	SourceSnapshot   string       `mapstructure:"sourceSnapshot"`
	SourceImage      string       `mapstructure:"sourceImage"`
	Family           string       `mapstructure:"family"`
	Description      string       `mapstructure:"description"`
	StorageLocations string       `mapstructure:"storageLocations"`
	ForceCreate      bool         `mapstructure:"forceCreate"`
	Labels           []LabelEntry `mapstructure:"labels"`
}

func (c *CreateImage) Name() string {
	return "gcp.createImage"
}

func (c *CreateImage) Label() string {
	return "Compute • Create Image"
}

func (c *CreateImage) Description() string {
	return "Create a Google Compute Engine custom image from a disk, snapshot, or another image"
}

func (c *CreateImage) Documentation() string {
	return `The Create Image component creates a custom Compute Engine image.

## Use Cases

- **Golden image pipelines**: Build immutable, reusable images from validated disks
- **Backup workflows**: Capture disk state as a restorable image before changes
- **Release automation**: Produce versioned images as part of CI/CD

## Configuration

- **Image Name**: Name for the new image (lowercase, numbers, hyphens; 1–63 chars).
- **Source**: Where the image is created from:
  - **Disk**: A persistent disk (pick the region, zone, then the disk).
  - **Snapshot**: A disk snapshot.
  - **Image**: Another custom image in the project.
- **Image family**: Optional family to group related images (e.g. ` + "`my-app`" + `).
- **Description**: Optional human-readable description.
- **Labels**: Optional key-value labels (billing, environment, team).
- **Storage location**: Optional single region or multi-region to store the image (e.g. ` + "`us`" + ` or ` + "`europe-west1`" + `). Defaults to the source's region.
- **Force create**: When the source is a disk attached to a running instance, create the image anyway (may produce an inconsistent image).

## Output

Emits the created image: name, selfLink, family, status, diskSizeGb, sourceDisk, labels, deprecationState, creationTimestamp.

## Important Notes

- Creating an image from a disk attached to a running VM is not recommended unless **Force create** is enabled.
- The component waits for the underlying global operation to complete before emitting.`
}

func (c *CreateImage) Icon() string {
	return "image"
}

func (c *CreateImage) Color() string {
	return "blue"
}

func (c *CreateImage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateImage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Image Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name for the new image. Start with a letter; use only a-z, 0-9, and hyphens; 1 to 63 characters.",
			Placeholder: "e.g. my-app-2026-06-02",
		},
		{
			Name:        "sourceType",
			Label:       "Source",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Where to create the image from.",
			Default:     ImageSourceDisk,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Disk", Value: ImageSourceDisk},
						{Label: "Snapshot", Value: ImageSourceSnapshot},
						{Label: "Image", Value: ImageSourceImage},
					},
				},
			},
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "GCP region of the source disk. Used to filter zones.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeRegion,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{ImageSourceDisk}},
			},
		},
		{
			Name:        "zone",
			Label:       "Zone",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "GCP zone of the source disk.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeZone,
					Parameters: []configuration.ParameterRef{
						{Name: "region", ValueFrom: &configuration.ParameterValueFrom{Field: "region"}},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{ImageSourceDisk}},
			},
		},
		{
			Name:        "sourceDisk",
			Label:       "Source disk",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "The persistent disk to create the image from.",
			Placeholder: "Select disk",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeDisks,
					Parameters: []configuration.ParameterRef{
						{Name: "zone", ValueFrom: &configuration.ParameterValueFrom{Field: "zone"}},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{ImageSourceDisk}},
			},
		},
		{
			Name:        "sourceSnapshot",
			Label:       "Source snapshot",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "The disk snapshot to create the image from.",
			Placeholder: "Select snapshot",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeSnapshots,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{ImageSourceSnapshot}},
			},
		},
		{
			Name:        "sourceImage",
			Label:       "Source image",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "An existing custom image to copy.",
			Placeholder: "Select image",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeCustomImages,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{ImageSourceImage}},
			},
		},
		{
			Name:        "family",
			Label:       "Image family",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional family to group related images.",
			Placeholder: "e.g. my-app",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "Optional image description",
		},
		{
			Name:        "storageLocations",
			Label:       "Storage location",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional Cloud Storage location (multi-region or region) to store the image. Defaults to the source's region.",
			Placeholder: "Select storage location",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeImageStorageLocation,
				},
			},
		},
		{
			Name:        "forceCreate",
			Label:       "Force create",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Create the image even if the source disk is attached to a running instance.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{ImageSourceDisk}},
			},
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Key-value labels for the image (billing, environment, team).",
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
								Description: "Label key (e.g. env, team, cost-center).",
								Placeholder: "e.g. env",
							},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Description: "Label value.",
								Placeholder: "e.g. production",
							},
						},
					},
				},
			},
		},
	}
}

func (c *CreateImage) Setup(ctx core.SetupContext) error {
	spec := CreateImageSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.Name) == "" {
		return errors.New("image name is required")
	}

	if err := validateImageSource(spec); err != nil {
		return err
	}

	return ctx.Metadata.Set(ImageNodeMetadata{ImageName: strings.TrimSpace(spec.Name)})
}

func validateImageSource(spec CreateImageSpec) error {
	sourceType := normalizeImageSourceType(spec.SourceType)
	switch sourceType {
	case ImageSourceDisk:
		disk := strings.TrimSpace(spec.SourceDisk)
		if disk == "" {
			return errors.New("source disk is required")
		}
		// A bare disk name (the form of an ID from the disk picker) needs a zone
		// to build the zone-qualified sourceDisk URL the images API requires. A
		// full path or selfLink already carries its location, so allow it as-is.
		if !strings.Contains(disk, "/") && strings.TrimSpace(spec.Zone) == "" {
			return errors.New("zone is required when selecting a source disk by name")
		}
	case ImageSourceSnapshot:
		if strings.TrimSpace(spec.SourceSnapshot) == "" {
			return errors.New("source snapshot is required")
		}
	case ImageSourceImage:
		if strings.TrimSpace(spec.SourceImage) == "" {
			return errors.New("source image is required")
		}
	default:
		return fmt.Errorf("invalid source type %q", spec.SourceType)
	}
	return nil
}

func normalizeImageSourceType(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ImageSourceDisk
	}
	return s
}

func (c *CreateImage) Execute(ctx core.ExecutionContext) error {
	spec := CreateImageSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return ctx.ExecutionState.Fail("error", "image name is required")
	}
	if err := validateImageSource(spec); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}
	project := client.ProjectID()

	image, err := buildImageFromSpec(project, spec)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	callCtx := context.Background()
	path := fmt.Sprintf("projects/%s/global/images", project)
	if normalizeImageSourceType(spec.SourceType) == ImageSourceDisk && spec.ForceCreate {
		path += "?forceCreate=true"
	}

	body, err := client.Post(callCtx, path, image)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create image: %v", err))
	}

	opName, err := operationNameFromResponse(body, "create image")
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	if err := WaitForGlobalOperation(callCtx, client, project, opName); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("error waiting for create image operation: %v", err))
	}

	imgBody, err := GetImage(callCtx, client, project, name)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read created image: %v", err))
	}
	payload, err := ImagePayloadFromGetResponse(imgBody)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse created image: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.compute.image.created",
		[]any{payload},
	)
}

func buildImageFromSpec(project string, spec CreateImageSpec) (*compute.Image, error) {
	image := &compute.Image{
		Name:        strings.TrimSpace(spec.Name),
		Family:      strings.TrimSpace(spec.Family),
		Description: strings.TrimSpace(spec.Description),
		Labels:      imageLabelsFromEntries(spec.Labels),
	}

	// The images API accepts at most one storage location. The field is a
	// single-select picker, so reject multiple values (e.g. a comma-separated
	// expression result) explicitly rather than silently dropping the extras.
	if locations := parseCommaList(spec.StorageLocations); len(locations) > 0 {
		if len(locations) > 1 {
			return nil, fmt.Errorf("only one storage location is supported, but %d were provided: %s", len(locations), strings.Join(locations, ", "))
		}
		image.StorageLocations = []string{locations[0]}
	}

	switch normalizeImageSourceType(spec.SourceType) {
	case ImageSourceDisk:
		image.SourceDisk = resolveDiskURL(project, strings.TrimSpace(spec.Zone), strings.TrimSpace(spec.SourceDisk))
	case ImageSourceSnapshot:
		image.SourceSnapshot = resolveSnapshotURL(project, strings.TrimSpace(spec.SourceSnapshot))
	case ImageSourceImage:
		image.SourceImage = resolveImageURL(project, strings.TrimSpace(spec.SourceImage))
	default:
		return nil, fmt.Errorf("invalid source type %q", spec.SourceType)
	}

	return image, nil
}

func parseCommaList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func (c *CreateImage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateImage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateImage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateImage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateImage) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateImage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
