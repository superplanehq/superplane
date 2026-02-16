package ecs

import (
	"fmt"
	"slices"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	ecsTaskPayloadType                = "aws.ecs.task"
	ecsTaskExecutionKVTaskARN         = "aws_ecs_task_arn"
	ecsEventBridgeSource              = "aws.ecs"
	ecsTaskStateChangeEventDetailType = "ECS Task State Change"
)

func hasTaskStateChangeRule(integrationMetadata common.IntegrationMetadata) bool {
	if integrationMetadata.EventBridge == nil {
		return false
	}

	rule, ok := integrationMetadata.EventBridge.Rules[ecsEventBridgeSource]
	if !ok {
		return false
	}

	return slices.Contains(rule.DetailTypes, ecsTaskStateChangeEventDetailType)
}

func scheduleTaskStateChangeRuleProvision(
	integration core.IntegrationContext,
	requests core.RequestContext,
	checkActionName string,
	initialCheckInterval time.Duration,
	region string,
) error {
	if err := integration.ScheduleActionCall(
		"provisionRule",
		common.ProvisionRuleParameters{
			Region:     region,
			Source:     ecsEventBridgeSource,
			DetailType: ecsTaskStateChangeEventDetailType,
		},
		time.Second,
	); err != nil {
		return fmt.Errorf("failed to schedule rule provisioning for integration: %w", err)
	}

	return requests.ScheduleActionCall(
		checkActionName,
		map[string]any{},
		initialCheckInterval,
	)
}

func taskStateChangeSubscriptionPattern(region string, detail map[string]any) *common.EventBridgeEvent {
	return &common.EventBridgeEvent{
		Region:     region,
		DetailType: ecsTaskStateChangeEventDetailType,
		Source:     ecsEventBridgeSource,
		Detail:     detail,
	}
}

func subscribeWhenTaskStateChangeRuleAvailable(
	ctx core.ActionContext,
	checkActionName string,
	retryInterval time.Duration,
	subscriptionPattern *common.EventBridgeEvent,
	onSubscribed func(subscriptionID string) error,
) error {
	integrationMetadata := common.IntegrationMetadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	if integrationMetadata.EventBridge == nil {
		ctx.Logger.Infof("EventBridge metadata not found - checking again in %s", retryInterval)
		return ctx.Requests.ScheduleActionCall(checkActionName, map[string]any{}, retryInterval)
	}

	rule, ok := integrationMetadata.EventBridge.Rules[ecsEventBridgeSource]
	if !ok {
		ctx.Logger.Infof(
			"Rule not found for source %s - checking again in %s",
			ecsEventBridgeSource,
			retryInterval,
		)
		return ctx.Requests.ScheduleActionCall(checkActionName, map[string]any{}, retryInterval)
	}

	if !slices.Contains(rule.DetailTypes, ecsTaskStateChangeEventDetailType) {
		ctx.Logger.Infof(
			"Rule does not have detail type %q - checking again in %s",
			ecsTaskStateChangeEventDetailType,
			retryInterval,
		)
		return ctx.Requests.ScheduleActionCall(checkActionName, map[string]any{}, retryInterval)
	}

	subscriptionID, err := ctx.Integration.Subscribe(subscriptionPattern)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return onSubscribed(subscriptionID.String())
}
