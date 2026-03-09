package codebuild

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

const (
	Source                     = "aws.codebuild"
	DetailTypeBuildStateChange = "CodeBuild Build State Change"
)

var AllBuildStates = []configuration.FieldOption{
	{Label: "IN_PROGRESS", Value: "IN_PROGRESS"},
	{Label: "SUCCEEDED", Value: "SUCCEEDED"},
	{Label: "FAILED", Value: "FAILED"},
	{Label: "STOPPED", Value: "STOPPED"},
	{Label: "TIMED_OUT", Value: "TIMED_OUT"},
	{Label: "FAULT", Value: "FAULT"},
}

type OnBuild struct{}

type OnBuildConfiguration struct {
	Region   string                    `json:"region" mapstructure:"region"`
	Projects []configuration.Predicate `json:"projects" mapstructure:"projects"`
	States   []string                  `json:"states" mapstructure:"states"`
}

type OnBuildMetadata struct {
	Region         string `json:"region" mapstructure:"region"`
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
}

func (t *OnBuild) Name() string {
	return "aws.codebuild.onBuild"
}

func (t *OnBuild) Label() string {
	return "CodeBuild • On Build Completed"
}

func (t *OnBuild) Description() string {
	return "Listen to AWS CodeBuild build state change events"
}

func (t *OnBuild) Documentation() string {
	return `The On Build Completed trigger starts a workflow execution when AWS CodeBuild emits a build state change event.

## Use Cases

- **Build visibility**: Start workflows whenever build state changes
- **Notification**: Notify teams when a build fails or times out
- **Workflow orchestration**: Trigger follow-up automations on specific build states

## Configuration

- **Region**: AWS region where build events are observed
- **Projects**: Optional project name filters (supports equals, not-equals, and regex matches)
- **Build States**: Optional list of states to match (for example SUCCEEDED, FAILED, STOPPED)

## Event Data

Each event includes:
- **detail.project-name**: CodeBuild project name
- **detail.build-id**: Build ID
- **detail.build-status**: Build status
`
}

func (t *OnBuild) Icon() string {
	return "aws"
}

func (t *OnBuild) Color() string {
	return "gray"
}

func (t *OnBuild) Configuration() []configuration.Field {
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
			Name:     "projects",
			Label:    "Projects",
			Type:     configuration.FieldTypeAnyPredicateList,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
		{
			Name:     "states",
			Label:    "Build States",
			Type:     configuration.FieldTypeMultiSelect,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: AllBuildStates,
				},
			},
		},
	}
}

func (t *OnBuild) Setup(ctx core.TriggerContext) error {
	metadata := OnBuildMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	config := OnBuildConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region := strings.TrimSpace(config.Region)
	if region == "" {
		return fmt.Errorf("region is required")
	}

	if metadata.SubscriptionID != "" {
		return nil
	}

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, Source, region, DetailTypeBuildStateChange)
	if err != nil {
		return fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		if err := ctx.Metadata.Set(OnBuildMetadata{Region: region}); err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}

		return t.provisionRule(ctx.Integration, ctx.Requests, region)
	}

	subscriptionID, err := ctx.Integration.Subscribe(t.subscriptionPattern(region))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return ctx.Metadata.Set(OnBuildMetadata{
		Region:         region,
		SubscriptionID: subscriptionID.String(),
	})
}

func (t *OnBuild) provisionRule(integration core.IntegrationContext, requests core.RequestContext, region string) error {
	err := integration.ScheduleActionCall(
		"provisionRule",
		common.ProvisionRuleParameters{
			Region:     region,
			Source:     Source,
			DetailType: DetailTypeBuildStateChange,
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

func (t *OnBuild) subscriptionPattern(region string) *common.EventBridgeEvent {
	return &common.EventBridgeEvent{
		Region:     region,
		DetailType: DetailTypeBuildStateChange,
		Source:     Source,
	}
}

func (t *OnBuild) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "checkRuleAvailability",
			Description: "Check if the EventBridge rule is available",
		},
	}
}

func (t *OnBuild) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case "checkRuleAvailability":
		return t.checkRuleAvailability(ctx)
	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (t *OnBuild) checkRuleAvailability(ctx core.TriggerActionContext) (map[string]any, error) {
	metadata := OnBuildMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, Source, metadata.Region, DetailTypeBuildStateChange)
	if err != nil {
		return nil, fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		return nil, ctx.Requests.ScheduleActionCall(ctx.Name, map[string]any{}, 10*time.Second)
	}

	subscriptionID, err := ctx.Integration.Subscribe(t.subscriptionPattern(metadata.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	metadata.SubscriptionID = subscriptionID.String()
	return nil, ctx.Metadata.Set(metadata)
}

func (t *OnBuild) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	metadata := OnBuildMetadata{}
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	config := OnBuildConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	event := common.EventBridgeEvent{}
	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	region := metadata.Region

	if region != "" && event.Region != region {
		ctx.Logger.Infof("Skipping event for region %s, expected %s", event.Region, region)
		return nil
	}

	projectName, ok := event.Detail["project-name"].(string)
	if !ok || projectName == "" {
		return fmt.Errorf("missing project name in event")
	}

	status, ok := event.Detail["build-status"].(string)
	if !ok || status == "" {
		return fmt.Errorf("missing build status in event")
	}
	status = strings.ToUpper(strings.TrimSpace(status))

	if len(config.Projects) > 0 {
		if !configuration.MatchesAnyPredicate(config.Projects, projectName) {
			ctx.Logger.Infof("Skipping event for project %s, does not match any predicate: %v", projectName, config.Projects)
			return nil
		}
	}

	if len(config.States) > 0 {
		normalized := make([]string, 0, len(config.States))
		for _, s := range config.States {
			normalized = append(normalized, strings.ToUpper(strings.TrimSpace(s)))
		}
		if !slices.Contains(normalized, status) {
			ctx.Logger.Infof("Skipping event for project %s with status %s", projectName, status)
			return nil
		}
	}

	return ctx.Events.Emit("aws.codebuild.build", ctx.Message)
}

func (t *OnBuild) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (t *OnBuild) Cleanup(ctx core.TriggerContext) error {
	return nil
}
