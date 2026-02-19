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

type CopyImage struct{}

type CopyImageConfiguration struct {
	Region        string `json:"region" mapstructure:"region"`
	SourceRegion  string `json:"sourceRegion" mapstructure:"sourceRegion"`
	SourceImageID string `json:"sourceImageId" mapstructure:"sourceImageId"`
	Name          string `json:"name" mapstructure:"name"`
	Description   string `json:"description" mapstructure:"description"`
}

func (c *CopyImage) Name() string {
	return "aws.ec2.copyImage"
}

func (c *CopyImage) Label() string {
	return "EC2 • Copy Image"
}

func (c *CopyImage) Description() string {
	return "Copy an EC2 AMI image to another region"
}

func (c *CopyImage) Documentation() string {
	return `The Copy Image component copies an AMI to another AWS region.

## Use Cases

- **Multi-region rollouts**: Replicate golden images to deployment regions
- **Disaster recovery**: Keep AMI backups in secondary regions
- **Promotion workflows**: Copy validated images across environments

## Configuration

- **Destination Region**: AWS region where the copied AMI is created
- **Source Region**: AWS region where the source AMI exists
- **Source Image ID**: AMI ID to copy
- **Image Name**: Name for the copied AMI
- **Description**: Optional AMI description

## Output behavior

- Returns the new AMI ID as soon as the copy request is accepted.
- The initial state is ` + "`pending`" + `.`
}

func (c *CopyImage) Icon() string {
	return "aws"
}

func (c *CopyImage) Color() string {
	return "gray"
}

func (c *CopyImage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CopyImage) Configuration() []configuration.Field {
	sourceRegionField := regionSelectField()
	sourceRegionField.Name = "sourceRegion"
	sourceRegionField.Label = "Source Region"

	destinationRegionField := regionSelectField()
	destinationRegionField.Label = "Destination Region"

	return []configuration.Field{
		destinationRegionField,
		sourceRegionField,
		imageIDField("sourceImageId", "Source Image ID", "AMI ID in the source region", "sourceRegion"),
		{
			Name:        "name",
			Label:       "Image Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "my-app-2026-02-19",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "Optional image description",
		},
	}
}

func (c *CopyImage) Setup(ctx core.SetupContext) error {
	config := CopyImageConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return err
	}
	if _, err := requireSourceRegion(config.SourceRegion); err != nil {
		return err
	}
	if _, err := requireSourceImageID(config.SourceImageID); err != nil {
		return err
	}
	if _, err := requireImageName(config.Name); err != nil {
		return err
	}

	return nil
}

func (c *CopyImage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CopyImage) Execute(ctx core.ExecutionContext) error {
	config := CopyImageConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}
	sourceRegion, err := requireSourceRegion(config.SourceRegion)
	if err != nil {
		return err
	}
	sourceImageID, err := requireSourceImageID(config.SourceImageID)
	if err != nil {
		return err
	}
	name, err := requireImageName(config.Name)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	output, err := client.CopyImage(CopyImageInput{
		SourceImageID: sourceImageID,
		SourceRegion:  sourceRegion,
		Name:          name,
		Description:   normalizeOptionalString(config.Description),
	})
	if err != nil {
		return fmt.Errorf("failed to copy image: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.ec2.image.copied", []any{output})
}

func (c *CopyImage) Actions() []core.Action {
	return []core.Action{}
}

func (c *CopyImage) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CopyImage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CopyImage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CopyImage) Cleanup(ctx core.SetupContext) error {
	return nil
}
