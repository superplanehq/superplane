package ecr

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type OnImagePush struct{}

type OnImagePushConfiguration struct {
	Region     string `json:"region" mapstructure:"region"`
	Repository string `json:"repository" mapstructure:"repository"`
}

type OnImagePushMetadata struct {
	Region         string      `json:"region" mapstructure:"region"`
	SubscriptionID string      `json:"subscriptionId" mapstructure:"subscriptionId"`
	Repository     *Repository `json:"repository" mapstructure:"repository"`
}

func (p *OnImagePush) Name() string {
	return "aws.ecr.onImagePush"
}

func (p *OnImagePush) Label() string {
	return "ECR • On Image Push"
}

func (p *OnImagePush) Description() string {
	return "Listen to AWS ECR image push events"
}

func (p *OnImagePush) Documentation() string {
	return `The On Image Push trigger starts a workflow execution when an image is pushed to an ECR repository.

## Use Cases

- **Build pipelines**: Trigger builds and deployments on container pushes
- **Security automation**: Kick off scans or alerts for newly pushed images
- **Release workflows**: Promote artifacts when a tag is published

## Configuration

- **Repositories**: Optional filters for ECR repository names
- **Image Tags**: Optional filters for image tags (for example: ` + "`latest`" + ` or ` + "`^v[0-9]+`" + `)

## Event Data

Each image push event includes:
- **detail.repository-name**: ECR repository name
- **detail.image-tag**: Tag that was pushed
- **detail.image-digest**: Digest of the image
`
}

func (p *OnImagePush) Icon() string {
	return "aws"
}

func (p *OnImagePush) Color() string {
	return "gray"
}

func (p *OnImagePush) Configuration() []configuration.Field {
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

func (p *OnImagePush) Setup(ctx core.TriggerContext) error {
	metadata := OnImagePushMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	config := OnImagePushConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	repository, err := validateRepository(ctx, config.Region, config.Repository, metadata.Repository)
	if err != nil {
		return fmt.Errorf("failed to validate repository: %w", err)
	}

	//
	// EventBridge rule and target have been setup already.
	//
	if metadata.Repository != nil && repositoryMatchesRef(metadata.Repository, config.Repository) {
		return nil
	}

	region := strings.TrimSpace(config.Region)
	if region == "" {
		return fmt.Errorf("region is required")
	}

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, Source, region, DetailTypeECRImageAction)
	if err != nil {
		return fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		err = ctx.Metadata.Set(OnImagePushMetadata{
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

	return ctx.Metadata.Set(OnImagePushMetadata{
		Region:         region,
		SubscriptionID: subscriptionID.String(),
		Repository:     repository,
	})
}

func (p *OnImagePush) subscriptionPattern(region string) *common.EventBridgeEvent {
	return &common.EventBridgeEvent{
		Region:     region,
		DetailType: DetailTypeECRImageAction,
		Source:     Source,
		Detail: map[string]any{
			"action-type": "PUSH",
			"result":      "SUCCESS",
		},
	}
}

func (p *OnImagePush) provisionRule(integration core.IntegrationContext, requests core.RequestContext, region string) error {
	err := integration.ScheduleActionCall(
		"provisionRule",
		common.ProvisionRuleParameters{
			Region:     region,
			Source:     Source,
			DetailType: DetailTypeECRImageAction,
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

func (p *OnImagePush) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "checkRuleAvailability",
			Description: "Check if the EventBridge rule is available",
		},
	}
}

func (p *OnImagePush) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case "checkRuleAvailability":
		return p.checkRuleAvailability(ctx)

	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (p *OnImagePush) checkRuleAvailability(ctx core.TriggerActionContext) (map[string]any, error) {
	metadata := OnImagePushMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, Source, metadata.Region, DetailTypeECRImageAction)
	if err != nil {
		return nil, fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		return nil, ctx.Requests.ScheduleActionCall(ctx.Name, map[string]any{}, 10*time.Second)
	}

	//
	// Rule is available, subscribe to the integration with the proper pattern.
	//
	subscriptionID, err := ctx.Integration.Subscribe(p.subscriptionPattern(metadata.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	metadata.SubscriptionID = subscriptionID.String()
	return nil, ctx.Metadata.Set(metadata)
}

func (p *OnImagePush) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	metadata := OnImagePushMetadata{}
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

	return ctx.Events.Emit("aws.ecr.image.push", ctx.Message)
}

func (p *OnImagePush) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// no-op, since events are received through the integration
	// and routed to OnIntegrationMessage()
	return http.StatusOK, nil
}

func (p *OnImagePush) Cleanup(ctx core.TriggerContext) error {
	return nil
}
