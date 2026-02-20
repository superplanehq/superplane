package ec2

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type GetImage struct{}

type GetImageConfiguration struct {
	Region  string `json:"region" mapstructure:"region"`
	ImageID string `json:"imageId" mapstructure:"imageId"`
}

func (c *GetImage) Name() string {
	return "aws.ec2.getImage"
}

func (c *GetImage) Label() string {
	return "EC2 • Get Image"
}

func (c *GetImage) Description() string {
	return "Get an EC2 AMI image by ID"
}

func (c *GetImage) Documentation() string {
	return `The Get Image component retrieves metadata for an EC2 AMI.

## Use Cases

- **Release automation**: Validate AMI metadata before deployment
- **Operational checks**: Inspect AMI state and ownership in workflows
- **Traceability**: Resolve AMI details by image ID

## Configuration

- **Region**: AWS region of the AMI
- **Image ID**: AMI ID (for example: ami-1234567890abcdef0)`
}

func (c *GetImage) Icon() string {
	return "aws"
}

func (c *GetImage) Color() string {
	return "gray"
}

func (c *GetImage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetImage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-east-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: common.AllRegions,
				},
			},
		},
		{
			Name:        "imageId",
			Label:       "Image ID",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "ami-1234567890abcdef0",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.image",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
		},
	}
}

func (c *GetImage) Setup(ctx core.SetupContext) error {
	config := GetImageConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.ImageID = strings.TrimSpace(config.ImageID)

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}
	if config.ImageID == "" {
		return fmt.Errorf("image ID is required")
	}

	return nil
}

func (c *GetImage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetImage) Execute(ctx core.ExecutionContext) error {
	config := GetImageConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.ImageID = strings.TrimSpace(config.ImageID)

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	image, err := client.DescribeImage(config.ImageID)
	if err != nil {
		return fmt.Errorf("failed to describe image: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.ec2.image",
		[]any{map[string]any{
			"image": image,
		}},
	)
}

func (c *GetImage) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetImage) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetImage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetImage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetImage) Cleanup(ctx core.SetupContext) error {
	return nil
}
