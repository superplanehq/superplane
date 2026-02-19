package ecr

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

type OnImageScan struct{}

type OnImageScanConfiguration struct {
	Region     string `json:"region" mapstructure:"region"`
	Repository string `json:"repository" mapstructure:"repository"`
}

type OnImageScanMetadata struct {
	Region         string      `json:"region" mapstructure:"region"`
	SubscriptionID string      `json:"subscriptionId" mapstructure:"subscriptionId"`
	Repository     *Repository `json:"repository" mapstructure:"repository"`
}

func (p *OnImageScan) Name() string {
	return "aws.ecr.onImageScan"
}

func (p *OnImageScan) Label() string {
	return "ECR • On Image Scan"
}

func (p *OnImageScan) Description() string {
	return "Listen to AWS ECR image scan events"
}

func (p *OnImageScan) Documentation() string {
	return `The On Image Scan trigger starts a workflow execution when an ECR image scan completes.

## Use Cases

- **Security automation**: Notify teams or open issues on new findings
- **Compliance checks**: Gate promotions based on severity thresholds
- **Reporting**: Aggregate scan findings across repositories

## Configuration

- **Repositories**: Optional filters for ECR repository names

## Notes

- **Enhanced scanning**: Enhanced scanning events are sent by Amazon Inspector (aws.inspector2)

## Event Data

Each image scan event includes:
- **detail.scan-status**: Scan status (for example: COMPLETE)
- **detail.repository-name**: ECR repository name
- **detail.image-digest**: Digest of the image
- **detail.image-tags**: Tags associated with the image
- **detail.finding-severity-counts**: Counts per severity level (if any)
`
}

func (p *OnImageScan) Icon() string {
	return "aws"
}

func (p *OnImageScan) Color() string {
	return "gray"
}

func (p *OnImageScan) Configuration() []configuration.Field {
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
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Filter by ECR repository name",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ecr.repository",
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

func (p *OnImageScan) Setup(ctx core.TriggerContext) error {
	metadata := OnImageScanMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	config := OnImageScanConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	repository, err := validateRepository(ctx, config.Region, config.Repository, metadata.Repository)
	if err != nil {
		return fmt.Errorf("failed to validate repository: %w", err)
	}

	region := strings.TrimSpace(config.Region)
	if region == "" {
		return fmt.Errorf("region is required")
	}

	//
	// EventBridge rule and target have been setup already.
	//
	if metadata.Repository != nil && repositoryMatchesRef(metadata.Repository, config.Repository) {
		return nil
	}

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, Source, region, DetailTypeECRImageScan)
	if err != nil {
		return fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		err = ctx.Metadata.Set(OnImageScanMetadata{
			Region:     region,
			Repository: repository,
		})

		if err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}

		return p.provisionRule(ctx.Integration, ctx.Requests, region)
	}

	//
	// If the rule exists, subscribe to the integration with the proper pattern.
	//
	subscriptionID, err := ctx.Integration.Subscribe(p.subscriptionPattern(region))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return ctx.Metadata.Set(OnImageScanMetadata{
		Region:         region,
		SubscriptionID: subscriptionID.String(),
		Repository:     repository,
	})
}

func (p *OnImageScan) provisionRule(integration core.IntegrationContext, requests core.RequestContext, region string) error {
	err := integration.ScheduleActionCall(
		"provisionRule",
		common.ProvisionRuleParameters{
			Region:     region,
			Source:     Source,
			DetailType: DetailTypeECRImageScan,
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

func (p *OnImageScan) subscriptionPattern(region string) *common.EventBridgeEvent {
	return &common.EventBridgeEvent{
		Region:     region,
		DetailType: DetailTypeECRImageScan,
		Source:     Source,
		Detail: map[string]any{
			"scan-status": "COMPLETE",
		},
	}
}

func (p *OnImageScan) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "checkRuleAvailability",
			Description: "Check if an EventBridge rule is available",
		},
	}
}

func (p *OnImageScan) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case "checkRuleAvailability":
		return p.checkRuleAvailability(ctx)

	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (p *OnImageScan) checkRuleAvailability(ctx core.TriggerActionContext) (map[string]any, error) {
	metadata := OnImagePushMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	integrationMetadata := common.IntegrationMetadata{}
	err = mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	//
	// If the rule was not provisioned yet, check again in 10 seconds.
	//
	rule, ok := integrationMetadata.EventBridge.Rules[Source]
	if !ok {
		ctx.Logger.Infof("Rule not found for source %s - checking again in 10 seconds", Source)
		return nil, ctx.Requests.ScheduleActionCall(
			"checkRuleAvailability",
			map[string]any{},
			10*time.Second,
		)
	}

	//
	// If the rule does not have the detail type we are interested in, check again in 10 seconds.
	//
	if !slices.Contains(rule.DetailTypes, DetailTypeECRImageScan) {
		ctx.Logger.Infof("Rule does not have detail type '%s' - checking again in 10 seconds", DetailTypeECRImageScan)
		return nil, ctx.Requests.ScheduleActionCall(
			"checkRuleAvailability",
			map[string]any{},
			10*time.Second,
		)
	}

	subscriptionID, err := ctx.Integration.Subscribe(p.subscriptionPattern(metadata.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	metadata.SubscriptionID = subscriptionID.String()
	return nil, ctx.Metadata.Set(metadata)
}

func (p *OnImageScan) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	metadata := OnImageScanMetadata{}
	err := mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	event := common.EventBridgeEvent{}
	err = mapstructure.Decode(ctx.Message, &event)
	if err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	repositoryName, ok := event.Detail["repository-name"]
	if !ok {
		return fmt.Errorf("missing repository-name in event")
	}

	r, ok := repositoryName.(string)
	if !ok {
		return fmt.Errorf("invalid repository-name in event")
	}

	if r != metadata.Repository.RepositoryName {
		ctx.Logger.Infof("Skipping event for repository %s, expected %s", r, metadata.Repository.RepositoryName)
		return nil
	}

	return ctx.Events.Emit("aws.ecr.image.scan", ctx.Message)
}

func (p *OnImageScan) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// no-op, since events are received through the integration
	// and routed to OnIntegrationMessage()
	return http.StatusOK, nil
}

func (p *OnImageScan) Cleanup(ctx core.TriggerContext) error {
	return nil
}
