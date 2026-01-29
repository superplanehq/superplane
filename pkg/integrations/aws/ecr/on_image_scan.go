package ecr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/pkg/integrations/aws/eventbridge"
)

type OnImageScan struct{}

type OnImageScanConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
}

type OnImageScanMetadata struct {
	Repository     *Repository `json:"repository" mapstructure:"repository"`
	SubscriptionID string      `json:"subscriptionId" mapstructure:"subscriptionId"`
	RuleArn        string      `json:"ruleArn" mapstructure:"ruleArn"`
	TargetID       string      `json:"targetId" mapstructure:"targetId"`
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

	repository, err := validateRepository(ctx, config.Repository, metadata.Repository)
	if err != nil {
		return fmt.Errorf("failed to validate repository: %w", err)
	}

	//
	// EventBridge rule and target have been setup already.
	//
	if metadata.Repository != nil && metadata.RuleArn != "" && repositoryMatchesRef(metadata.Repository, config.Repository) {
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

	region := strings.TrimSpace(common.RegionFromInstallation(ctx.Integration))
	if region == "" {
		return fmt.Errorf("region is required")
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := eventbridge.NewClient(ctx.HTTP, creds, region)

	//
	// NOTE: we intentionally do not include the repository-name
	// in the event pattern to allow the user to change the repository,
	// without us having to update the rule.
	//
	eventPattern, err := json.Marshal(map[string]any{
		"source":      []string{"aws.ecr"},
		"detail-type": []string{"ECR Image Scan"},
		"detail": map[string]any{
			"scan-status": []string{"COMPLETE"},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to marshal event pattern: %w", err)
	}

	// TODO: use a more meaningful names and IDs
	targetID := uuid.NewString()
	ruleName := fmt.Sprintf("superplane-%s", uuid.NewString())
	ruleArn, err := client.PutRule(ruleName, string(eventPattern), "SuperPlane ECR webhook rule")
	if err != nil {
		return fmt.Errorf("error creating EventBridge rule: %v", err)
	}

	err = client.PutTargets(ruleName, []eventbridge.Target{
		{
			ID:  targetID,
			Arn: integrationMetadata.APIDestination.ApiDestinationArn,
		},
	})

	if err != nil {
		return fmt.Errorf("error creating EventBridge target: %v", err)
	}

	subscriptionID, err := ctx.Integration.Subscribe(
		common.EventBridgeEvent{
			DetailType: "ECR Image Scan",
			Source:     "aws.ecr",
			Detail: map[string]any{
				"scan-status":     "COMPLETE",
				"repository-name": repository.RepositoryName,
			},
		},
	)

	if err != nil {
		return fmt.Errorf("failed to subscribe to ECR image scan events: %w", err)
	}

	return ctx.Metadata.Set(OnImageScanMetadata{
		SubscriptionID: subscriptionID.String(),
		Repository:     repository,
		RuleArn:        ruleArn,
		TargetID:       targetID,
	})
}

func (p *OnImageScan) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnImageScan) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnImageScan) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	return ctx.Events.Emit("aws.ecr.image.scan", ctx.Message)
}

func (p *OnImageScan) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// no-op, since events are received through the integration
	// and routed to OnIntegrationMessage()
	return http.StatusOK, nil
}
