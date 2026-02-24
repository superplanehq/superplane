package codepipeline

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
	Source                                 = "aws.codepipeline"
	DetailTypePipelineExecutionStateChange = "CodePipeline Pipeline Execution State Change"
)

var AllPipelineExecutionStates = []configuration.FieldOption{
	{Label: "STARTED", Value: "STARTED"},
	{Label: "SUCCEEDED", Value: "SUCCEEDED"},
	{Label: "FAILED", Value: "FAILED"},
	{Label: "CANCELED", Value: "CANCELED"},
	{Label: "STOPPED", Value: "STOPPED"},
	{Label: "SUPERSEDED", Value: "SUPERSEDED"},
	{Label: "RESUMED", Value: "RESUMED"},
}

type OnPipeline struct{}

type OnPipelineConfiguration struct {
	Region    string                    `json:"region" mapstructure:"region"`
	Pipelines []configuration.Predicate `json:"pipelines" mapstructure:"pipelines"`
	States    []string                  `json:"states" mapstructure:"states"`
}

type OnPipelineMetadata struct {
	Region         string `json:"region" mapstructure:"region"`
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
}

func (p *OnPipeline) Name() string {
	return "aws.codepipeline.onPipeline"
}

func (p *OnPipeline) Label() string {
	return "CodePipeline • On Pipeline"
}

func (p *OnPipeline) Description() string {
	return "Listen to AWS CodePipeline pipeline execution state change events"
}

func (p *OnPipeline) Documentation() string {
	return `The On Pipeline trigger starts a workflow execution when AWS CodePipeline emits a pipeline execution state change event.

## Use Cases

- **Deployment visibility**: Start workflows whenever pipeline state changes
- **Incident response**: Notify teams when a pipeline fails or is canceled
- **Workflow orchestration**: Trigger follow-up automations on specific pipeline states

## Configuration

- **Region**: AWS region where pipeline execution events are observed
- **Pipelines**: Optional pipeline name filters (supports equals, not-equals, and regex matches)
- **Pipeline States**: Optional list of states to match (for example STARTED, SUCCEEDED, FAILED)

## Event Data

Each event includes:
- **detail.pipeline**: CodePipeline pipeline name
- **detail.execution-id**: Pipeline execution ID
- **detail.state**: Pipeline execution state
`
}

func (p *OnPipeline) Icon() string {
	return "aws"
}

func (p *OnPipeline) Color() string {
	return "gray"
}

func (p *OnPipeline) Configuration() []configuration.Field {
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
			Name:     "pipelines",
			Label:    "Pipelines",
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
			Label:    "Pipeline States",
			Type:     configuration.FieldTypeMultiSelect,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: AllPipelineExecutionStates,
				},
			},
		},
	}
}

func (p *OnPipeline) Setup(ctx core.TriggerContext) error {
	metadata := OnPipelineMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	config := OnPipelineConfiguration{}
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

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, Source, region, DetailTypePipelineExecutionStateChange)
	if err != nil {
		return fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		if err := ctx.Metadata.Set(OnPipelineMetadata{Region: region}); err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}

		return p.provisionRule(ctx.Integration, ctx.Requests, region)
	}

	subscriptionID, err := ctx.Integration.Subscribe(p.subscriptionPattern(region))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return ctx.Metadata.Set(OnPipelineMetadata{
		Region:         region,
		SubscriptionID: subscriptionID.String(),
	})
}

func (p *OnPipeline) provisionRule(integration core.IntegrationContext, requests core.RequestContext, region string) error {
	err := integration.ScheduleActionCall(
		"provisionRule",
		common.ProvisionRuleParameters{
			Region:     region,
			Source:     Source,
			DetailType: DetailTypePipelineExecutionStateChange,
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

func (p *OnPipeline) subscriptionPattern(region string) *common.EventBridgeEvent {
	return &common.EventBridgeEvent{
		Region:     region,
		DetailType: DetailTypePipelineExecutionStateChange,
		Source:     Source,
	}
}

func (p *OnPipeline) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "checkRuleAvailability",
			Description: "Check if the EventBridge rule is available",
		},
	}
}

func (p *OnPipeline) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case "checkRuleAvailability":
		return p.checkRuleAvailability(ctx)
	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (p *OnPipeline) checkRuleAvailability(ctx core.TriggerActionContext) (map[string]any, error) {
	metadata := OnPipelineMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, Source, metadata.Region, DetailTypePipelineExecutionStateChange)
	if err != nil {
		return nil, fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		return nil, ctx.Requests.ScheduleActionCall(ctx.Name, map[string]any{}, 10*time.Second)
	}

	subscriptionID, err := ctx.Integration.Subscribe(p.subscriptionPattern(metadata.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	metadata.SubscriptionID = subscriptionID.String()
	return nil, ctx.Metadata.Set(metadata)
}

func (p *OnPipeline) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	metadata := OnPipelineMetadata{}
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	config := OnPipelineConfiguration{}
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

	pipeline, ok := event.Detail["pipeline"].(string)
	if !ok || pipeline == "" {
		return fmt.Errorf("missing pipeline name in event")
	}

	state, ok := event.Detail["state"].(string)
	if !ok || state == "" {
		return fmt.Errorf("missing pipeline state in event")
	}
	// Normalize event spelling variants (e.g. CANCELLED -> CANCELED).
	state = normalizePipelineExecutionState(state)

	if len(config.Pipelines) > 0 {
		if !configuration.MatchesAnyPredicate(config.Pipelines, pipeline) {
			ctx.Logger.Infof("Skipping event for pipeline %s, does not match any predicate: %v", pipeline, config.Pipelines)
			return nil
		}
	}

	if len(config.States) > 0 {
		// Normalize configured values so both CANCELED and CANCELLED match.
		if !slices.Contains(normalizePipelineExecutionStates(config.States), state) {
			ctx.Logger.Infof("Skipping event for pipeline %s with state %s", pipeline, state)
			return nil
		}
	}

	return ctx.Events.Emit("aws.codepipeline.pipeline", ctx.Message)
}

func (p *OnPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (p *OnPipeline) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func normalizePipelineExecutionState(state string) string {
	value := strings.ToUpper(strings.TrimSpace(state))
	// Keep a single canonical cancellation token for comparisons.
	if value == "CANCELLED" {
		return "CANCELED"
	}

	return value
}

func normalizePipelineExecutionStates(states []string) []string {
	// Apply the same normalization rules to all configured states.
	normalized := make([]string, 0, len(states))
	for _, state := range states {
		normalized = append(normalized, normalizePipelineExecutionState(state))
	}

	return normalized
}
