package ec2

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	ec2CopyImageExecutionKVImageID          = "aws_ec2_copy_image_id"
	copyImageCheckRuleRetryInterval         = 10 * time.Second
	copyImageInitialRuleAvailabilityTimeout = 5 * time.Second
)

type CopyImage struct{}

type CopyImageConfiguration struct {
	Region        string `json:"region" mapstructure:"region"`
	SourceRegion  string `json:"sourceRegion" mapstructure:"sourceRegion"`
	SourceImageID string `json:"sourceImageId" mapstructure:"sourceImageId"`
	Name          string `json:"name" mapstructure:"name"`
	Description   string `json:"description" mapstructure:"description"`
}

type CopyImageNodeMetadata struct {
	Region         string `json:"region" mapstructure:"region"`
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
}

type CopyImageExecutionMetadata struct {
	ImageID       string `json:"imageId" mapstructure:"imageId"`
	SourceImageID string `json:"sourceImageId" mapstructure:"sourceImageId"`
	SourceRegion  string `json:"sourceRegion" mapstructure:"sourceRegion"`
	State         string `json:"state" mapstructure:"state"`
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

## Completion behavior

- The component waits for EventBridge ` + "`EC2 AMI State Change`" + ` events for the copied AMI.
- It completes when the AMI state becomes ` + "`available`" + `.
- It fails if the AMI state becomes ` + "`failed`" + `.
`
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
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Destination Region",
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
			Name:     "sourceRegion",
			Label:    "Source Region",
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
			Name:        "sourceImageId",
			Label:       "Source Image ID",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "AMI ID in the source region",
			Placeholder: "ami-1234567890abcdef0",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.image",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "sourceRegion",
							},
						},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "sourceRegion",
					Values: []string{"*"},
				},
			},
		},
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

	nodeMetadata := CopyImageNodeMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &nodeMetadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
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

	if nodeMetadata.SubscriptionID != "" && nodeMetadata.Region == region {
		return nil
	}

	integrationMetadata := common.IntegrationMetadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	if integrationMetadata.EventBridge == nil {
		return fmt.Errorf("event bridge metadata is not configured")
	}

	if !hasAMIStateChangeRule(integrationMetadata) {
		if err := ctx.Metadata.Set(CopyImageNodeMetadata{Region: region}); err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}

		if err := ctx.Integration.ScheduleActionCall(
			"provisionRule",
			common.ProvisionRuleParameters{
				Region:     region,
				Source:     Source,
				DetailType: DetailTypeAMIStateChange,
			},
			time.Second,
		); err != nil {
			return fmt.Errorf("failed to schedule rule provisioning for integration: %w", err)
		}

		return ctx.Requests.ScheduleActionCall(
			"checkRuleAvailability",
			map[string]any{},
			copyImageInitialRuleAvailabilityTimeout,
		)
	}

	subscriptionID, err := ctx.Integration.Subscribe(c.subscriptionPattern(region))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return ctx.Metadata.Set(CopyImageNodeMetadata{
		Region:         region,
		SubscriptionID: subscriptionID.String(),
	})
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

	err = ctx.Metadata.Set(CopyImageExecutionMetadata{
		ImageID:       output.ImageID,
		SourceImageID: sourceImageID,
		SourceRegion:  sourceRegion,
		State:         output.State,
	})
	if err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	if err := ctx.ExecutionState.SetKV(ec2CopyImageExecutionKVImageID, output.ImageID); err != nil {
		return fmt.Errorf("failed to set execution kv: %w", err)
	}

	return nil
}

func (c *CopyImage) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	event := common.EventBridgeEvent{}
	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	if event.Source != Source || event.DetailType != DetailTypeAMIStateChange {
		ctx.Logger.Infof("Skipping event for source %s or detail type %s", event.Source, event.DetailType)
		return nil
	}

	detail := AMIStateChangeDetail{}
	if err := mapstructure.Decode(event.Detail, &detail); err != nil {
		return fmt.Errorf("failed to decode event detail: %w", err)
	}

	state := strings.TrimSpace(detail.State)
	if state != ImageStateAvailable && state != ImageStateFailed {
		ctx.Logger.Infof("Skipping event for state %s", detail.State)
		return nil
	}

	executionCtx, err := ctx.FindExecutionByKV(ec2CopyImageExecutionKVImageID, detail.ImageID)
	if err != nil {
		return err
	}
	if executionCtx == nil {
		return nil
	}

	executionMetadata := CopyImageExecutionMetadata{}
	if err := mapstructure.Decode(executionCtx.Metadata.Get(), &executionMetadata); err != nil {
		return fmt.Errorf("failed to decode execution metadata: %w", err)
	}

	executionMetadata.ImageID = detail.ImageID
	executionMetadata.State = state
	if err := executionCtx.Metadata.Set(executionMetadata); err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	if state == ImageStateFailed {
		return executionCtx.ExecutionState.Fail(models.CanvasNodeExecutionResultReasonError, detail.ErrorMessage)
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, event.Region)
	image, err := client.DescribeImage(executionMetadata.ImageID)
	if err != nil {
		return fmt.Errorf("failed to describe image: %w", err)
	}

	return executionCtx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.ec2.image",
		[]any{map[string]any{
			"image": image,
		}},
	)
}

func (c *CopyImage) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "checkRuleAvailability",
			Description:    "Check if the EventBridge rule is available",
			UserAccessible: false,
		},
	}
}

func (c *CopyImage) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "checkRuleAvailability":
		return c.checkRuleAvailability(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
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

func (c *CopyImage) checkRuleAvailability(ctx core.ActionContext) error {
	nodeMetadata := CopyImageNodeMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &nodeMetadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	integrationMetadata := common.IntegrationMetadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	if !hasAMIStateChangeRule(integrationMetadata) {
		ctx.Logger.Infof("Rule not available for source %s - checking again in %s", Source, copyImageCheckRuleRetryInterval)
		return ctx.Requests.ScheduleActionCall(
			"checkRuleAvailability",
			map[string]any{},
			copyImageCheckRuleRetryInterval,
		)
	}

	subscriptionID, err := ctx.Integration.Subscribe(c.subscriptionPattern(nodeMetadata.Region))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	nodeMetadata.SubscriptionID = subscriptionID.String()
	return ctx.Metadata.Set(nodeMetadata)
}

func (c *CopyImage) subscriptionPattern(region string) *common.EventBridgeEvent {
	return &common.EventBridgeEvent{
		Region:     region,
		DetailType: DetailTypeAMIStateChange,
		Source:     Source,
	}
}
