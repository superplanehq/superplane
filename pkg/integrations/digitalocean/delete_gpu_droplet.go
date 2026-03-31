package digitalocean

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteGPUDroplet struct{}

type DeleteGPUDropletSpec struct {
	Droplet string `json:"droplet"`
}

func (d *DeleteGPUDroplet) Name() string {
	return "digitalocean.deleteGPUDroplet"
}

func (d *DeleteGPUDroplet) Label() string {
	return "Delete GPU Droplet"
}

func (d *DeleteGPUDroplet) Description() string {
	return "Delete a DigitalOcean GPU Droplet by ID"
}

func (d *DeleteGPUDroplet) Documentation() string {
	return `The Delete GPU Droplet component permanently deletes a GPU droplet from your DigitalOcean account.

## Use Cases

- **Cleanup**: Remove temporary GPU droplets after training or inference tasks
- **Cost optimization**: Automatically tear down expensive GPU infrastructure when not in use
- **Automated workflows**: Delete GPU droplets as part of ML pipeline cleanup
- **Environment management**: Remove ephemeral GPU environments after testing

## Configuration

- **Droplet**: The GPU droplet to delete (required, supports expressions)

## Output

Returns information about the deleted GPU droplet:
- **dropletId**: The ID of the GPU droplet that was deleted

## Important Notes

- This operation is **permanent** and cannot be undone
- All data on the GPU droplet will be lost
- The droplet will be shut down if it's running before deletion
- Any snapshots of the GPU droplet will remain in your account`
}

func (d *DeleteGPUDroplet) Icon() string {
	return "trash-2"
}

func (d *DeleteGPUDroplet) Color() string {
	return "red"
}

func (d *DeleteGPUDroplet) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteGPUDroplet) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "droplet",
			Label:       "GPU Droplet",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The GPU droplet ID to delete",
			Placeholder: "Select GPU droplet",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "gpu_droplet",
					UseNameAsValue: false,
				},
			},
		},
	}
}

func (d *DeleteGPUDroplet) Setup(ctx core.SetupContext) error {
	spec := DeleteGPUDropletSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Droplet == "" {
		return errors.New("droplet is required")
	}

	err = resolveDropletMetadata(ctx, spec.Droplet)
	if err != nil {
		return fmt.Errorf("error resolving droplet metadata: %v", err)
	}

	return nil
}

func (d *DeleteGPUDroplet) Execute(ctx core.ExecutionContext) error {
	spec := DeleteGPUDropletSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	dropletID, err := parseDropletID(spec.Droplet)
	if err != nil {
		return fmt.Errorf("invalid droplet ID %q: %w", spec.Droplet, err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.DeleteDroplet(dropletID)
	if err != nil {
		if doErr, ok := err.(*DOAPIError); ok && doErr.StatusCode == http.StatusNotFound {
			// GPU droplet already deleted, emit success
			return ctx.ExecutionState.Emit(
				core.DefaultOutputChannel.Name,
				"digitalocean.gpuDroplet.deleted",
				[]any{map[string]any{"dropletId": dropletID}},
			)
		}
		return fmt.Errorf("failed to delete GPU droplet: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.gpuDroplet.deleted",
		[]any{map[string]any{"dropletId": dropletID}},
	)
}

func (d *DeleteGPUDroplet) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteGPUDroplet) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteGPUDroplet) Actions() []core.Action {
	return []core.Action{}
}

func (d *DeleteGPUDroplet) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (d *DeleteGPUDroplet) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteGPUDroplet) Cleanup(ctx core.SetupContext) error {
	return nil
}
