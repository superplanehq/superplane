package ec2

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type DisableImage struct{}

type DisableImageConfiguration struct {
	Region  string `json:"region" mapstructure:"region"`
	ImageID string `json:"imageId" mapstructure:"imageId"`
}

func (c *DisableImage) Name() string {
	return "aws.ec2.disableImage"
}

func (c *DisableImage) Label() string {
	return "EC2 • Disable Image"
}

func (c *DisableImage) Description() string {
	return "Disable an EC2 AMI image"
}

func (c *DisableImage) Documentation() string {
	return `The Disable Image component disables an AMI so it cannot be launched.

## Use Cases

- **Risk containment**: Prevent new launches from vulnerable images
- **Release control**: Temporarily block image usage during maintenance
- **Lifecycle governance**: Enforce policies before image retirement

## Configuration

- **Region**: AWS region where the AMI exists
- **Image ID**: AMI ID to disable`
}

func (c *DisableImage) Icon() string {
	return "aws"
}

func (c *DisableImage) Color() string {
	return "gray"
}

func (c *DisableImage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DisableImage) Configuration() []configuration.Field {
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
			Description: "AMI ID to disable",
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
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
		},
	}
}

func (c *DisableImage) Setup(ctx core.SetupContext) error {
	config := DisableImageConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return err
	}
	if _, err := requireImageID(config.ImageID); err != nil {
		return err
	}

	return nil
}

func (c *DisableImage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DisableImage) Execute(ctx core.ExecutionContext) error {
	config := DisableImageConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}
	imageID, err := requireImageID(config.ImageID)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	requestID, err := client.DisableImage(imageID)
	if err != nil {
		return fmt.Errorf("failed to disable image: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.ec2.image.disabled", []any{map[string]any{
		"requestId": requestID,
		"image": map[string]any{
			"imageId": imageID,
		},
	}})
}

func (c *DisableImage) Actions() []core.Action {
	return []core.Action{}
}

func (c *DisableImage) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DisableImage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DisableImage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DisableImage) Cleanup(ctx core.SetupContext) error {
	return nil
}
