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

type EnableImage struct{}

type EnableImageConfiguration struct {
	Region  string `json:"region" mapstructure:"region"`
	ImageID string `json:"imageId" mapstructure:"imageId"`
}

type EnableImageOutput struct {
	RequestID string `json:"requestId" mapstructure:"requestId"`
	ImageID   string `json:"imageId" mapstructure:"imageId"`
	Region    string `json:"region" mapstructure:"region"`
	Enabled   bool   `json:"enabled" mapstructure:"enabled"`
}

func (c *EnableImage) Name() string {
	return "aws.ec2.enableImage"
}

func (c *EnableImage) Label() string {
	return "EC2 • Enable Image"
}

func (c *EnableImage) Description() string {
	return "Enable an EC2 AMI image"
}

func (c *EnableImage) Documentation() string {
	return `The Enable Image component enables a previously disabled AMI.

## Use Cases

- **Release promotion**: Re-enable AMIs after staged validation
- **Operational recovery**: Restore image availability after temporary restrictions
- **Lifecycle workflows**: Toggle image launchability based on policy checks

## Configuration

- **Region**: AWS region where the AMI exists
- **Image ID**: AMI ID to enable`
}

func (c *EnableImage) Icon() string {
	return "aws"
}

func (c *EnableImage) Color() string {
	return "gray"
}

func (c *EnableImage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *EnableImage) Configuration() []configuration.Field {
	return []configuration.Field{
		regionSelectField(),
		imageIDField("imageId", "Image ID", "AMI ID to enable", "region"),
	}
}

func (c *EnableImage) Setup(ctx core.SetupContext) error {
	config := EnableImageConfiguration{}
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

func (c *EnableImage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *EnableImage) Execute(ctx core.ExecutionContext) error {
	config := EnableImageConfiguration{}
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
	requestID, err := client.EnableImage(imageID)
	if err != nil {
		return fmt.Errorf("failed to enable image: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.ec2.image.enabled", []any{EnableImageOutput{
		RequestID: requestID,
		ImageID:   imageID,
		Region:    region,
		Enabled:   true,
	}})
}

func (c *EnableImage) Actions() []core.Action {
	return []core.Action{}
}

func (c *EnableImage) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *EnableImage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *EnableImage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *EnableImage) Cleanup(ctx core.SetupContext) error {
	return nil
}
