package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const gpuDropletPollInterval = 10 * time.Second

type CreateGPUDroplet struct{}

type CreateGPUDropletSpec struct {
	Name       string   `json:"name"`
	Region     string   `json:"region"`
	Size       string   `json:"size"`
	Image      string   `json:"image"`
	SSHKeys    []string `json:"sshKeys"`
	Tags       []string `json:"tags"`
	UserData   string   `json:"userData"`
	Backups    bool     `json:"backups"`
	IPv6       bool     `json:"ipv6"`
	Monitoring bool     `json:"monitoring"`
	VpcUUID    string   `json:"vpcUuid"`
}

func (c *CreateGPUDroplet) Name() string {
	return "digitalocean.createGPUDroplet"
}

func (c *CreateGPUDroplet) Label() string {
	return "Create GPU Droplet"
}

func (c *CreateGPUDroplet) Description() string {
	return "Create a new DigitalOcean GPU Droplet"
}

func (c *CreateGPUDroplet) Documentation() string {
	return `The Create GPU Droplet component creates a new GPU-powered droplet in DigitalOcean.

## Use Cases

- **AI/ML workloads**: Provision GPU-powered instances for training and inference
- **High-performance computing**: Spin up GPU instances for compute-intensive tasks
- **Rendering workloads**: Create GPU droplets for 3D rendering pipelines

## Configuration

- **Name**: The hostname for the GPU droplet (required, supports expressions)
- **Region**: Region slug where the GPU droplet will be created (required, only shows GPU-capable regions)
- **Size**: GPU size slug for the droplet (required, only shows GPU sizes)
- **Image**: Image slug or ID for the droplet OS (required, shows GPU-compatible images)
- **SSH Keys**: SSH keys to add to the droplet. Must have been added to the DigitalOcean team. (optional)
- **Tags**: Tags to apply to the droplet (optional)
- **User Data**: Cloud-init user data script (optional)
- **Backups**: Enable automated backups for the droplet (optional)
- **IPv6**: Enable IPv6 networking on the droplet (optional)
- **Monitoring**: Enable DigitalOcean monitoring agent on the droplet (optional)
- **VPC UUID**: UUID of the VPC to create the droplet in (optional)

## Output

Returns the created GPU droplet object including:
- **id**: Droplet ID
- **name**: Droplet hostname
- **status**: Current droplet status
- **region**: Region information
- **size_slug**: GPU size identifier
- **networks**: Network information including IP addresses`
}

func (c *CreateGPUDroplet) Icon() string {
	return "gpu"
}

func (c *CreateGPUDroplet) Color() string {
	return "purple"
}

func (c *CreateGPUDroplet) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateGPUDroplet) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Droplet Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The hostname for the GPU droplet",
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The region where the GPU droplet will be created",
			Placeholder: "Select a region",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "gpu_region",
				},
			},
		},
		{
			Name:        "size",
			Label:       "GPU Size",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The GPU size for the droplet",
			Placeholder: "Select a GPU size",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "gpu_size",
				},
			},
		},
		{
			Name:        "image",
			Label:       "Image",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The OS image for the GPU droplet",
			Placeholder: "Select an image",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "image",
				},
			},
		},
		{
			Name:        "sshKeys",
			Label:       "SSH Keys",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "SSH keys to add to the droplet",
			Placeholder: "Select SSH keys",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "ssh_key",
					Multi: true,
				},
			},
		},
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Tags to apply to the GPU droplet",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "userData",
			Label:       "User Data",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Description: "Cloud-init user data script",
		},
		{
			Name:        "backups",
			Label:       "Enable Backups",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Enable automated backups for the droplet",
		},
		{
			Name:        "ipv6",
			Label:       "Enable IPv6",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Enable IPv6 networking on the droplet",
		},
		{
			Name:        "monitoring",
			Label:       "Enable Monitoring",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Enable DigitalOcean monitoring agent on the droplet",
		},
		{
			Name:        "vpcUuid",
			Label:       "VPC",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "VPC to create the GPU droplet in",
			Placeholder: "Select a VPC",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "vpc",
				},
			},
		},
	}
}

func (c *CreateGPUDroplet) Setup(ctx core.SetupContext) error {
	spec := CreateGPUDropletSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}

	if spec.Region == "" {
		// Check if there are any GPU regions available
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return fmt.Errorf("region is required (unable to check available GPU regions: %v)", err)
		}

		gpuRegions, err := client.ListGPURegions()
		if err != nil {
			return fmt.Errorf("region is required (unable to check available GPU regions: %v)", err)
		}

		if len(gpuRegions) == 0 {
			return errors.New("no GPU-capable regions are currently available in your DigitalOcean account")
		}

		return errors.New("region is required")
	}

	if spec.Size == "" {
		// Check if there are any GPU sizes available
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return fmt.Errorf("GPU size is required (unable to check available GPU sizes: %v)", err)
		}

		gpuSizes, err := client.ListGPUSizes()
		if err != nil {
			return fmt.Errorf("GPU size is required (unable to check available GPU sizes: %v)", err)
		}

		if len(gpuSizes) == 0 {
			return errors.New("no GPU sizes are currently available in your DigitalOcean account")
		}

		return errors.New("GPU size is required")
	}

	if spec.Image == "" {
		return errors.New("image is required")
	}

	return nil
}

func (c *CreateGPUDroplet) Execute(ctx core.ExecutionContext) error {
	spec := CreateGPUDropletSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if err := validateHostname(spec.Name); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	droplet, err := client.CreateDroplet(CreateDropletRequest{
		Name:       spec.Name,
		Region:     spec.Region,
		Size:       spec.Size,
		Image:      spec.Image,
		SSHKeys:    spec.SSHKeys,
		Tags:       spec.Tags,
		UserData:   spec.UserData,
		Backups:    spec.Backups,
		IPv6:       spec.IPv6,
		Monitoring: spec.Monitoring,
		VpcUUID:    spec.VpcUUID,
	})
	if err != nil {
		return fmt.Errorf("failed to create GPU droplet: %v", err)
	}

	err = ctx.Metadata.Set(map[string]any{"dropletID": droplet.ID})
	if err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, gpuDropletPollInterval)
}

func (c *CreateGPUDroplet) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateGPUDroplet) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateGPUDroplet) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (c *CreateGPUDroplet) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata struct {
		DropletID int `mapstructure:"dropletID"`
	}

	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	droplet, err := client.GetDroplet(metadata.DropletID)
	if err != nil {
		return fmt.Errorf("failed to get GPU droplet: %v", err)
	}

	switch droplet.Status {
	case "active":
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"digitalocean.gpuDroplet.created",
			[]any{droplet},
		)
	case "new":
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, gpuDropletPollInterval)
	default:
		return fmt.Errorf("GPU droplet reached unexpected status %q", droplet.Status)
	}
}

func (c *CreateGPUDroplet) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateGPUDroplet) Cleanup(ctx core.SetupContext) error {
	return nil
}
