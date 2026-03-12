package ec2

import (
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type DeregisterImage struct{}

type DeregisterImageConfiguration struct {
	Region          string `json:"region" mapstructure:"region"`
	ImageID         string `json:"imageId" mapstructure:"imageId"`
	DeleteSnapshots bool   `json:"deleteSnapshots" mapstructure:"deleteSnapshots"`
}

func (c *DeregisterImage) Name() string {
	return "aws.ec2.deregisterImage"
}

func (c *DeregisterImage) Label() string {
	return "EC2 • Deregister Image"
}

func (c *DeregisterImage) Description() string {
	return "Deregister an EC2 AMI image"
}

func (c *DeregisterImage) Documentation() string {
	return `The Deregister Image component removes an AMI from your account in a region.

## Use Cases

- **Image lifecycle cleanup**: Remove unused AMIs after promotion
- **Compliance operations**: Retire images that should no longer be launched
- **Automation rollback**: Clean up AMIs created by failed workflows

## Configuration

- **Region**: AWS region where the AMI exists
- **Image ID**: AMI ID to deregister
- **Delete Snapshots**: If enabled, delete the snapshots associated with the AMI`
}

func (c *DeregisterImage) Icon() string {
	return "aws"
}

func (c *DeregisterImage) Color() string {
	return "gray"
}

func (c *DeregisterImage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeregisterImage) Configuration() []configuration.Field {
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
			Description: "AMI ID to deregister",
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
						{
							Name:  "includeDisabled",
							Value: aws.String("true"),
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
			Name:        "deleteSnapshots",
			Label:       "Delete Snapshots",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Delete the snapshots associated with the AMI",
		},
	}
}

func (c *DeregisterImage) Setup(ctx core.SetupContext) error {
	config := DeregisterImageConfiguration{}
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

func (c *DeregisterImage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeregisterImage) Execute(ctx core.ExecutionContext) error {
	config := DeregisterImageConfiguration{}
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
	snapshotIDs, err := c.getSnapshotIDs(config, client)
	if err != nil {
		return fmt.Errorf("failed to get snapshot IDs: %w", err)
	}

	requestID, err := client.DeregisterImage(imageID)
	if err != nil {
		return fmt.Errorf("failed to deregister image: %w", err)
	}

	for _, snapshotID := range snapshotIDs {
		if _, err := client.DeleteSnapshot(snapshotID); err != nil {
			return fmt.Errorf("failed to delete snapshot: %w", err)
		}
	}

	output := map[string]any{
		"requestId": requestID,
		"image": map[string]any{
			"imageId": imageID,
		},
	}

	if len(snapshotIDs) > 0 {
		output["deletedSnapshots"] = snapshotIDs
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.ec2.image.deregistered",
		[]any{output},
	)
}

func (c *DeregisterImage) getSnapshotIDs(config DeregisterImageConfiguration, client *Client) ([]string, error) {
	//
	// If deleting the snapshot is not enabled,
	// nothing to do here.
	//
	if !config.DeleteSnapshots {
		return nil, nil
	}

	//
	// Otherwise, we need to describe the image to get the snapshot IDs.
	//
	image, err := client.DescribeImage(config.ImageID)
	if err != nil {
		return nil, fmt.Errorf("failed to describe image: %w", err)
	}

	snapshotIDs := make([]string, 0)
	for _, mapping := range image.BlockDeviceMappings {
		if mapping.Ebs.SnapshotID != "" {
			snapshotIDs = append(snapshotIDs, mapping.Ebs.SnapshotID)
		}
	}

	return snapshotIDs, nil
}

func (c *DeregisterImage) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeregisterImage) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeregisterImage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DeregisterImage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeregisterImage) Cleanup(ctx core.SetupContext) error {
	return nil
}
