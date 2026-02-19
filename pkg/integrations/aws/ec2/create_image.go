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
	ec2ImageExecutionKVImageID             = "aws_ec2_image_id"
	createImageCheckRuleRetryInterval      = 10 * time.Second
	createImageInitialRuleAvailabilityWait = 5 * time.Second
)

type CreateImage struct{}

type CreateImageConfiguration struct {
	Region      string `json:"region" mapstructure:"region"`
	InstanceID  string `json:"instanceId" mapstructure:"instanceId"`
	Name        string `json:"name" mapstructure:"name"`
	Description string `json:"description" mapstructure:"description"`
	Reboot      bool   `json:"reboot" mapstructure:"reboot"`
}

type CreateImageNodeMetadata struct {
	Region         string `json:"region" mapstructure:"region"`
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
}

type CreateImageExecutionMetadata struct {
	ImageID string `json:"imageId" mapstructure:"imageId"`
	State   string `json:"state" mapstructure:"state"`
}

func (c *CreateImage) Name() string {
	return "aws.ec2.createImage"
}

func (c *CreateImage) Label() string {
	return "EC2 • Create Image"
}

func (c *CreateImage) Description() string {
	return "Create a new AMI image from an EC2 instance"
}

func (c *CreateImage) Documentation() string {
	return `The Create Image component creates a new Amazon Machine Image (AMI) from an EC2 instance.

## Use Cases

- **Golden image pipelines**: Build immutable infrastructure images from validated instances
- **Backup workflows**: Snapshot instance state before deployments or migrations
- **Release automation**: Produce versioned AMIs as part of CI/CD

## Configuration

- **Region**: AWS region where the instance runs
- **Instance**: EC2 instance ID to create an image from
- **Image Name**: Name for the AMI
- **Description**: Optional image description
- **No Reboot**: If enabled, create the image without rebooting the instance

## Completion behavior

- The component waits for EventBridge ` + "`EC2 AMI State Change`" + ` events for the created AMI.
- It completes when the AMI state becomes ` + "`available`" + `.
- It fails if the AMI state becomes ` + "`failed`" + `.
`
}

func (c *CreateImage) Icon() string {
	return "aws"
}

func (c *CreateImage) Color() string {
	return "gray"
}

func (c *CreateImage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateImage) Configuration() []configuration.Field {
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
			Name:        "instanceId",
			Label:       "Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "EC2 instance ID",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.instance",
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
		{
			Name:        "name",
			Label:       "Image Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "my-app-2026-02-18",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "Optional image description",
		},
		{
			Name:        "reboot",
			Label:       "Reboot",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Create the AMI after rebooting the instance",
		},
	}
}

func (c *CreateImage) Setup(ctx core.SetupContext) error {
	config := CreateImageConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	nodeMetadata := CreateImageNodeMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &nodeMetadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	region := strings.TrimSpace(config.Region)
	if region == "" {
		return fmt.Errorf("region is required")
	}

	if nodeMetadata.SubscriptionID != "" && nodeMetadata.Region == region {
		return nil
	}

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, Source, region, DetailTypeAMIStateChange)
	if err != nil {
		return fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		if err := ctx.Metadata.Set(CreateImageNodeMetadata{Region: region}); err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}

		return c.provisionRule(ctx.Integration, ctx.Requests, region)
	}

	subscriptionID, err := ctx.Integration.Subscribe(c.subscriptionPattern(region))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return ctx.Metadata.Set(CreateImageNodeMetadata{
		Region:         region,
		SubscriptionID: subscriptionID.String(),
	})
}

func (c *CreateImage) provisionRule(integration core.IntegrationContext, requests core.RequestContext, region string) error {
	err := integration.ScheduleActionCall(
		"provisionRule",
		common.ProvisionRuleParameters{
			Region:     region,
			Source:     Source,
			DetailType: DetailTypeAMIStateChange,
		},
		time.Second,
	)

	if err != nil {
		return fmt.Errorf("failed to schedule rule provisioning for integration: %w", err)
	}

	return requests.ScheduleActionCall(
		"checkRuleAvailability",
		map[string]any{},
		createImageInitialRuleAvailabilityWait,
	)
}

func (c *CreateImage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateImage) Execute(ctx core.ExecutionContext) error {
	config := CreateImageConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	output, err := client.CreateImage(CreateImageInput{
		InstanceID:  config.InstanceID,
		Name:        config.Name,
		Description: config.Description,
		NoReboot:    !config.Reboot,
	})
	if err != nil {
		return fmt.Errorf("failed to create image: %w", err)
	}

	err = ctx.Metadata.Set(CreateImageExecutionMetadata{
		ImageID: output.ImageID,
		State:   output.State,
	})

	if err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	if err := ctx.ExecutionState.SetKV(ec2ImageExecutionKVImageID, output.ImageID); err != nil {
		return fmt.Errorf("failed to set execution kv: %w", err)
	}

	return nil
}

func (c *CreateImage) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
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

	if detail.State != ImageStateAvailable && detail.State != ImageStateFailed {
		ctx.Logger.Infof("Skipping event for state %s", detail.State)
		return nil
	}

	executionCtx, err := ctx.FindExecutionByKV(ec2ImageExecutionKVImageID, detail.ImageID)
	if err != nil {
		return err
	}
	if executionCtx == nil {
		return nil
	}

	executionMetadata := CreateImageExecutionMetadata{}
	if err := mapstructure.Decode(executionCtx.Metadata.Get(), &executionMetadata); err != nil {
		return fmt.Errorf("failed to decode execution metadata: %w", err)
	}

	executionMetadata.ImageID = detail.ImageID
	executionMetadata.State = detail.State

	if detail.State == ImageStateFailed {
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

func (c *CreateImage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateImage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateImage) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "checkRuleAvailability",
			Description:    "Check if the EventBridge rule is available",
			UserAccessible: false,
		},
	}
}

func (c *CreateImage) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "checkRuleAvailability":
		return c.checkRuleAvailability(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *CreateImage) checkRuleAvailability(ctx core.ActionContext) error {
	nodeMetadata := CreateImageNodeMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &nodeMetadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, Source, nodeMetadata.Region, DetailTypeAMIStateChange)
	if err != nil {
		return fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		return ctx.Requests.ScheduleActionCall(ctx.Name, map[string]any{}, 10*time.Second)
	}

	subscriptionID, err := ctx.Integration.Subscribe(c.subscriptionPattern(nodeMetadata.Region))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	nodeMetadata.SubscriptionID = subscriptionID.String()
	return ctx.Metadata.Set(nodeMetadata)
}

func (c *CreateImage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateImage) subscriptionPattern(region string) *common.EventBridgeEvent {
	return &common.EventBridgeEvent{
		Region:     region,
		DetailType: DetailTypeAMIStateChange,
		Source:     Source,
	}
}
