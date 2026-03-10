package digitalocean

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetDroplet struct{}

type GetDropletSpec struct {
	DropletID string `json:"dropletId" mapstructure:"dropletId"`
}

func (c *GetDroplet) Name() string {
	return "digitalocean.getDroplet"
}

func (c *GetDroplet) Label() string {
	return "Get Droplet"
}

func (c *GetDroplet) Description() string {
	return "Fetch details of a DigitalOcean Droplet by ID"
}

func (c *GetDroplet) Documentation() string {
	return `The Get Droplet component retrieves details of an existing DigitalOcean Droplet.

## Use Cases

- **Monitoring**: Check the current status of a droplet
- **Chaining**: Fetch droplet details before performing other operations
- **Inventory**: Retrieve droplet configuration for auditing

## Configuration

- **Droplet ID**: The ID of the droplet to fetch (required, supports expressions)

## Output

Returns the droplet object including:
- **id**: Droplet ID
- **name**: Droplet hostname
- **status**: Current droplet status
- **region**: Region information
- **networks**: Network information including IP addresses`
}

func (c *GetDroplet) Icon() string {
	return "server"
}

func (c *GetDroplet) Color() string {
	return "gray"
}

func (c *GetDroplet) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetDroplet) ExampleOutput() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"id":        98765432,
			"name":      "my-droplet",
			"memory":    1024,
			"vcpus":     1,
			"disk":      25,
			"status":    "active",
			"region":    map[string]any{"name": "New York 3", "slug": "nyc3"},
			"image":     map[string]any{"id": 12345, "name": "Ubuntu 24.04 (LTS) x64", "slug": "ubuntu-24-04-x64"},
			"size_slug": "s-1vcpu-1gb",
			"networks":  map[string]any{"v4": []any{map[string]any{"ip_address": "104.131.186.241", "type": "public"}}},
			"tags":      []any{"web"},
		},
	}
}

func (c *GetDroplet) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "dropletId",
			Label:       "Droplet ID",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The ID of the droplet to fetch",
		},
	}
}

func (c *GetDroplet) Setup(ctx core.SetupContext) error {
	spec := GetDropletSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.DropletID == "" {
		return fmt.Errorf("dropletId is required")
	}

	return nil
}

func (c *GetDroplet) Execute(ctx core.ExecutionContext) error {
	spec := GetDropletSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	dropletID, err := resolveIntID(ctx.Configuration, "dropletId")
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	droplet, err := client.GetDroplet(dropletID)
	if err != nil {
		return fmt.Errorf("failed to get droplet: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.droplet.fetched",
		[]any{droplet},
	)
}

func (c *GetDroplet) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetDroplet) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetDroplet) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetDroplet) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetDroplet) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *GetDroplet) Cleanup(ctx core.SetupContext) error {
	return nil
}
