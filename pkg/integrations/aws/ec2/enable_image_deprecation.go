package ec2

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type EnableImageDeprecation struct{}

type EnableImageDeprecationConfiguration struct {
	Region      string `json:"region" mapstructure:"region"`
	ImageID     string `json:"imageId" mapstructure:"imageId"`
	DeprecateAt string `json:"deprecateAt" mapstructure:"deprecateAt"`
}

func (c *EnableImageDeprecation) Name() string {
	return "aws.ec2.enableImageDeprecation"
}

func (c *EnableImageDeprecation) Label() string {
	return "EC2 • Enable Image Deprecation"
}

func (c *EnableImageDeprecation) Description() string {
	return "Enable deprecation for an EC2 AMI image"
}

func (c *EnableImageDeprecation) Documentation() string {
	return `The Enable Image Deprecation component sets a deprecation time for an AMI.

## Use Cases

- **Release lifecycle**: Schedule AMI retirement dates
- **Compliance enforcement**: Ensure images expire on policy deadlines
- **Operational hygiene**: Phase out outdated images in a controlled window

## Configuration

- **Region**: AWS region where the AMI exists
- **Image ID**: AMI ID to deprecate
- **Deprecate At**: RFC3339 timestamp when deprecation takes effect`
}

func (c *EnableImageDeprecation) Icon() string {
	return "aws"
}

func (c *EnableImageDeprecation) Color() string {
	return "gray"
}

func (c *EnableImageDeprecation) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *EnableImageDeprecation) Configuration() []configuration.Field {
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
			Description: "AMI ID to deprecate",
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
		{
			Name:        "deprecateAt",
			Label:       "Deprecate At",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "2026-03-01T00:00:00Z",
			Description: "RFC3339 timestamp for AMI deprecation",
		},
	}
}

func (c *EnableImageDeprecation) Setup(ctx core.SetupContext) error {
	config := EnableImageDeprecationConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return err
	}

	if _, err := requireImageID(config.ImageID); err != nil {
		return err
	}

	if config.DeprecateAt == "" {
		return fmt.Errorf("deprecateAt is required")
	}

	return nil
}

func (c *EnableImageDeprecation) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *EnableImageDeprecation) Execute(ctx core.ExecutionContext) error {
	config := EnableImageDeprecationConfiguration{}
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

	deprecateAt, err := time.Parse(time.RFC3339, config.DeprecateAt)
	if err != nil {
		return fmt.Errorf("deprecateAt must be a valid RFC3339 timestamp: %w", err)
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	output, err := client.EnableImageDeprecation(imageID, deprecateAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to enable image deprecation: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.ec2.image.deprecationEnabled", []any{
		map[string]any{
			"requestId":   output.RequestID,
			"deprecateAt": output.DeprecateAt,
			"image": map[string]any{
				"imageId": output.ImageID,
			},
		},
	})
}

func (c *EnableImageDeprecation) Actions() []core.Action {
	return []core.Action{}
}

func (c *EnableImageDeprecation) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *EnableImageDeprecation) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *EnableImageDeprecation) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *EnableImageDeprecation) Cleanup(ctx core.SetupContext) error {
	return nil
}
