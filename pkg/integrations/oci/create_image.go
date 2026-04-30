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
	CompartmentID   string `json:"compartmentId" mapstructure:"compartmentId"`
	InstanceID      string `json:"instanceId" mapstructure:"instanceId"`
	DisplayName     string `json:"displayName" mapstructure:"displayName"`
	SourceImageType string `json:"sourceImageType" mapstructure:"sourceImageType"`
	SourceURI       string `json:"sourceUri" mapstructure:"sourceUri"`
	NamespaceName   string `json:"namespaceName" mapstructure:"namespaceName"`
	BucketName      string `json:"bucketName" mapstructure:"bucketName"`
	ObjectName      string `json:"objectName" mapstructure:"objectName"`
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
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     createImageSourceInstance,
			Description: "Choose whether to create from an instance or import an exported image",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Instance", Value: createImageSourceInstance},
						{Label: "Object Storage URL", Value: createImageSourceObjectStorageURI},
						{Label: "Object Storage Object", Value: createImageSourceObjectStorageObject},
					},
				},
			},
		},
		compartmentField(),
		displayNameField(true),
		{
			Name:        "instanceId",
			Label:       "Instance OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "OCID of the source instance",
			Placeholder: "ocid1.instance.oc1..",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{createImageSourceInstance}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "sourceType", Values: []string{createImageSourceInstance}},
			},
		},
		{
			Name:        "sourceImageType",
			Label:       "Source Image Type",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Format of the exported image in Object Storage",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageURI, createImageSourceObjectStorageObject}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageURI, createImageSourceObjectStorageObject}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "QCOW2", Value: "QCOW2"},
						{Label: "VMDK", Value: "VMDK"},
						{Label: "OCI", Value: "OCI"},
					},
				},
			},
		},
		{
			Name:        "sourceUri",
			Label:       "Source URI",
			Type:        configuration.FieldTypeString,
			Required:    true,
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
			Name:        "namespaceName",
			Label:       "Namespace",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Object Storage namespace",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageObject}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageObject}},
			},
		},
		{
			Name:        "bucketName",
			Label:       "Bucket",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Object Storage bucket name",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageObject}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageObject}},
			},
		},
		{
			Name:        "objectName",
			Label:       "Object",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Object name for the exported image",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageObject}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "sourceType", Values: []string{createImageSourceObjectStorageObject}},
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

	return ctx.Metadata.Set(imageNodeMetadata{
		CompartmentID: config.CompartmentID,
		DisplayName:   config.DisplayName,
	})
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
	config.InstanceID = strings.TrimSpace(config.InstanceID)
	config.DisplayName = strings.TrimSpace(config.DisplayName)
	config.SourceImageType = strings.TrimSpace(config.SourceImageType)
	config.SourceURI = strings.TrimSpace(config.SourceURI)
	config.NamespaceName = strings.TrimSpace(config.NamespaceName)
	config.BucketName = strings.TrimSpace(config.BucketName)
	config.ObjectName = strings.TrimSpace(config.ObjectName)

	return config, nil
}

func validateCreateImageConfiguration(config CreateImageConfiguration) error {
	if config.CompartmentID == "" {
		return errors.New("compartmentId is required")
	}
	if config.DisplayName == "" {
		return errors.New("displayName is required")
	}

	switch config.SourceType {
	case createImageSourceInstance:
		if config.InstanceID == "" {
			return errors.New("instanceId is required")
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
			return errors.New("namespaceName is required")
		}
		if config.BucketName == "" {
			return errors.New("bucketName is required")
		}
		if config.ObjectName == "" {
			return errors.New("objectName is required")
		}
	default:
		return fmt.Errorf("unsupported sourceType: %s", config.SourceType)
	}

	return nil
}

func createImageRequest(config CreateImageConfiguration) CreateImageRequest {
	req := CreateImageRequest{
		CompartmentID: config.CompartmentID,
		DisplayName:   config.DisplayName,
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
