package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteDroplet struct{}

type DeleteDropletSpec struct {
	Droplet string `json:"droplet"`
}

func (d *DeleteDroplet) Name() string {
	return "digitalocean.deleteDroplet"
}

func (d *DeleteDroplet) Label() string {
	return "Delete Droplet"
}

func (d *DeleteDroplet) Description() string {
	return "Delete a DigitalOcean Droplet by ID"
}

func (d *DeleteDroplet) Documentation() string {
	return `The Delete Droplet component permanently deletes a droplet from your DigitalOcean account.

## Use Cases

- **Cleanup**: Remove temporary or test droplets after use
- **Cost optimization**: Automatically tear down unused infrastructure
- **Automated workflows**: Delete droplets as part of deployment rollback or cleanup processes
- **Environment management**: Remove ephemeral environments after testing

## Configuration

- **Droplet**: The droplet to delete (required, supports expressions)

## Output

Returns information about the deleted droplet:
- **dropletId**: The ID of the droplet that was deleted

## Important Notes

- This operation is **permanent** and cannot be undone
- All data on the droplet will be lost
- The droplet will be shut down if it's running before deletion
- Any snapshots of the droplet will remain in your account`
}

func (d *DeleteDroplet) Icon() string {
	return "trash-2"
}

func (d *DeleteDroplet) Color() string {
	return "red"
}

func (d *DeleteDroplet) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteDroplet) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "droplet",
			Label:       "Droplet",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The droplet ID to delete",
			Placeholder: "Select droplet",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "droplet",
					UseNameAsValue: false,
				},
			},
		},
	}
}

func (d *DeleteDroplet) Setup(ctx core.SetupContext) error {
	spec := DeleteDropletSpec{}
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

func (d *DeleteDroplet) Execute(ctx core.ExecutionContext) error {
	spec := DeleteDropletSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	dropletID, err := strconv.Atoi(spec.Droplet)
	if err != nil {
		return fmt.Errorf("invalid droplet ID must be a number")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.DeleteDroplet(dropletID)
	if err != nil {
		if doErr, ok := err.(*DOAPIError); ok && doErr.StatusCode == http.StatusNotFound {
			// Droplet already deleted, emit success
			return ctx.ExecutionState.Emit(
				core.DefaultOutputChannel.Name,
				"digitalocean.droplet.deleted",
				[]any{map[string]any{"dropletId": dropletID}},
			)
		}
		return fmt.Errorf("failed to delete droplet: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.droplet.deleted",
		[]any{map[string]any{"dropletId": dropletID}},
	)
}

func (d *DeleteDroplet) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteDroplet) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteDroplet) Actions() []core.Action {
	return []core.Action{}
}

func (d *DeleteDroplet) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (d *DeleteDroplet) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteDroplet) Cleanup(ctx core.SetupContext) error {
	return nil
}
