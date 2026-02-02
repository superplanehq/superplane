package ecr

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/pkg/integrations/aws/eventbridge"
)

type OnImagePush struct{}

type OnImagePushConfiguration struct {
	Region     string `json:"region" mapstructure:"region"`
	Repository string `json:"repository" mapstructure:"repository"`
}

type OnImagePushMetadata struct {
	Region         string                    `json:"region" mapstructure:"region"`
	SubscriptionID string                    `json:"subscriptionId" mapstructure:"subscriptionId"`
	Repository     *Repository               `json:"repository" mapstructure:"repository"`
	Rule           *eventbridge.RuleMetadata `json:"rule" mapstructure:"rule"`
}

func (p *OnImagePush) Name() string {
	return "aws.ecr.onImagePush"
}

func (p *OnImagePush) Label() string {
	return "ECR - On Image Push"
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
			Type:     configuration.FieldTypeString,
			Required: true,
			Default:  "us-east-1",
		},
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Filter by ECR repository name",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ecr.repository",
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
	if metadata.Repository != nil && metadata.Rule != nil && repositoryMatchesRef(metadata.Repository, config.Repository) {
		return nil
	}

	//
	// Create EventBridge rule and target
	//
	integrationMetadata := common.IntegrationMetadata{}
	err = mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	//
	// If an API destination does not yet exist in the region,
	// we ask the integration to provision it for us.
	//
	apiDestination, ok := integrationMetadata.EventBridge.APIDestinations[config.Region]
	if !ok {
		err = ctx.Metadata.Set(OnImagePushMetadata{
			Region:     config.Region,
			Repository: repository,
		})

		if err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}

		return p.provisionDestination(ctx.Integration, ctx.Requests, config.Region)
	}

	//
	// If an API destination exists in the region,
	// we can use it to subscribe to ECR image push events.
	//
	ruleMetadata, err := eventbridge.CreateRule(
		ctx.Integration,
		ctx.HTTP,
		config.Region,
		apiDestination.ApiDestinationArn,
		p.eventPattern(),
		integrationMetadata.Tags,
	)

	if err != nil {
		return fmt.Errorf("failed to create rule and subscribe: %w", err)
	}

	subscriptionID, err := ctx.Integration.Subscribe(p.eventPattern())
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return ctx.Metadata.Set(OnImagePushMetadata{
		Region:         config.Region,
		SubscriptionID: subscriptionID.String(),
		Repository:     repository,
		Rule:           ruleMetadata,
	})
}

// NOTE: we intentionally do not include the repository-name
// in the event pattern to allow the user to change the repository,
// without us having to update the rule.
func (p *OnImagePush) eventPattern() *common.EventBridgeEvent {
	return &common.EventBridgeEvent{
		DetailType: "ECR Image Action",
		Source:     "aws.ecr",
		Detail: map[string]any{
			"action-type": "PUSH",
			"result":      "SUCCESS",
		},
	}
}

func (p *OnImagePush) provisionDestination(integration core.IntegrationContext, requests core.RequestContext, region string) error {
	err := integration.ScheduleActionCall(
		"provisionDestination",
		common.ProvisionDestinationParameters{Region: region},
		time.Second,
	)

	if err != nil {
		return fmt.Errorf("failed to provision API destination: %w", err)
	}

	return requests.ScheduleActionCall(
		"checkDestinationAvailability",
		map[string]any{},
		5*time.Second,
	)
}

func (p *OnImagePush) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "checkDestinationAvailability",
			Description: "Check if an API destination is available",
		},
	}
}

func (p *OnImagePush) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case "checkDestinationAvailability":
		return p.checkDestinationAvailability(ctx)

	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (p *OnImagePush) checkDestinationAvailability(ctx core.TriggerActionContext) (map[string]any, error) {
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

	apiDestination, ok := integrationMetadata.EventBridge.APIDestinations[metadata.Region]
	if !ok {
		return nil, fmt.Errorf("API destination not found for region: %s", metadata.Region)
	}

	ruleMetadata, err := eventbridge.CreateRule(
		ctx.Integration,
		ctx.HTTP,
		metadata.Region,
		apiDestination.ApiDestinationArn,
		p.eventPattern(),
		integrationMetadata.Tags,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create rule: %w", err)
	}

	subscriptionID, err := ctx.Integration.Subscribe(p.eventPattern())
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	metadata.SubscriptionID = subscriptionID.String()
	metadata.Rule = ruleMetadata
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
		return nil
	}

	return ctx.Events.Emit("aws.ecr.image", ctx.Message)
}

func (p *OnImagePush) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// no-op, since events are received through the integration
	// and routed to OnIntegrationMessage()
	return http.StatusOK, nil
}
