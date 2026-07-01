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

type DeleteImage struct{}

type DeleteImageConfiguration struct {
	ImageID string `json:"image" mapstructure:"image"`
}

func (c *DeleteImage) Name() string        { return "oci.deleteImage" }
func (c *DeleteImage) Label() string       { return "Delete Image" }
func (c *DeleteImage) Description() string { return "Delete an OCI custom image by OCID" }
func (c *DeleteImage) Icon() string        { return "oci" }
func (c *DeleteImage) Color() string       { return "red" }

func (c *DeleteImage) Documentation() string {
	return `The Delete Image component deletes an OCI Compute custom image. Oracle-provided platform images cannot be deleted.

## Use Cases

- **Cleanup workflows**: Remove obsolete custom images
- **Quota management**: Delete old images before creating new ones
- **Security response**: Retire vulnerable images from automated provisioning`
}

func (c *DeleteImage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteImage) ExampleOutput() map[string]any {
	return exampleOutputDeleteImage()
}

func (c *DeleteImage) Configuration() []configuration.Field {
	return []configuration.Field{customImageIDField(true)}
}

func (c *DeleteImage) Setup(ctx core.SetupContext) error {
	config, err := decodeDeleteImageConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateImageID(config.ImageID); err != nil {
		return err
	}
	return ctx.Metadata.Set(resolveImageNodeMetadata(ctx, imageNodeMetadata{ImageID: config.ImageID}))
}

func (c *DeleteImage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return defaultProcessQueue(ctx)
}

func (c *DeleteImage) Execute(ctx core.ExecutionContext) error {
	config, err := decodeDeleteImageConfiguration(ctx.Configuration)
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

	if _, err := ensureCustomImage(client, config.ImageID); err != nil {
		return err
	}

	startedAt := time.Now().UTC().Format(time.RFC3339)
	if err := client.DeleteImage(config.ImageID); err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	if err := ctx.Metadata.Set(imageExecutionMetadata{
		ImageID:   config.ImageID,
		State:     imageStateDeleted,
		StartedAt: startedAt,
	}); err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, imageDeletedPayloadType, []any{
		map[string]any{
			"imageId":   config.ImageID,
			"state":     imageStateDeleted,
			"deletedAt": startedAt,
		},
	})
}

func (c *DeleteImage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return noWebhook(ctx)
}

func (c *DeleteImage) Cancel(ctx core.ExecutionContext) error { return nil }
func (c *DeleteImage) Cleanup(ctx core.SetupContext) error    { return nil }
func (c *DeleteImage) Hooks() []core.Hook                     { return []core.Hook{} }
func (c *DeleteImage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func decodeDeleteImageConfiguration(input any) (DeleteImageConfiguration, error) {
	var config DeleteImageConfiguration
	if err := mapstructure.WeakDecode(input, &config); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.ImageID = strings.TrimSpace(config.ImageID)
	return config, nil
}
