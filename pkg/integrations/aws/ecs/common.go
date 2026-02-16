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
	ecsTaskStateChangeEventSource     = "aws.ecs"
	ecsTaskStateChangeEventDetailType = "ECS Task State Change"
)

func decodeIntegrationMetadata(metadata any) (common.IntegrationMetadata, error) {
	integrationMetadata := common.IntegrationMetadata{}
	if err := mapstructure.Decode(metadata, &integrationMetadata); err != nil {
		return common.IntegrationMetadata{}, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	return integrationMetadata, nil
}

func hasTaskStateChangeRule(integrationMetadata common.IntegrationMetadata) bool {
	if integrationMetadata.EventBridge == nil {
		return false
	}

	rule, ok := integrationMetadata.EventBridge.Rules[ecsTaskStateChangeEventSource]
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
			Source:     ecsTaskStateChangeEventSource,
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
		Source:     ecsTaskStateChangeEventSource,
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
	integrationMetadata, err := decodeIntegrationMetadata(ctx.Integration.GetMetadata())
	if err != nil {
		return err
	}

	if integrationMetadata.EventBridge == nil {
		ctx.Logger.Infof("EventBridge metadata not found - checking again in %s", retryInterval)
		return ctx.Requests.ScheduleActionCall(checkActionName, map[string]any{}, retryInterval)
	}

	rule, ok := integrationMetadata.EventBridge.Rules[ecsTaskStateChangeEventSource]
	if !ok {
		ctx.Logger.Infof(
			"Rule not found for source %s - checking again in %s",
			ecsTaskStateChangeEventSource,
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

func resolveExecutionByTaskARN(ctx core.IntegrationMessageContext, taskARN string) (*core.ExecutionContext, error) {
	executionCtx, err := ctx.FindExecutionByKV(ecsTaskExecutionKVTaskARN, taskARN)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve execution by task ARN %s: %w", taskARN, err)
	}
	if executionCtx == nil {
		return nil, nil
	}
	if executionCtx.ExecutionState.IsFinished() {
		return nil, nil
	}

	return executionCtx, nil
}
