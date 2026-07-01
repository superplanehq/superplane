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

type UpdateImage struct{}

type UpdateImageConfiguration struct {
	ImageID     string `json:"image" mapstructure:"image"`
	DisplayName string `json:"displayName" mapstructure:"displayName"`
}

func (c *UpdateImage) Name() string        { return "oci.updateImage" }
func (c *UpdateImage) Label() string       { return "Update Image" }
func (c *UpdateImage) Description() string { return "Update an OCI custom image display name" }
func (c *UpdateImage) Icon() string        { return "oci" }
func (c *UpdateImage) Color() string       { return "red" }

func (c *UpdateImage) Documentation() string {
	return `The Update Image component updates the display name of an OCI Compute custom image. Oracle-provided platform images cannot be updated.

## Use Cases

- **Release promotion**: Rename an image after validation
- **Operational cleanup**: Standardize image names
- **Inventory hygiene**: Keep image metadata readable in automated workflows`
}

func (c *UpdateImage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateImage) ExampleOutput() map[string]any {
	return exampleOutputUpdateImage()
}

func (c *UpdateImage) Configuration() []configuration.Field {
	return []configuration.Field{
		customImageIDField(true),
		displayNameField(true),
	}
}

func (c *UpdateImage) Setup(ctx core.SetupContext) error {
	config, err := decodeUpdateImageConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateUpdateImageConfiguration(config); err != nil {
		return err
	}
	return ctx.Metadata.Set(resolveImageNodeMetadata(ctx, imageNodeMetadata{ImageID: config.ImageID, DisplayName: config.DisplayName}))
}

func (c *UpdateImage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return defaultProcessQueue(ctx)
}

func (c *UpdateImage) Execute(ctx core.ExecutionContext) error {
	config, err := decodeUpdateImageConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateUpdateImageConfiguration(config); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	if _, err := ensureCustomImage(client, config.ImageID); err != nil {
		return err
	}

	image, err := client.UpdateImage(config.ImageID, config.DisplayName)
	if err != nil {
		return fmt.Errorf("failed to update image: %w", err)
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

func (c *UpdateImage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return noWebhook(ctx)
}

func (c *UpdateImage) Cancel(ctx core.ExecutionContext) error { return nil }
func (c *UpdateImage) Cleanup(ctx core.SetupContext) error    { return nil }
func (c *UpdateImage) Hooks() []core.Hook                     { return []core.Hook{} }
func (c *UpdateImage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func decodeUpdateImageConfiguration(input any) (UpdateImageConfiguration, error) {
	var config UpdateImageConfiguration
	if err := mapstructure.WeakDecode(input, &config); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.ImageID = strings.TrimSpace(config.ImageID)
	config.DisplayName = strings.TrimSpace(config.DisplayName)
	return config, nil
}

func validateUpdateImageConfiguration(config UpdateImageConfiguration) error {
	if err := validateImageID(config.ImageID); err != nil {
		return err
	}
	if config.DisplayName == "" {
		return errors.New("displayName is required")
	}
	return nil
}
