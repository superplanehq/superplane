package oci

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetImage struct{}

type GetImageConfiguration struct {
	ImageID string `json:"image" mapstructure:"image"`
}

func (c *GetImage) Name() string        { return "oci.getImage" }
func (c *GetImage) Label() string       { return "Get Image" }
func (c *GetImage) Description() string { return "Get OCI custom image details by OCID" }
func (c *GetImage) Icon() string        { return "oci" }
func (c *GetImage) Color() string       { return "red" }

func (c *GetImage) Documentation() string {
	return `The Get Image component retrieves metadata for an OCI Compute custom image. Oracle-provided platform images are not returned by this component.

## Use Cases

- **Deployment checks**: Verify that an image is available before launching instances
- **Traceability**: Resolve image details by OCID
- **Audit workflows**: Capture operating system, launch mode, and lifecycle state metadata`
}

func (c *GetImage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetImage) ExampleOutput() map[string]any {
	return exampleOutputGetImage()
}

func (c *GetImage) Configuration() []configuration.Field {
	return []configuration.Field{customImageIDField(true)}
}

func (c *GetImage) Setup(ctx core.SetupContext) error {
	config, err := decodeGetImageConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateImageID(config.ImageID); err != nil {
		return err
	}
	return ctx.Metadata.Set(resolveImageNodeMetadata(ctx, imageNodeMetadata{ImageID: config.ImageID}))
}

func (c *GetImage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return defaultProcessQueue(ctx)
}

func (c *GetImage) Execute(ctx core.ExecutionContext) error {
	config, err := decodeGetImageConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateImageID(config.ImageID); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	image, err := ensureCustomImageForOperation(client, config.ImageID, "retrieved")
	if err != nil {
		return err
	}

	if err := ctx.Metadata.Set(imageExecutionMetadata{
		ImageID:       image.ID,
		CompartmentID: image.CompartmentID,
		DisplayName:   image.DisplayName,
		State:         image.LifecycleState,
		StartedAt:     time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return err
	}

	return emitImage(ctx.ExecutionState, image)
}

func (c *GetImage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return noWebhook(ctx)
}

func (c *GetImage) Cancel(ctx core.ExecutionContext) error { return nil }
func (c *GetImage) Cleanup(ctx core.SetupContext) error    { return nil }
func (c *GetImage) Hooks() []core.Hook                     { return []core.Hook{} }
func (c *GetImage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func decodeGetImageConfiguration(input any) (GetImageConfiguration, error) {
	var config GetImageConfiguration
	if err := mapstructure.WeakDecode(input, &config); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.ImageID = strings.TrimSpace(config.ImageID)
	return config, nil
}
