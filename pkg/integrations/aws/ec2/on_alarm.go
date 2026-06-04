package ec2

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/cloudwatch"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

// OnAlarmConfiguration holds configuration for the aws.ec2.onAlarm trigger.
// AlarmName is the name of a specific alarm to match; leave empty to match all
// alarms for the configured instance.

type OnAlarm struct{}

type OnAlarmConfiguration struct {
	Region     string `json:"region" mapstructure:"region"`
	InstanceID string `json:"instance" mapstructure:"instance"`
	State      string `json:"state" mapstructure:"state"`
	AlarmName  string `json:"alarm" mapstructure:"alarm"`
}

type OnAlarmMetadata struct {
	Region         string `json:"region" mapstructure:"region"`
	InstanceID     string `json:"instanceId" mapstructure:"instanceId"`
	InstanceName   string `json:"instanceName" mapstructure:"instanceName"`
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
}

type ec2AlarmStateChangeDetail struct {
	AlarmName     string             `mapstructure:"alarmName"`
	State         alarmStateValue    `mapstructure:"state"`
	PreviousState alarmStateValue    `mapstructure:"previousState"`
	Configuration alarmConfiguration `mapstructure:"configuration"`
}

type alarmStateValue struct {
	Value string `mapstructure:"value"`
}

type alarmConfiguration struct {
	Metrics []alarmMetric `mapstructure:"metrics"`
}

type alarmMetric struct {
	MetricStat *alarmMetricStat `mapstructure:"metricStat"`
}

type alarmMetricStat struct {
	Metric alarmMetricDetails `mapstructure:"metric"`
}

type alarmMetricDetails struct {
	Dimensions map[string]any `mapstructure:"dimensions"`
	Name       string         `mapstructure:"name"`
	Namespace  string         `mapstructure:"namespace"`
}

func (p *OnAlarm) Name() string {
	return "aws.ec2.onAlarm"
}

func (p *OnAlarm) Label() string {
	return "EC2 • On Alarm"
}

func (p *OnAlarm) Description() string {
	return "Listen to CloudWatch alarm state change events for a specific EC2 instance"
}

func (p *OnAlarm) Documentation() string {
	return `The On Alarm trigger starts a workflow execution when a CloudWatch alarm attached to a specific EC2 instance transitions to the configured state.

## Use Cases

- **Incident response**: Trigger remediation workflows when a CPU, network, or status-check alarm fires
- **Auto-scaling**: React to alarms on individual instances without polling
- **Audit**: Record alarm state transitions for specific instances over time

## Configuration

- **Region**: AWS region where the instance and alarms reside
- **Instance**: EC2 instance whose alarms to monitor (required)
- **Alarm State**: Only trigger for a specific state: ALARM, OK, or INSUFFICIENT_DATA (required)
- **Alarm**: Optionally restrict to a single alarm selected from the alarms attached to the chosen instance; leave empty to trigger on any alarm for that instance

## How it works

The trigger subscribes to the EventBridge "CloudWatch Alarm State Change" rule and filters
events in two steps:
1. Checks that ` + "`detail.state.value`" + ` matches the configured Alarm State.
2. Checks that the ` + "`InstanceId`" + ` dimension in ` + "`detail.configuration.metrics`" + ` matches the configured instance.
If an Alarm is selected, an additional exact-name check on ` + "`detail.alarmName`" + ` is applied.

## Event Data

Each matched event includes the full EventBridge payload:
- **detail.alarmName**: CloudWatch alarm name
- **detail.state.value**: Current alarm state (ALARM / OK / INSUFFICIENT_DATA)
- **detail.previousState.value**: Previous alarm state
- **detail.configuration**: Full alarm configuration including metric, threshold, and dimensions
- **region**, **account**, **time**: Event envelope fields
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
			Name:        "instance",
			Label:       "Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "EC2 instance whose alarms to monitor",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.instance",
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
		{
			Name:     "state",
			Label:    "Alarm State",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  cloudwatch.AlarmStateAlarm,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: cloudwatch.AllAlarmStates,
				},
			},
		},
		{
			Name:        "alarm",
			Label:       "Alarm",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Filter to a specific alarm for this instance",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "instance", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.instanceAlarm",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
						{
							Name: "instanceId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "instance",
							},
						},
					},
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

	instanceID := strings.TrimSpace(config.InstanceID)
	if instanceID == "" {
		return fmt.Errorf("instance is required")
	}

	state := strings.TrimSpace(config.State)
	if state == "" {
		return fmt.Errorf("alarm state is required")
	}

	if metadata.SubscriptionID != "" && metadata.Region == region && metadata.InstanceID == instanceID {
		return nil
	}

	instanceName := resolveInstanceNameFromTrigger(ctx, region, instanceID)
	regionChanged := metadata.Region != "" && metadata.Region != region

	// Instance filtering is applied in OnIntegrationMessage; keep the existing subscription when only the instance changes.
	if metadata.SubscriptionID != "" && !regionChanged {
		return ctx.Metadata.Set(OnAlarmMetadata{
			Region:         region,
			InstanceID:     instanceID,
			InstanceName:   instanceName,
			SubscriptionID: metadata.SubscriptionID,
		})
	}

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, CloudWatchSource, region, DetailTypeCloudWatchAlarmStateChange)
	if err != nil {
		return fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		updatedMetadata := OnAlarmMetadata{
			Region:       region,
			InstanceID:   instanceID,
			InstanceName: instanceName,
		}

		// Subscribe upserts this node's subscription; update the filter when the region changes
		// so events for the previous region are no longer routed here.
		if metadata.SubscriptionID != "" {
			subscriptionID, err := ctx.Integration.Subscribe(p.subscriptionPattern(region))
			if err != nil {
				return fmt.Errorf("failed to update subscription: %w", err)
			}
			updatedMetadata.SubscriptionID = subscriptionID.String()
		}

		if err := ctx.Metadata.Set(updatedMetadata); err != nil {
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
		InstanceID:     instanceID,
		InstanceName:   instanceName,
		SubscriptionID: subscriptionID.String(),
	})
}

func (p *OnAlarm) provisionRule(integration core.IntegrationContext, requests core.RequestContext, region string) error {
	err := integration.ScheduleActionCall(
		"provisionRule",
		common.ProvisionRuleParameters{
			Region:     region,
			Source:     CloudWatchSource,
			DetailType: DetailTypeCloudWatchAlarmStateChange,
		},
		time.Second,
	)
	if err != nil {
		return fmt.Errorf("failed to schedule rule provisioning for integration: %w", err)
	}

	return requests.ScheduleActionCall("checkRuleAvailability", map[string]any{}, 5*time.Second)
}

func (p *OnAlarm) subscriptionPattern(region string) *common.EventBridgeEvent {
	return &common.EventBridgeEvent{
		Region:     region,
		DetailType: DetailTypeCloudWatchAlarmStateChange,
		Source:     CloudWatchSource,
	}
}

func (p *OnAlarm) Hooks() []core.Hook {
	return []core.Hook{
		{
			Name: "checkRuleAvailability",
			Type: core.HookTypeInternal,
		},
	}
}

func (p *OnAlarm) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	switch ctx.Name {
	case "checkRuleAvailability":
		return p.checkRuleAvailability(ctx)
	default:
		return nil, fmt.Errorf("unknown hook: %s", ctx.Name)
	}
}

func (p *OnAlarm) checkRuleAvailability(ctx core.TriggerHookContext) (map[string]any, error) {
	metadata := OnAlarmMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, CloudWatchSource, metadata.Region, DetailTypeCloudWatchAlarmStateChange)
	if err != nil {
		return nil, fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		return nil, ctx.Requests.ScheduleActionCall("checkRuleAvailability", map[string]any{}, 10*time.Second)
	}

	if metadata.SubscriptionID != "" {
		return nil, nil
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

	detail := ec2AlarmStateChangeDetail{}
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
		ctx.Logger.Infof("Skipping event for alarm %s with state %s, expected %s", alarmName, state, config.State)
		return nil
	}

	if config.AlarmName != "" && alarmName != strings.TrimSpace(config.AlarmName) {
		ctx.Logger.Infof("Skipping event for alarm %s, does not match configured alarm %s", alarmName, config.AlarmName)
		return nil
	}

	instanceID := strings.TrimSpace(config.InstanceID)
	if instanceID != "" && !instanceMatchesAlarm(detail, instanceID) {
		ctx.Logger.Infof("Skipping event for alarm %s, InstanceId dimension does not match %s", alarmName, instanceID)
		return nil
	}

	return ctx.Events.Emit("aws.ec2.alarm", ctx.Message)
}

func (p *OnAlarm) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (p *OnAlarm) Cleanup(_ core.TriggerContext) error {
	return nil
}

func instanceMatchesAlarm(detail ec2AlarmStateChangeDetail, instanceID string) bool {
	for _, m := range detail.Configuration.Metrics {
		if m.MetricStat == nil {
			continue
		}
		if v, ok := m.MetricStat.Metric.Dimensions["InstanceId"]; ok {
			if fmt.Sprintf("%v", v) == strings.TrimSpace(instanceID) {
				return true
			}
		}
	}
	return false
}

func resolveInstanceNameFromTrigger(ctx core.TriggerContext, region, instanceID string) string {
	if ctx.HTTP == nil || ctx.Integration == nil {
		return instanceID
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return instanceID
	}

	instance, err := NewClient(ctx.HTTP, creds, region).DescribeInstance(instanceID)
	if err != nil {
		return instanceID
	}

	name := strings.TrimSpace(instance.Name)
	if name != "" {
		return name
	}

	return instanceID
}
