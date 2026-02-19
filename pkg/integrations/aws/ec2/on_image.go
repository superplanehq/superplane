package ec2

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type OnImage struct{}

type OnImageConfiguration struct {
	Region string   `json:"region" mapstructure:"region"`
	States []string `json:"states" mapstructure:"states"`
}

type OnImageMetadata struct {
	Region         string `json:"region" mapstructure:"region"`
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
}

type AMIStateChangeDetail struct {
	ImageID      string `json:"ImageId" mapstructure:"ImageId"`
	State        string `json:"State" mapstructure:"State"`
	ErrorMessage string `json:"ErrorMessage" mapstructure:"ErrorMessage"`
}

func (p *OnImage) Name() string {
	return "aws.ec2.onImage"
}

func (p *OnImage) Label() string {
	return "EC2 • On Image"
}

func (p *OnImage) Description() string {
	return "Listen to AWS EC2 AMI state change events"
}

func (p *OnImage) Documentation() string {
	return `The On Image trigger starts a workflow execution when an EC2 AMI changes state.

## Use Cases

- **Image pipeline orchestration**: Continue workflows when a new AMI becomes available
- **Failure handling**: Alert and remediate when AMI creation fails
- **Compliance workflows**: Run validation and distribution after image creation

## Configuration

- **Region**: AWS region where AMI state changes are monitored
- **Image State**: State to trigger on (pending, available, failed)

## Event Data

Each AMI state event includes:
- **detail.ImageId**: AMI ID (for example: ami-1234567890abcdef0)
- **detail.State**: AMI state
- **detail.ErrorMessage**: Error message for failed states (if available)
`
}

func (p *OnImage) Icon() string {
	return "aws"
}

func (p *OnImage) Color() string {
	return "gray"
}

func (p *OnImage) Configuration() []configuration.Field {
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
			Name:     "states",
			Label:    "Image State",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{ImageStateAvailable},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Available", Value: ImageStateAvailable},
						{Label: "Failed", Value: ImageStateFailed},
						{Label: "Deregistered", Value: ImageStateDeregistered},
						{Label: "Disabled", Value: ImageStateDisabled},
					},
				},
			},
		},
	}
}

func (p *OnImage) Setup(ctx core.TriggerContext) error {
	metadata := OnImageMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	config := OnImageConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region := strings.TrimSpace(config.Region)
	if region == "" {
		return fmt.Errorf("region is required")
	}

	if metadata.SubscriptionID != "" && metadata.Region == region {
		return nil
	}

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, Source, region, DetailTypeAMIStateChange)
	if err != nil {
		return fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		if err := ctx.Metadata.Set(OnImageMetadata{Region: region}); err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}

		return p.provisionRule(ctx.Integration, ctx.Requests, region)
	}

	subscriptionID, err := ctx.Integration.Subscribe(p.subscriptionPattern(region))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return ctx.Metadata.Set(OnImageMetadata{
		Region:         region,
		SubscriptionID: subscriptionID.String(),
	})
}

func (p *OnImage) provisionRule(integration core.IntegrationContext, requests core.RequestContext, region string) error {
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
		5*time.Second,
	)
}

func (p *OnImage) subscriptionPattern(region string) *common.EventBridgeEvent {
	return &common.EventBridgeEvent{
		Region:     region,
		DetailType: DetailTypeAMIStateChange,
		Source:     Source,
	}
}

func (p *OnImage) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "checkRuleAvailability",
			Description: "Check if the EventBridge rule is available",
		},
	}
}

func (p *OnImage) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case "checkRuleAvailability":
		return p.checkRuleAvailability(ctx)
	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (p *OnImage) checkRuleAvailability(ctx core.TriggerActionContext) (map[string]any, error) {
	metadata := OnImageMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, Source, metadata.Region, DetailTypeAMIStateChange)
	if err != nil {
		return nil, fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		return nil, ctx.Requests.ScheduleActionCall("checkRuleAvailability", map[string]any{}, 10*time.Second)
	}

	subscriptionID, err := ctx.Integration.Subscribe(p.subscriptionPattern(metadata.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	metadata.SubscriptionID = subscriptionID.String()
	return nil, ctx.Metadata.Set(metadata)
}

func (p *OnImage) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	metadata := OnImageMetadata{}
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	config := OnImageConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	event := common.EventBridgeEvent{}
	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	if metadata.Region != "" && event.Region != metadata.Region {
		ctx.Logger.Infof("Skipping event for region %s, expected %s", event.Region, metadata.Region)
		return nil
	}

	detail := AMIStateChangeDetail{}
	if err := mapstructure.Decode(event.Detail, &detail); err != nil {
		return fmt.Errorf("failed to decode event detail: %w", err)
	}

	state := strings.ToLower(strings.TrimSpace(detail.State))
	if state == "" {
		return fmt.Errorf("missing image state in event")
	}

	if len(config.States) > 0 {
		if !slices.Contains(config.States, state) {
			ctx.Logger.Infof("Skipping event for state %s, expected %s", state, config.States)
			return nil
		}
	}

	return ctx.Events.Emit("aws.ec2.image", ctx.Message)
}

func (p *OnImage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (p *OnImage) Cleanup(ctx core.TriggerContext) error {
	return nil
}
