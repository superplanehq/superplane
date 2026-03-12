package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const dropletPollInterval = 10 * time.Second

var validHostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.\-]*$`)

type CreateDroplet struct{}

type CreateDropletSpec struct {
	Name     string   `json:"name"`
	Region   string   `json:"region"`
	Size     string   `json:"size"`
	Image    string   `json:"image"`
	SSHKeys  []string `json:"sshKeys"`
	Tags     []string `json:"tags"`
	UserData string   `json:"userData"`
}

func (c *CreateDroplet) Name() string {
	return "digitalocean.createDroplet"
}

func (c *CreateDroplet) Label() string {
	return "Create Droplet"
}

func (c *CreateDroplet) Description() string {
	return "Create a new DigitalOcean Droplet"
}

func (c *CreateDroplet) Documentation() string {
	return `The Create Droplet component creates a new droplet in DigitalOcean.

## Use Cases

- **Infrastructure provisioning**: Automatically provision droplets from workflow events
- **Scaling**: Create new instances in response to load or alerts
- **Environment setup**: Spin up droplets for testing or staging environments

## Configuration

- **Name**: The hostname for the droplet (required, supports expressions)
- **Region**: Region slug where the droplet will be created (required)
- **Size**: Size slug for the droplet (required)
- **Image**: Image slug or ID for the droplet OS (required)
- **SSH Keys**: SSH key fingerprints or IDs to add to the droplet (optional)
- **Tags**: Tags to apply to the droplet (optional)
- **User Data**: Cloud-init user data script (optional)

## Output

Returns the created droplet object including:
- **id**: Droplet ID
- **name**: Droplet hostname
- **status**: Current droplet status
- **region**: Region information
- **networks**: Network information including IP addresses`
}

func (c *CreateDroplet) Icon() string {
	return "server"
}

func (c *CreateDroplet) Color() string {
	return "gray"
}

func (c *CreateDroplet) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateDroplet) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Droplet Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The hostname for the droplet",
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The region where the droplet will be created",
			Placeholder: "Select a region",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "region",
				},
			},
		},
		{
			Name:        "size",
			Label:       "Size",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The size (CPU/RAM) for the droplet",
			Placeholder: "Select a size",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "size",
				},
			},
		},
		{
			Name:        "image",
			Label:       "Image",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The OS image for the droplet",
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
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "SSH key fingerprints or IDs to add to the droplet",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "SSH Key",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Tags to apply to the droplet",
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
	}
}

func (c *CreateDroplet) Setup(ctx core.SetupContext) error {
	spec := CreateDropletSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}

	if spec.Region == "" {
		return errors.New("region is required")
	}

	if spec.Size == "" {
		return errors.New("size is required")
	}

	if spec.Image == "" {
		return errors.New("image is required")
	}

	return nil
}

func validateHostname(name string) error {
	if !validHostnameRegex.MatchString(name) {
		return fmt.Errorf("invalid droplet name %q: only letters (a-z, A-Z), numbers (0-9), hyphens (-) and dots (.) are allowed", name)
	}
	return nil
}

func (c *CreateDroplet) Execute(ctx core.ExecutionContext) error {
	spec := CreateDropletSpec{}
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
		Name:     spec.Name,
		Region:   spec.Region,
		Size:     spec.Size,
		Image:    spec.Image,
		SSHKeys:  spec.SSHKeys,
		Tags:     spec.Tags,
		UserData: spec.UserData,
	})
	if err != nil {
		return fmt.Errorf("failed to create droplet: %v", err)
	}

	err = ctx.Metadata.Set(map[string]any{"dropletID": droplet.ID})
	if err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, dropletPollInterval)
}

func (c *CreateDroplet) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateDroplet) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateDroplet) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (c *CreateDroplet) HandleAction(ctx core.ActionContext) error {
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
		return fmt.Errorf("failed to get droplet: %v", err)
	}

	switch droplet.Status {
	case "active":
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"digitalocean.droplet.created",
			[]any{droplet},
		)
	case "new":
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, dropletPollInterval)
	default:
		return fmt.Errorf("droplet reached unexpected status %q", droplet.Status)
	}
}

func (c *CreateDroplet) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateDroplet) Cleanup(ctx core.SetupContext) error {
	return nil
}
