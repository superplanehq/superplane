package cloudwatch

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

const (
	Source                     = "aws.cloudwatch"
	DetailTypeAlarmStateChange = "CloudWatch Alarm State Change"
)

type OnAlarm struct{}

type OnAlarmConfiguration struct {
	Region string                    `json:"region" mapstructure:"region"`
	Alarms []configuration.Predicate `json:"alarms" mapstructure:"alarms"`
	State  string                    `json:"state" mapstructure:"state"`
}

type OnAlarmMetadata struct {
	Region         string `json:"region" mapstructure:"region"`
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
}

type AlarmStateChangeDetail struct {
	AlarmName     string     `json:"alarmName" mapstructure:"alarmName"`
	State         AlarmState `json:"state" mapstructure:"state"`
	PreviousState AlarmState `json:"previousState" mapstructure:"previousState"`
}

type AlarmState struct {
	Value     string `json:"value" mapstructure:"value"`
	Reason    string `json:"reason" mapstructure:"reason"`
	Timestamp string `json:"timestamp" mapstructure:"timestamp"`
}

func (p *OnAlarm) Name() string {
	return "aws.cloudwatch.onAlarm"
}

func (p *OnAlarm) Label() string {
	return "CloudWatch • On Alarm"
}

func (p *OnAlarm) Description() string {
	return "Listen to AWS CloudWatch alarm state change events"
}

func (p *OnAlarm) Documentation() string {
	return `The On Alarm trigger starts a workflow execution when a CloudWatch alarm transitions to the ALARM state.

## Use Cases

- **Incident response**: Notify responders and open incidents when alarms fire
- **Auto-remediation**: Execute rollback or recovery workflows immediately
- **Audit and reporting**: Track alarm transitions over time

## Configuration

- **Region**: AWS region where alarms are evaluated
- **Alarms**: Optional alarm name filters (supports equals, not-equals, and regex matches)
- **State**: Only trigger for alarms in the specified state (OK, ALARM, or INSUFFICIENT_DATA)

## Event Data

Each alarm event includes:
- **detail.alarmName**: CloudWatch alarm name
- **detail.state.value**: Current alarm state
- **detail.previousState.value**: Previous alarm state
`
}

func (p *OnAlarm) Icon() string {
	return "aws"
}

func (p *OnAlarm) Color() string {
	return "gray"
}

func (p *OnAlarm) Configuration() []configuration.Field {
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
			Name:     "state",
			Label:    "Alarm State",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  AlarmStateAlarm,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: AllAlarmStates,
				},
			},
		},
		{
			Name:     "alarms",
			Label:    "Alarms",
			Type:     configuration.FieldTypeAnyPredicateList,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
	}
}

func (p *OnAlarm) Setup(ctx core.TriggerContext) error {
	metadata := OnAlarmMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	config := OnAlarmConfiguration{}
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

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, Source, region, DetailTypeAlarmStateChange)
	if err != nil {
		return fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		if err := ctx.Metadata.Set(OnAlarmMetadata{Region: region}); err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}

		return p.provisionRule(ctx.Integration, ctx.Requests, region)
	}

	subscriptionID, err := ctx.Integration.Subscribe(p.subscriptionPattern(region))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return ctx.Metadata.Set(OnAlarmMetadata{
		Region:         region,
		SubscriptionID: subscriptionID.String(),
	})
}

func (p *OnAlarm) provisionRule(integration core.IntegrationContext, requests core.RequestContext, region string) error {
	err := integration.ScheduleActionCall(
		"provisionRule",
		common.ProvisionRuleParameters{
			Region:     region,
			Source:     Source,
			DetailType: DetailTypeAlarmStateChange,
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

func (p *OnAlarm) subscriptionPattern(region string) *common.EventBridgeEvent {
	return &common.EventBridgeEvent{
		Region:     region,
		DetailType: DetailTypeAlarmStateChange,
		Source:     Source,
	}
}

func (p *OnAlarm) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "checkRuleAvailability",
			Description: "Check if the EventBridge rule is available",
		},
	}
}

func (p *OnAlarm) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case "checkRuleAvailability":
		return p.checkRuleAvailability(ctx)

	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (p *OnAlarm) checkRuleAvailability(ctx core.TriggerActionContext) (map[string]any, error) {
	metadata := OnAlarmMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, Source, metadata.Region, DetailTypeAlarmStateChange)
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

func (p *OnAlarm) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	metadata := OnAlarmMetadata{}
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	config := OnAlarmConfiguration{}
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

	detail := AlarmStateChangeDetail{}
	if err := mapstructure.Decode(event.Detail, &detail); err != nil {
		return fmt.Errorf("failed to decode event detail: %w", err)
	}

	alarmName := strings.TrimSpace(detail.AlarmName)
	if alarmName == "" {
		return fmt.Errorf("missing alarm name in event")
	}

	state := strings.TrimSpace(detail.State.Value)
	if state == "" {
		return fmt.Errorf("missing alarm state in event")
	}

	if state != config.State {
		ctx.Logger.Infof("Skipping event for alarm %s with state %s", alarmName, state)
		return nil
	}

	if len(config.Alarms) > 0 {
		if !configuration.MatchesAnyPredicate(config.Alarms, alarmName) {
			ctx.Logger.Infof("Skipping event for alarm %s, does not match any predicate: %v", alarmName, config.Alarms)
			return nil
		}
	}

	return ctx.Events.Emit("aws.cloudwatch.alarm", ctx.Message)
}

func (p *OnAlarm) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// no-op, since events are received through the integration
	// and routed to OnIntegrationMessage()
	return http.StatusOK, nil
}

func (p *OnAlarm) Cleanup(ctx core.TriggerContext) error {
	return nil
}
