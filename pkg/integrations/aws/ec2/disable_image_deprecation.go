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

type DisableImageDeprecation struct{}

type DisableImageDeprecationConfiguration struct {
	Region  string `json:"region" mapstructure:"region"`
	ImageID string `json:"imageId" mapstructure:"imageId"`
}

func (c *DisableImageDeprecation) Name() string {
	return "aws.ec2.disableImageDeprecation"
}

func (c *DisableImageDeprecation) Label() string {
	return "EC2 • Disable Image Deprecation"
}

func (c *DisableImageDeprecation) Description() string {
	return "Disable deprecation for an EC2 AMI image"
}

func (c *DisableImageDeprecation) Documentation() string {
	return `The Disable Image Deprecation component removes the deprecation schedule from an AMI.

## Use Cases

- **Release extension**: Keep an image available longer than planned
- **Rollback support**: Reopen older images for temporary use
- **Policy exceptions**: Remove deprecation when operational needs change

## Configuration

- **Region**: AWS region where the AMI exists
- **Image ID**: AMI ID to remove deprecation from`
}

func (c *DisableImageDeprecation) Icon() string {
	return "aws"
}

func (c *DisableImageDeprecation) Color() string {
	return "gray"
}

func (c *DisableImageDeprecation) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DisableImageDeprecation) Configuration() []configuration.Field {
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
			Description: "AMI ID to remove deprecation from",
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

func (c *DisableImageDeprecation) Setup(ctx core.SetupContext) error {
	config := DisableImageDeprecationConfiguration{}
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

func (c *DisableImageDeprecation) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DisableImageDeprecation) Execute(ctx core.ExecutionContext) error {
	config := DisableImageDeprecationConfiguration{}
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
	requestID, err := client.DisableImageDeprecation(imageID)
	if err != nil {
		return fmt.Errorf("failed to disable image deprecation: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.ec2.image.deprecationDisabled", []any{
		map[string]any{
			"requestId": requestID,
			"image": map[string]any{
				"imageId": imageID,
			},
		},
	})
}

func (c *DisableImageDeprecation) Actions() []core.Action {
	return []core.Action{}
}

func (c *DisableImageDeprecation) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DisableImageDeprecation) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DisableImageDeprecation) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DisableImageDeprecation) Cleanup(ctx core.SetupContext) error {
	return nil
}
