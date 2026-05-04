package oci

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	createImageSourceInstance            = "instance"
	createImageSourceObjectStorageURI    = "objectStorageUri"
	createImageSourceObjectStorageObject = "objectStorageObject"
)

type CreateImage struct{}

type CreateImageConfiguration struct {
	SourceType      string `json:"sourceType" mapstructure:"sourceType"`
	CompartmentID   string `json:"compartment" mapstructure:"compartment"`
	Instance        string `json:"instance" mapstructure:"instance"`
	InstanceID      string `json:"instanceId" mapstructure:"instanceId"`
	DisplayName     string `json:"displayName" mapstructure:"displayName"`
	SourceImageType string `json:"sourceImageType" mapstructure:"sourceImageType"`
	SourceURI       string `json:"sourceUri" mapstructure:"sourceUri"`
	NamespaceName   string `json:"namespace" mapstructure:"namespace"`
	BucketName      string `json:"bucket" mapstructure:"bucket"`
	ObjectName      string `json:"object" mapstructure:"object"`
	Tags            []Tag  `json:"tags" mapstructure:"tags"`
}

type Tag struct {
	Key   string `json:"key" mapstructure:"key"`
	Value string `json:"value" mapstructure:"value"`
}

func (c *CreateImage) Name() string {
	return "oci.createImage"
}

func (c *CreateImage) Label() string {
	return "Create Image"
}

func (c *CreateImage) Description() string {
	return "Create an OCI boot disk image from an instance or Object Storage export"
}

func (c *CreateImage) Documentation() string {
	return `The Create Image component creates an OCI Compute boot disk image.

## Use Cases

- **Golden image pipelines**: Bake reusable server images from configured instances
- **Environment promotion**: Import exported images from Object Storage into a target compartment
- **Backup workflows**: Capture a known-good instance image before a risky change

## Configuration

- **Source Type**: Create from an instance, import from an Object Storage URL, or import from a bucket object
- **Compartment**: Compartment where the image is created
- **Display Name**: Friendly name for the image
- **Instance OCID**: Source instance when creating from an instance
- **Source Image Type**: Image format for imported images, such as QCOW2, VMDK, or OCI
- **Namespace, Bucket, Object**: Object Storage location when importing from a bucket object
- **Tags**: Optional freeform tags to apply to the image

## Completion behavior

The component waits for the image to reach ` + "`AVAILABLE`" + ` before emitting.`
}

func (c *CreateImage) Icon() string  { return "oci" }
func (c *CreateImage) Color() string { return "red" }

func (c *CreateImage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateImage) ExampleOutput() map[string]any {
	return exampleOutputCreateImage()
}

func (c *CreateImage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "sourceType",
			Label:       "Source Type",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Default:     createImageSourceInstance,
			Description: "Choose whether to create from an instance or import an exported image",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeImageSource,
				},
			},
		},
		compartmentField(),
		displayNameField(true),
		{
			Name:        "instance",
			Label:       "Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "OCID of the source instance",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{createImageSourceInstance}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "sourceType", Values: []string{createImageSourceInstance}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstance,
					Parameters: []configuration.ParameterRef{
						{
							Name: "compartmentId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "compartment",
							},
						},
					},
				},
			},
		},
		{
			Name:        "sourceImageType",
			Label:       "Source Image Type",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Format of the exported image in Object Storage",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageURI, createImageSourceObjectStorageObject}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageURI, createImageSourceObjectStorageObject}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeSourceImageType,
				},
			},
		},
		{
			Name:        "sourceUri",
			Label:       "Source URI",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Object Storage URL or pre-authenticated request URL for the image",
			Placeholder: "https://objectstorage.region.oraclecloud.com/...",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageURI}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageURI}},
			},
		},
		{
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Object Storage namespace",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageObject}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageObject}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeObjectNamespace,
				},
			},
		},
		{
			Name:        "bucket",
			Label:       "Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Object Storage bucket name",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageObject}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageObject}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeObjectBucket,
					Parameters: []configuration.ParameterRef{
						{
							Name: "compartmentId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "compartment",
							},
						},
						{
							Name: "namespaceName",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "namespace",
							},
						},
					},
				},
			},
		},
		{
			Name:        "object",
			Label:       "Object",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Object name for the exported image",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageObject}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageObject}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeObject,
					Parameters: []configuration.ParameterRef{
						{
							Name: "namespaceName",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "namespace",
							},
						},
						{
							Name: "bucketName",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "bucket",
							},
						},
					},
				},
			},
		},
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Optional freeform tags to apply to the image",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "key",
								Label:    "Key",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "value",
								Label:    "Value",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
	}
}

func (c *CreateImage) Setup(ctx core.SetupContext) error {
	config, err := decodeCreateImageConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateCreateImageConfiguration(config); err != nil {
		return err
	}

	return ctx.Metadata.Set(resolveImageNodeMetadata(ctx, imageNodeMetadata{
		CompartmentID: config.CompartmentID,
		DisplayName:   config.DisplayName,
		InstanceID:    config.Instance,
		SourceType:    config.SourceType,
	}))
}

func (c *CreateImage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return defaultProcessQueue(ctx)
}

func (c *CreateImage) Execute(ctx core.ExecutionContext) error {
	config, err := decodeCreateImageConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateCreateImageConfiguration(config); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	image, err := client.CreateImage(createImageRequest(config))
	if err != nil {
		return fmt.Errorf("failed to create image: %w", err)
	}

	if err := ctx.Metadata.Set(imageExecutionMetadata{
		ImageID:       image.ID,
		CompartmentID: config.CompartmentID,
		DisplayName:   image.DisplayName,
		State:         image.LifecycleState,
		StartedAt:     time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createImagePollInterval)
}

func (c *CreateImage) Hooks() []core.Hook {
	return []core.Hook{{Name: "poll", Type: core.HookTypeInternal}}
}

func (c *CreateImage) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	return c.poll(ctx)
}

func (c *CreateImage) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata imageExecutionMetadata
	if err := mapstructure.WeakDecode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}
	if metadata.ImageID == "" {
		return errors.New("poll metadata is missing imageId: execution state may be corrupted")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	image, err := client.GetImage(metadata.ImageID)
	if err != nil {
		metadata.PollErrors++
		ctx.Logger.Warnf("failed to get image %s (attempt %d/%d): %v", metadata.ImageID, metadata.PollErrors, maxPollErrors, err)
		if metadata.PollErrors >= maxPollErrors {
			return fmt.Errorf("giving up polling image %s after %d consecutive errors: %w", metadata.ImageID, maxPollErrors, err)
		}
		if err := ctx.Metadata.Set(metadata); err != nil {
			return err
		}
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createImagePollInterval)
	}

	metadata.PollErrors = 0
	metadata.PollAttempts++
	metadata.State = image.LifecycleState
	metadata.DisplayName = image.DisplayName
	if metadata.CompartmentID == "" {
		metadata.CompartmentID = image.CompartmentID
	}
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	switch image.LifecycleState {
	case imageStateAvailable:
		return emitImage(ctx.ExecutionState, image)
	case imageStateDeleted, imageStateDisabled:
		return fmt.Errorf("image %s entered state %s unexpectedly", image.ID, image.LifecycleState)
	default:
		if metadata.PollAttempts >= createImageMaxPollAttempts {
			return fmt.Errorf("timed out waiting for image %s to reach AVAILABLE after %d poll attempts (state: %s)", image.ID, metadata.PollAttempts, image.LifecycleState)
		}
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createImagePollInterval)
	}
}

func (c *CreateImage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return noWebhook(ctx)
}

func (c *CreateImage) Cancel(ctx core.ExecutionContext) error { return nil }
func (c *CreateImage) Cleanup(ctx core.SetupContext) error    { return nil }

func decodeCreateImageConfiguration(input any) (CreateImageConfiguration, error) {
	var config CreateImageConfiguration
	if err := mapstructure.WeakDecode(input, &config); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.SourceType = strings.TrimSpace(config.SourceType)
	if config.SourceType == "" {
		config.SourceType = createImageSourceInstance
	}
	config.CompartmentID = strings.TrimSpace(config.CompartmentID)
	config.Instance = strings.TrimSpace(config.Instance)
	config.InstanceID = strings.TrimSpace(config.InstanceID)
	if config.Instance == "" {
		config.Instance = config.InstanceID
	}
	config.InstanceID = config.Instance
	config.DisplayName = strings.TrimSpace(config.DisplayName)
	config.SourceImageType = strings.TrimSpace(config.SourceImageType)
	config.SourceURI = strings.TrimSpace(config.SourceURI)
	config.NamespaceName = strings.TrimSpace(config.NamespaceName)
	config.BucketName = strings.TrimSpace(config.BucketName)
	config.ObjectName = strings.TrimSpace(config.ObjectName)
	for i := range config.Tags {
		config.Tags[i].Key = strings.TrimSpace(config.Tags[i].Key)
		config.Tags[i].Value = strings.TrimSpace(config.Tags[i].Value)
	}

	return config, nil
}

func validateCreateImageConfiguration(config CreateImageConfiguration) error {
	if config.CompartmentID == "" {
		return errors.New("compartment is required")
	}
	if config.DisplayName == "" {
		return errors.New("displayName is required")
	}

	switch config.SourceType {
	case createImageSourceInstance:
		if config.InstanceID == "" {
			return errors.New("instance is required")
		}
	case createImageSourceObjectStorageURI:
		if config.SourceImageType == "" {
			return errors.New("sourceImageType is required")
		}
		if config.SourceURI == "" {
			return errors.New("sourceUri is required")
		}
	case createImageSourceObjectStorageObject:
		if config.SourceImageType == "" {
			return errors.New("sourceImageType is required")
		}
		if config.NamespaceName == "" {
			return errors.New("namespace is required")
		}
		if config.BucketName == "" {
			return errors.New("bucket is required")
		}
		if config.ObjectName == "" {
			return errors.New("object is required")
		}
	default:
		return fmt.Errorf("unsupported sourceType: %s", config.SourceType)
	}

	for _, tag := range config.Tags {
		if tag.Key == "" && tag.Value == "" {
			continue
		}
		if tag.Key == "" {
			return errors.New("tag key is required")
		}
		if tag.Value == "" {
			return errors.New("tag value is required")
		}
	}

	return nil
}

func createImageRequest(config CreateImageConfiguration) CreateImageRequest {
	req := CreateImageRequest{
		CompartmentID: config.CompartmentID,
		DisplayName:   config.DisplayName,
		FreeformTags:  tagsToMap(config.Tags),
	}

	switch config.SourceType {
	case createImageSourceInstance:
		req.InstanceID = config.InstanceID
	case createImageSourceObjectStorageURI:
		req.ImageSourceDetails = &ImageSourceDetails{
			SourceType:      "objectStorageUri",
			SourceImageType: config.SourceImageType,
			SourceURI:       config.SourceURI,
		}
	case createImageSourceObjectStorageObject:
		req.ImageSourceDetails = &ImageSourceDetails{
			SourceType:      "objectStorageTuple",
			SourceImageType: config.SourceImageType,
			NamespaceName:   config.NamespaceName,
			BucketName:      config.BucketName,
			ObjectName:      config.ObjectName,
		}
	}

	return req
}

func tagsToMap(tags []Tag) map[string]string {
	if len(tags) == 0 {
		return nil
	}

	out := map[string]string{}
	for _, tag := range tags {
		if tag.Key == "" && tag.Value == "" {
			continue
		}
		out[tag.Key] = tag.Value
	}
	if len(out) == 0 {
		return nil
	}

	return out
}
