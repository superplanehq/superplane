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

type OnImageScan struct{}

type OnImageScanConfiguration struct {
	Region     string `json:"region" mapstructure:"region"`
	Repository string `json:"repository" mapstructure:"repository"`
}

type OnImageScanMetadata struct {
	Region         string                    `json:"region" mapstructure:"region"`
	SubscriptionID string                    `json:"subscriptionId" mapstructure:"subscriptionId"`
	Repository     *Repository               `json:"repository" mapstructure:"repository"`
	Rule           *eventbridge.RuleMetadata `json:"rule" mapstructure:"rule"`
}

func (p *OnImageScan) Name() string {
	return "aws.ecr.onImageScan"
}

func (p *OnImageScan) Label() string {
	return "ECR - On Image Scan"
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
		err = ctx.Integration.ScheduleActionCall(
			"provisionDestination",
			common.ProvisionDestinationParameters{Region: config.Region},
			time.Second,
		)

		if err != nil {
			return fmt.Errorf("failed to provision API destination: %w", err)
		}

		return ctx.Requests.ScheduleActionCall(
			"checkDestinationAvailability",
			map[string]any{},
			5*time.Second,
		)
	}

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

	return ctx.Metadata.Set(OnImageScanMetadata{
		Region:         config.Region,
		SubscriptionID: subscriptionID.String(),
		Repository:     repository,
		Rule:           ruleMetadata,
	})
}

func (p *OnImageScan) eventPattern() *common.EventBridgeEvent {
	return &common.EventBridgeEvent{
		DetailType: "ECR Image Scan",
		Source:     "aws.ecr",
		Detail: map[string]any{
			"scan-status": "COMPLETE",
		},
	}
}

func (p *OnImageScan) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "checkDestinationAvailability",
			Description: "Check if an API destination is available",
		},
	}
}

func (p *OnImageScan) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case "checkDestinationAvailability":
		return p.checkDestinationAvailability(ctx)

	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (p *OnImageScan) checkDestinationAvailability(ctx core.TriggerActionContext) (map[string]any, error) {
	metadata := OnImageScanMetadata{}
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

	metadata.Rule = ruleMetadata
	metadata.SubscriptionID = subscriptionID.String()
	return nil, ctx.Metadata.Set(metadata)
}

func (p *OnImageScan) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	return ctx.Events.Emit("aws.ecr.image.scan", ctx.Message)
}

func (p *OnImageScan) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// no-op, since events are received through the integration
	// and routed to OnIntegrationMessage()
	return http.StatusOK, nil
}
