package ec2

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnImage__Setup(t *testing.T) {
	trigger := &OnImage{}

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
			Configuration: OnImageConfiguration{
				Region: "us-east-1",
				States: []string{ImageStateAvailable},
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.ActionRequests, 1)
		assert.Equal(t, "provisionRule", integrationCtx.ActionRequests[0].ActionName)

		params := integrationCtx.ActionRequests[0].Parameters.(common.ProvisionRuleParameters)
		assert.Equal(t, "us-east-1", params.Region)
		assert.Equal(t, Source, params.Source)
		assert.Equal(t, DetailTypeAMIStateChange, params.DetailType)

		assert.Equal(t, "checkRuleAvailability", requests.Action)
		assert.Equal(t, 5*time.Second, requests.Duration)

		stored, ok := metadata.Get().(OnImageMetadata)
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
						"aws.ec2:us-east-1": {
							Source:      Source,
							DetailTypes: []string{DetailTypeAMIStateChange},
						},
					},
				},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Logger:      logrus.NewEntry(logrus.New()),
			Integration: integrationCtx,
			Metadata:    metadata,
			Configuration: OnImageConfiguration{
				Region: "us-east-1",
				States: []string{ImageStateAvailable},
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)

		stored, ok := metadata.Get().(OnImageMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.NotEmpty(t, stored.SubscriptionID)
	})
}

func Test__OnImage__HandleAction(t *testing.T) {
	trigger := &OnImage{}

	t.Run("rule missing -> reschedules check", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name:     "checkRuleAvailability",
			Logger:   logrus.NewEntry(logrus.New()),
			Requests: requests,
			Metadata: &contexts.MetadataContext{Metadata: OnImageMetadata{Region: "us-east-1"}},
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
		metadata := &contexts.MetadataContext{Metadata: OnImageMetadata{Region: "us-east-1"}}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						"aws.ec2:us-east-1": {
							Source:      Source,
							DetailTypes: []string{DetailTypeAMIStateChange},
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
		stored, ok := metadata.Get().(OnImageMetadata)
		require.True(t, ok)
		assert.NotEmpty(t, stored.SubscriptionID)
	})
}

func Test__OnImage__OnIntegrationMessage(t *testing.T) {
	trigger := &OnImage{}

	t.Run("region mismatch -> no event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnImageMetadata{Region: "us-east-1"},
			},
			Configuration: OnImageConfiguration{States: []string{ImageStateAvailable}},
			Message: common.EventBridgeEvent{
				Region: "us-west-2",
				Detail: map[string]any{
					"ImageId": "ami-123",
					"State":   ImageStateAvailable,
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
				Metadata: OnImageMetadata{Region: "us-east-1"},
			},
			Configuration: OnImageConfiguration{States: []string{ImageStateAvailable}},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: map[string]any{
					"ImageId": "ami-123",
					"State":   ImageStatePending,
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("matching event -> emits", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnImageMetadata{Region: "us-east-1"},
			},
			Configuration: OnImageConfiguration{States: []string{ImageStateAvailable}},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: map[string]any{
					"ImageId": "ami-123",
					"State":   ImageStateAvailable,
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "aws.ec2.image", eventContext.Payloads[0].Type)
	})
}
