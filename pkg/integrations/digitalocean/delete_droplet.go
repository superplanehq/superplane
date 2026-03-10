package digitalocean

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteDroplet struct{}

type DeleteDropletSpec struct {
	DropletID string `json:"dropletId" mapstructure:"dropletId"`
}

func (c *DeleteDroplet) Name() string {
	return "digitalocean.deleteDroplet"
}

func (c *DeleteDroplet) Label() string {
	return "Delete Droplet"
}

func (c *DeleteDroplet) Description() string {
	return "Delete a DigitalOcean Droplet"
}

func (c *DeleteDroplet) Documentation() string {
	return `The Delete Droplet component deletes an existing DigitalOcean Droplet.

## How It Works

1. Deletes the specified droplet via the DigitalOcean API
2. Emits on the default output when the droplet is deleted. If deletion fails, the execution errors.

## Configuration

- **Droplet ID**: The ID of the droplet to delete (required, supports expressions)

## Output

Returns confirmation of the deleted droplet:
- **dropletId**: The ID of the deleted droplet`
}

func (c *DeleteDroplet) Icon() string {
	return "server"
}

func (c *DeleteDroplet) Color() string {
	return "gray"
}

func (c *DeleteDroplet) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteDroplet) ExampleOutput() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"dropletId": 98765432,
		},
	}
}

func (c *DeleteDroplet) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "dropletId",
			Label:       "Droplet ID",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The ID of the droplet to delete",
		},
	}
}

func (c *DeleteDroplet) Setup(ctx core.SetupContext) error {
	spec := DeleteDropletSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.DropletID == "" {
		return fmt.Errorf("dropletId is required")
	}

	return nil
}

func (c *DeleteDroplet) Execute(ctx core.ExecutionContext) error {
	dropletID, err := resolveIntID(ctx.Configuration, "dropletId")
	if err != nil {
		return err
	}

	if err := ctx.Metadata.Set(map[string]any{"dropletId": dropletID}); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.DeleteDroplet(dropletID); err != nil {
		return fmt.Errorf("failed to delete droplet: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.droplet.deleted",
		[]any{map[string]any{"dropletId": dropletID}},
	)
}

func (c *DeleteDroplet) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteDroplet) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteDroplet) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteDroplet) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteDroplet) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *DeleteDroplet) Cleanup(ctx core.SetupContext) error {
	return nil
}
