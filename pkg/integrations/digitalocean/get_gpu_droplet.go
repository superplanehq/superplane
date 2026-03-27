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

type GetGPUDroplet struct{}

type GetGPUDropletSpec struct {
	Droplet string `json:"droplet"`
}

func (g *GetGPUDroplet) Name() string {
	return "digitalocean.getGPUDroplet"
}

func (g *GetGPUDroplet) Label() string {
	return "Get GPU Droplet"
}

func (g *GetGPUDroplet) Description() string {
	return "Fetch details of a DigitalOcean GPU Droplet by ID"
}

func (g *GetGPUDroplet) Documentation() string {
	return `The Get GPU Droplet component retrieves detailed information about a specific GPU droplet.

## Use Cases

- **Status checks**: Verify GPU droplet state before performing operations
- **Information retrieval**: Get current IP addresses, GPU configuration, and status
- **Pre-flight validation**: Check GPU droplet exists before operations like snapshot or power management
- **Monitoring**: Track GPU droplet configuration and network details

## Configuration

- **Droplet**: The GPU droplet to retrieve (required, supports expressions)

## Output

Returns the GPU droplet object including:
- **id**: Droplet ID
- **name**: Droplet hostname
- **status**: Current droplet status (new, active, off, archive)
- **memory**: RAM in MB
- **vcpus**: Number of virtual CPUs
- **disk**: Disk size in GB
- **region**: Region information
- **image**: Image information
- **size_slug**: GPU size identifier
- **networks**: Network information including IP addresses
- **tags**: Applied tags`
}

func (g *GetGPUDroplet) Icon() string {
	return "gpu"
}

func (g *GetGPUDroplet) Color() string {
	return "purple"
}

func (g *GetGPUDroplet) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetGPUDroplet) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "droplet",
			Label:       "GPU Droplet",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The GPU droplet to retrieve",
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

func (g *GetGPUDroplet) Setup(ctx core.SetupContext) error {
	spec := GetGPUDropletSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Droplet == "" {
		return errors.New("droplet is required")
	}

	err = resolveDropletMetadata(ctx, spec.Droplet)
	if err != nil {
		return fmt.Errorf("error resolving GPU droplet metadata: %v", err)
	}

	return nil
}

func (g *GetGPUDroplet) Execute(ctx core.ExecutionContext) error {
	spec := GetGPUDropletSpec{}
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
		return fmt.Errorf("invalid GPU droplet ID %q: %w", spec.Droplet, err)
	}

	droplet, err := client.GetDroplet(dropletID)
	if err != nil {
		return fmt.Errorf("failed to get GPU droplet: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.gpuDroplet.fetched",
		[]any{droplet},
	)
}

func (g *GetGPUDroplet) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetGPUDroplet) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetGPUDroplet) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetGPUDroplet) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (g *GetGPUDroplet) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetGPUDroplet) Cleanup(ctx core.SetupContext) error {
	return nil
}
