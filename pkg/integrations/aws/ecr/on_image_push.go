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

type OnImagePush struct{}

type OnImagePushConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
}

type OnImagePushMetadata struct {
	Repository     *Repository `json:"repository" mapstructure:"repository"`
	SubscriptionID string      `json:"subscriptionId" mapstructure:"subscriptionId"`
	RuleArn        string      `json:"ruleArn" mapstructure:"ruleArn"`
	TargetID       string      `json:"targetId" mapstructure:"targetId"`
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
		"detail-type": []string{"ECR Image Action"},
		"detail": map[string]any{
			"action-type": []string{"PUSH"},
			"result":      []string{"SUCCESS"},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to marshal event pattern: %w", err)
	}

	// TODO: use a more meaningful names and IDs
	targetID := uuid.NewString()
	ruleName := fmt.Sprintf("superplane-%s", uuid.NewString())
	ruleArn, err := client.PutRule(ruleName, string(eventPattern), "SuperPlane ECR webhook rule", integrationMetadata.Tags)
	if err != nil {
		return fmt.Errorf("error creating EventBridge rule: %v", err)
	}

	if err := client.TagResource(ruleArn, integrationMetadata.Tags); err != nil {
		return fmt.Errorf("error tagging EventBridge rule: %v", err)
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
			DetailType: "ECR Image Action",
			Source:     "aws.ecr",
			Detail: map[string]any{
				"action-type":     "PUSH",
				"result":          "SUCCESS",
				"repository-name": repository.RepositoryName,
			},
		},
	)

	if err != nil {
		return fmt.Errorf("failed to subscribe to ECR image push events: %w", err)
	}

	return ctx.Metadata.Set(OnImagePushMetadata{
		SubscriptionID: subscriptionID.String(),
		Repository:     repository,
		RuleArn:        ruleArn,
		TargetID:       targetID,
	})
}

func (p *OnImagePush) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnImagePush) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnImagePush) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	return ctx.Events.Emit("aws.ecr.image.push", ctx.Message)
}

func (p *OnImagePush) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// no-op, since events are received through the integration
	// and routed to OnIntegrationMessage()
	return http.StatusOK, nil
}
