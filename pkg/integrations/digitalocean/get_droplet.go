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

type GetDroplet struct{}

type GetDropletSpec struct {
	Droplet string `json:"droplet"`
}

func (g *GetDroplet) Name() string {
	return "digitalocean.getDroplet"
}

func (g *GetDroplet) Label() string {
	return "Get Droplet"
}

func (g *GetDroplet) Description() string {
	return "Fetch details of a DigitalOcean Droplet by ID"
}

func (g *GetDroplet) Documentation() string {
	return `The Get Droplet component retrieves detailed information about a specific droplet.

## Use Cases

- **Status checks**: Verify droplet state before performing operations
- **Information retrieval**: Get current IP addresses, configuration, and status
- **Pre-flight validation**: Check droplet exists before operations like snapshot or power management
- **Monitoring**: Track droplet configuration and network details

## Configuration

- **Droplet**: The droplet to retrieve (required, supports expressions)

## Output

Returns the droplet object including:
- **id**: Droplet ID
- **name**: Droplet hostname
- **status**: Current droplet status (new, active, off, archive)
- **memory**: RAM in MB
- **vcpus**: Number of virtual CPUs
- **disk**: Disk size in GB
- **region**: Region information
- **image**: Image information
- **size_slug**: Size identifier
- **networks**: Network information including IP addresses
- **tags**: Applied tags`
}

func (g *GetDroplet) Icon() string {
	return "info"
}

func (g *GetDroplet) Color() string {
	return "gray"
}

func (g *GetDroplet) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetDroplet) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "droplet",
			Label:       "Droplet",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The droplet to retrieve",
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

func (g *GetDroplet) Setup(ctx core.SetupContext) error {
	spec := GetDropletSpec{}
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

func (g *GetDroplet) Execute(ctx core.ExecutionContext) error {
	spec := GetDropletSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	dropletID, err := parseDropletID(spec.Droplet)
	if err != nil {
		return fmt.Errorf("invalid droplet ID %q: %w", spec.Droplet, err)
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

func (g *GetDroplet) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetDroplet) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetDroplet) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetDroplet) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (g *GetDroplet) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetDroplet) Cleanup(ctx core.SetupContext) error {
	return nil
}
