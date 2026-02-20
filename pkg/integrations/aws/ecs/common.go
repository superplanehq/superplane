package ecs

import (
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	ecsTaskPayloadType                = "aws.ecs.task"
	ecsTaskExecutionKVTaskARN         = "aws_ecs_task_arn"
	ecsEventBridgeSource              = "aws.ecs"
	ecsTaskStateChangeEventDetailType = "ECS Task State Change"
)

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
	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, ecsEventBridgeSource, subscriptionPattern.Region, subscriptionPattern.DetailType)
	if err != nil {
		return fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		return ctx.Requests.ScheduleActionCall(checkActionName, map[string]any{}, retryInterval)
	}

	subscriptionID, err := ctx.Integration.Subscribe(subscriptionPattern)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return onSubscribed(subscriptionID.String())
}
