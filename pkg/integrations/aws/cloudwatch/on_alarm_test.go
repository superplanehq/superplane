package cloudwatch

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnAlarm__Setup(t *testing.T) {
	trigger := &OnAlarm{}

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
			Logger:        logrus.NewEntry(logrus.New()),
			Integration:   integrationCtx,
			Metadata:      metadata,
			Requests:      requests,
			Configuration: OnAlarmConfiguration{Region: "us-east-1"},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.ActionRequests, 1)
		assert.Equal(t, "provisionRule", integrationCtx.ActionRequests[0].ActionName)

		params := integrationCtx.ActionRequests[0].Parameters.(common.ProvisionRuleParameters)
		assert.Equal(t, "us-east-1", params.Region)
		assert.Equal(t, Source, params.Source)
		assert.Equal(t, DetailTypeAlarmStateChange, params.DetailType)

		assert.Equal(t, "checkRuleAvailability", requests.Action)
		assert.Equal(t, 5*time.Second, requests.Duration)

		stored, ok := metadata.Get().(OnAlarmMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Empty(t, stored.SubscriptionID)
	})

	t.Run("rule available -> subscribes", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						"aws.cloudwatch:us-east-1": {
							Source:      Source,
							DetailTypes: []string{DetailTypeAlarmStateChange},
						},
					},
				},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Logger:        logrus.NewEntry(logrus.New()),
			Integration:   integrationCtx,
			Metadata:      metadata,
			Configuration: OnAlarmConfiguration{Region: "us-east-1"},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)

		stored, ok := metadata.Get().(OnAlarmMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.NotEmpty(t, stored.SubscriptionID)
	})
}

func Test__OnAlarm__HandleAction(t *testing.T) {
	trigger := &OnAlarm{}

	t.Run("rule missing -> reschedules check", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name:     "checkRuleAvailability",
			Logger:   logrus.NewEntry(logrus.New()),
			Requests: requests,
			Metadata: &contexts.MetadataContext{
				Metadata: OnAlarmMetadata{Region: "us-east-1"},
			},
			Integration: &contexts.IntegrationContext{
				Metadata: common.IntegrationMetadata{
					EventBridge: &common.EventBridgeMetadata{
						Rules: map[string]common.EventBridgeRuleMetadata{},
					},
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "checkRuleAvailability", requests.Action)
		assert.Equal(t, 10*time.Second, requests.Duration)
	})

	t.Run("rule available -> subscribes", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		metadata := &contexts.MetadataContext{
			Metadata: OnAlarmMetadata{Region: "us-east-1"},
		}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						"aws.cloudwatch:us-east-1": {
							Source:      Source,
							DetailTypes: []string{DetailTypeAlarmStateChange},
						},
					},
				},
			},
		}

		_, err := trigger.HandleAction(core.TriggerActionContext{
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
}

func Test__OnAlarm__OnIntegrationMessage(t *testing.T) {
	trigger := &OnAlarm{}

	t.Run("region mismatch -> no event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnAlarmMetadata{Region: "us-east-1"},
			},
			Configuration: OnAlarmConfiguration{
				Alarms: []configuration.Predicate{
					{
						Type:  configuration.PredicateTypeMatches,
						Value: ".*",
					},
				},
			},
			Message: common.EventBridgeEvent{
				Region: "us-west-2",
				Detail: map[string]any{
					"alarmName": "HighCPUUtilization",
					"state": map[string]any{
						"value": "ALARM",
					},
				},
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
				Metadata: OnAlarmMetadata{Region: "us-east-1"},
			},
			Configuration: OnAlarmConfiguration{
				State: AlarmStateAlarm,
				Alarms: []configuration.Predicate{
					{
						Type:  configuration.PredicateTypeMatches,
						Value: ".*",
					},
				},
			},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: map[string]any{
					"alarmName": "HighCPUUtilization",
					"state": map[string]any{
						"value": "OK",
					},
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("alarm does not match predicates -> no event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnAlarmMetadata{Region: "us-east-1"},
			},
			Configuration: OnAlarmConfiguration{
				Alarms: []configuration.Predicate{
					{
						Type:  configuration.PredicateTypeEquals,
						Value: "APIErrorRateHigh",
					},
				},
			},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: map[string]any{
					"alarmName": "HighCPUUtilization",
					"state": map[string]any{
						"value": "ALARM",
					},
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("matching alarm -> emits event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnAlarmMetadata{Region: "us-east-1"},
			},
			Configuration: OnAlarmConfiguration{
				State: AlarmStateAlarm,
				Alarms: []configuration.Predicate{
					{
						Type:  configuration.PredicateTypeEquals,
						Value: "HighCPUUtilization",
					},
				},
			},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: map[string]any{
					"alarmName": "HighCPUUtilization",
					"state": map[string]any{
						"value": "ALARM",
					},
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "aws.cloudwatch.alarm", eventContext.Payloads[0].Type)
	})
}
