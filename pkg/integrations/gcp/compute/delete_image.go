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
)

type DeleteImage struct{}

type DeleteImageSpec struct {
	Image string `mapstructure:"image"`
}

func (d *DeleteImage) Name() string {
	return "gcp.deleteImage"
}

func (d *DeleteImage) Label() string {
	return "Compute • Delete Image"
}

func (d *DeleteImage) Description() string {
	return "Permanently delete a Google Compute Engine custom image"
}

func (d *DeleteImage) Documentation() string {
	return `The Delete Image component permanently deletes a Compute Engine custom image.

## Use Cases

- **Cleanup**: Remove temporary or test images after use
- **Cost optimization**: Delete unused images to reduce storage costs
- **Lifecycle automation**: Remove obsolete images as part of release pipelines

## Configuration

- **Image**: The custom image to delete. Pick from the list of images in your project, or pass an expression chained from an upstream node (e.g. the ` + "`selfLink`" + ` emitted by ` + "`gcp.createImage`" + `).

## Output

Returns the name of the deleted image.

## Important Notes

- This operation is **permanent** and cannot be undone.
- If the image is not found, the action fails so that misconfigured or stale expressions do not silently mask incomplete cleanup.
- Deleting an image does not affect VM instances or disks already created from it.`
}

func (d *DeleteImage) Icon() string {
	return "trash-2"
}

func (d *DeleteImage) Color() string {
	return "red"
}

func (d *DeleteImage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteImage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "image",
			Label:       "Image",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The custom image to delete. Lists every custom image in your project.",
			Placeholder: "Select image",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeCustomImages,
				},
			},
		},
	}
}

func (d *DeleteImage) Setup(ctx core.SetupContext) error {
	spec := DeleteImageSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.Image) == "" {
		return errors.New("image is required")
	}

	return resolveImageNodeMetadata(ctx, spec.Image)
}

func (d *DeleteImage) Execute(ctx core.ExecutionContext) error {
	spec := DeleteImageSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	urlProject, name, err := parseImagePath(spec.Image)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	project := client.ProjectID()
	if urlProject != "" && urlProject != project {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf(
			"image belongs to project %q but this GCP integration is bound to project %q; cross-project deletes are not supported",
			urlProject, project,
		))
	}

	callCtx := context.Background()
	path := fmt.Sprintf("projects/%s/global/images/%s", project, name)
	body, err := client.Delete(callCtx, path)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to delete image: %v", err))
	}

	opName, err := operationNameFromResponse(body, "delete image")
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	if err := WaitForGlobalOperation(callCtx, client, project, opName); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("error waiting for delete image operation: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.compute.image.deleted",
		[]any{map[string]any{"imageName": name}},
	)
}

func (d *DeleteImage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteImage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteImage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteImage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (d *DeleteImage) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeleteImage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
