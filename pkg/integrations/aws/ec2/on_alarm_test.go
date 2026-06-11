package ec2

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/cloudwatch"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__EC2OnAlarm__Setup(t *testing.T) {
	trigger := &OnAlarm{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: OnAlarmConfiguration{
				Region:     " ",
				InstanceID: "i-abc123",
				State:      cloudwatch.AlarmStateAlarm,
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing instance -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: OnAlarmConfiguration{
				Region:     "us-east-1",
				InstanceID: " ",
				State:      cloudwatch.AlarmStateAlarm,
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "instance is required")
	})

	t.Run("missing state -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: OnAlarmConfiguration{
				Region:     "us-east-1",
				InstanceID: "i-abc123",
				State:      " ",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "alarm state is required")
	})

	t.Run("rule missing -> schedules provisioning and check", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{},
				},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Logger:      logrus.NewEntry(logrus.New()),
			Integration: integrationCtx,
			Metadata:    metadata,
			Requests:    requests,
			Configuration: OnAlarmConfiguration{
				Region:     "us-east-1",
				InstanceID: "i-abc123",
				State:      cloudwatch.AlarmStateAlarm,
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.ActionRequests, 1)
		assert.Equal(t, "provisionRule", integrationCtx.ActionRequests[0].ActionName)

		params := integrationCtx.ActionRequests[0].Parameters.(common.ProvisionRuleParameters)
		assert.Equal(t, "us-east-1", params.Region)
		assert.Equal(t, CloudWatchSource, params.Source)
		assert.Equal(t, DetailTypeCloudWatchAlarmStateChange, params.DetailType)

		assert.Equal(t, "checkRuleAvailability", requests.Action)
		assert.Equal(t, 5*time.Second, requests.Duration)

		stored, ok := metadata.Get().(OnAlarmMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Equal(t, "i-abc123", stored.InstanceID)
		assert.Empty(t, stored.SubscriptionID)
	})

	t.Run("rule available -> subscribes", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						"aws.cloudwatch:us-east-1": {
							Source:      CloudWatchSource,
							DetailTypes: []string{DetailTypeCloudWatchAlarmStateChange},
						},
					},
				},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Logger:      logrus.NewEntry(logrus.New()),
			Integration: integrationCtx,
			Metadata:    metadata,
			Configuration: OnAlarmConfiguration{
				Region:     "us-east-1",
				InstanceID: "i-abc123",
				State:      cloudwatch.AlarmStateAlarm,
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)

		stored, ok := metadata.Get().(OnAlarmMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Equal(t, "i-abc123", stored.InstanceID)
		assert.NotEmpty(t, stored.SubscriptionID)
	})

	t.Run("already subscribed with same config -> no-op", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: OnAlarmMetadata{
				Region:         "us-east-1",
				InstanceID:     "i-abc123",
				SubscriptionID: "existing-sub",
			},
		}
		integrationCtx := &contexts.IntegrationContext{}

		err := trigger.Setup(core.TriggerContext{
			Logger:        logrus.NewEntry(logrus.New()),
			Integration:   integrationCtx,
			Metadata:      metadata,
			Configuration: OnAlarmConfiguration{Region: "us-east-1", InstanceID: "i-abc123", State: cloudwatch.AlarmStateAlarm},
		})

		require.NoError(t, err)
		assert.Len(t, integrationCtx.Subscriptions, 0)
	})

	t.Run("subscribed but region changed -> re-subscribes", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: OnAlarmMetadata{
				Region:         "us-east-1",
				InstanceID:     "i-abc123",
				SubscriptionID: "existing-sub",
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						"aws.cloudwatch:eu-west-1": {
							Source:      CloudWatchSource,
							DetailTypes: []string{DetailTypeCloudWatchAlarmStateChange},
						},
					},
				},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Logger:      logrus.NewEntry(logrus.New()),
			Integration: integrationCtx,
			Metadata:    metadata,
			Configuration: OnAlarmConfiguration{
				Region:     "eu-west-1",
				InstanceID: "i-abc123",
				State:      "ALARM",
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)
		stored, ok := metadata.Get().(OnAlarmMetadata)
		require.True(t, ok)
		assert.Equal(t, "eu-west-1", stored.Region)
		assert.NotEmpty(t, stored.SubscriptionID)
	})

	t.Run("subscribed but instance changed -> updates metadata without re-subscribing", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: OnAlarmMetadata{
				Region:         "us-east-1",
				InstanceID:     "i-abc123",
				SubscriptionID: "existing-sub",
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						"aws.cloudwatch:us-east-1": {
							Source:      CloudWatchSource,
							DetailTypes: []string{DetailTypeCloudWatchAlarmStateChange},
						},
					},
				},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Logger:      logrus.NewEntry(logrus.New()),
			Integration: integrationCtx,
			Metadata:    metadata,
			Configuration: OnAlarmConfiguration{
				Region:     "us-east-1",
				InstanceID: "i-new456",
				State:      "ALARM",
			},
		})

		require.NoError(t, err)
		assert.Empty(t, integrationCtx.Subscriptions)
		stored, ok := metadata.Get().(OnAlarmMetadata)
		require.True(t, ok)
		assert.Equal(t, "i-new456", stored.InstanceID)
		assert.Equal(t, "existing-sub", stored.SubscriptionID)
	})
}

func Test__EC2OnAlarm__HandleHook(t *testing.T) {
	trigger := &OnAlarm{}

	t.Run("rule missing -> reschedules check", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		_, err := trigger.HandleHook(core.TriggerHookContext{
			Name:     "checkRuleAvailability",
			Logger:   logrus.NewEntry(logrus.New()),
			Requests: requests,
			Metadata: &contexts.MetadataContext{Metadata: OnAlarmMetadata{Region: "us-east-1"}},
			Integration: &contexts.IntegrationContext{
				Metadata: common.IntegrationMetadata{
					EventBridge: &common.EventBridgeMetadata{Rules: map[string]common.EventBridgeRuleMetadata{}},
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "checkRuleAvailability", requests.Action)
		assert.Equal(t, 10*time.Second, requests.Duration)
	})

	t.Run("rule available -> subscribes", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		metadata := &contexts.MetadataContext{Metadata: OnAlarmMetadata{Region: "us-east-1", InstanceID: "i-abc123"}}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						"aws.cloudwatch:us-east-1": {
							Source:      CloudWatchSource,
							DetailTypes: []string{DetailTypeCloudWatchAlarmStateChange},
						},
					},
				},
			},
		}

		_, err := trigger.HandleHook(core.TriggerHookContext{
			Name:        "checkRuleAvailability",
			Logger:      logrus.NewEntry(logrus.New()),
			Requests:    requests,
			Metadata:    metadata,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)

		stored, ok := metadata.Get().(OnAlarmMetadata)
		require.True(t, ok)
		assert.NotEmpty(t, stored.SubscriptionID)
	})

	t.Run("already subscribed -> no-op", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		metadata := &contexts.MetadataContext{
			Metadata: OnAlarmMetadata{
				Region:         "us-east-1",
				InstanceID:     "i-abc123",
				SubscriptionID: "existing-sub",
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Subscriptions: []contexts.Subscription{{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")}},
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						"aws.cloudwatch:us-east-1": {
							Source:      CloudWatchSource,
							DetailTypes: []string{DetailTypeCloudWatchAlarmStateChange},
						},
					},
				},
			},
		}

		_, err := trigger.HandleHook(core.TriggerHookContext{
			Name:        "checkRuleAvailability",
			Logger:      logrus.NewEntry(logrus.New()),
			Requests:    requests,
			Metadata:    metadata,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, integrationCtx.Subscriptions, 1)
	})
}

func Test__EC2OnAlarm__OnIntegrationMessage(t *testing.T) {
	trigger := &OnAlarm{}

	alarmDetail := func(alarmName, state, instanceID string) map[string]any {
		return map[string]any{
			"alarmName":     alarmName,
			"state":         map[string]any{"value": state},
			"previousState": map[string]any{"value": "OK"},
			"configuration": map[string]any{
				"metrics": []any{
					map[string]any{
						"metricStat": map[string]any{
							"metric": map[string]any{
								"dimensions": map[string]any{
									"InstanceId": instanceID,
								},
								"name":      "CPUUtilization",
								"namespace": "AWS/EC2",
							},
						},
					},
				},
			},
		}
	}

	t.Run("region mismatch -> no event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnAlarmMetadata{Region: "us-east-1", InstanceID: "i-abc123"},
			},
			Configuration: OnAlarmConfiguration{
				State:      cloudwatch.AlarmStateAlarm,
				InstanceID: "i-abc123",
			},
			Message: common.EventBridgeEvent{
				Region: "us-west-2",
				Detail: alarmDetail("HighCPU", cloudwatch.AlarmStateAlarm, "i-abc123"),
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("empty config state -> no event (never matches)", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnAlarmMetadata{Region: "us-east-1", InstanceID: "i-abc123"},
			},
			Configuration: OnAlarmConfiguration{
				State:      "",
				InstanceID: "i-abc123",
			},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: alarmDetail("HighCPU", cloudwatch.AlarmStateAlarm, "i-abc123"),
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("state mismatch -> no event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnAlarmMetadata{Region: "us-east-1", InstanceID: "i-abc123"},
			},
			Configuration: OnAlarmConfiguration{
				State:      cloudwatch.AlarmStateAlarm,
				InstanceID: "i-abc123",
			},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: alarmDetail("HighCPU", cloudwatch.AlarmStateOK, "i-abc123"),
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("alarm name mismatch -> no event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnAlarmMetadata{Region: "us-east-1", InstanceID: "i-abc123"},
			},
			Configuration: OnAlarmConfiguration{
				State:      cloudwatch.AlarmStateAlarm,
				InstanceID: "i-abc123",
				AlarmName:  "OtherAlarm",
			},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: alarmDetail("HighCPU", cloudwatch.AlarmStateAlarm, "i-abc123"),
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("instance ID mismatch -> no event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnAlarmMetadata{Region: "us-east-1", InstanceID: "i-abc123"},
			},
			Configuration: OnAlarmConfiguration{
				State:      cloudwatch.AlarmStateAlarm,
				InstanceID: "i-abc123",
			},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: alarmDetail("HighCPU", cloudwatch.AlarmStateAlarm, "i-different"),
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("all filters match -> emits event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnAlarmMetadata{Region: "us-east-1", InstanceID: "i-abc123"},
			},
			Configuration: OnAlarmConfiguration{
				State:      cloudwatch.AlarmStateAlarm,
				InstanceID: "i-abc123",
				AlarmName:  "HighCPU",
			},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: alarmDetail("HighCPU", cloudwatch.AlarmStateAlarm, "i-abc123"),
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "aws.ec2.alarm", eventContext.Payloads[0].Type)
	})

	t.Run("no alarm name filter -> emits on any alarm for instance", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnAlarmMetadata{Region: "us-east-1", InstanceID: "i-abc123"},
			},
			Configuration: OnAlarmConfiguration{
				State:      cloudwatch.AlarmStateAlarm,
				InstanceID: "i-abc123",
				AlarmName:  "",
			},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: alarmDetail("AnyAlarm", cloudwatch.AlarmStateAlarm, "i-abc123"),
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "aws.ec2.alarm", eventContext.Payloads[0].Type)
	})
}
