package codepipeline

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

func Test__OnPipeline__Setup(t *testing.T) {
	trigger := &OnPipeline{}

	t.Run("invalid configuration decode -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Logger:        logrus.NewEntry(logrus.New()),
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Requests:      &contexts.RequestContext{},
			Configuration: "invalid-config-shape",
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Logger:      logrus.NewEntry(logrus.New()),
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Requests:    &contexts.RequestContext{},
			Configuration: OnPipelineConfiguration{
				Region: "",
			},
		})

		require.Error(t, err)
		assert.EqualError(t, err, "region is required")
	})

	t.Run("subscriptionId already present in metadata -> no-op", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: OnPipelineMetadata{
				Region:         "us-east-1",
				SubscriptionID: "existing-subscription-id",
			},
		}
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
			Configuration: OnPipelineConfiguration{
				Region: "us-east-1",
			},
		})

		require.NoError(t, err)
		assert.Len(t, integrationCtx.ActionRequests, 0)
		assert.Len(t, integrationCtx.Subscriptions, 0)
		assert.Empty(t, requests.Action)
	})

	t.Run("rule missing -> schedules provisionRule and checkRuleAvailability", func(t *testing.T) {
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
			Configuration: OnPipelineConfiguration{
				Region: "us-east-1",
			},
		})

		require.NoError(t, err)

		require.Len(t, integrationCtx.ActionRequests, 1)
		assert.Equal(t, "provisionRule", integrationCtx.ActionRequests[0].ActionName)

		params, ok := integrationCtx.ActionRequests[0].Parameters.(common.ProvisionRuleParameters)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", params.Region)
		assert.Equal(t, Source, params.Source)
		assert.Equal(t, DetailTypePipelineExecutionStateChange, params.DetailType)

		assert.Equal(t, "checkRuleAvailability", requests.Action)
		assert.Equal(t, 5*time.Second, requests.Duration)

		stored, ok := metadata.Get().(OnPipelineMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Empty(t, stored.SubscriptionID)
	})

	t.Run("rule exists -> subscribes and stores subscription ID", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						"aws.codepipeline:us-east-1": {
							Source:      Source,
							DetailTypes: []string{DetailTypePipelineExecutionStateChange},
						},
					},
				},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Logger:      logrus.NewEntry(logrus.New()),
			Integration: integrationCtx,
			Metadata:    metadata,
			Configuration: OnPipelineConfiguration{
				Region: "us-east-1",
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)

		stored, ok := metadata.Get().(OnPipelineMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.NotEmpty(t, stored.SubscriptionID)
	})
}

func Test__OnPipeline__HandleAction(t *testing.T) {
	trigger := &OnPipeline{}

	t.Run("unknown action -> error", func(t *testing.T) {
		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name:        "unknown-action",
			Logger:      logrus.NewEntry(logrus.New()),
			Requests:    &contexts.RequestContext{},
			Metadata:    &contexts.MetadataContext{},
			Integration: &contexts.IntegrationContext{},
		})

		require.Error(t, err)
		assert.EqualError(t, err, "unknown action: unknown-action")
	})

	t.Run("rule still missing -> re-schedules same action", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		metadata := &contexts.MetadataContext{
			Metadata: OnPipelineMetadata{
				Region: "us-east-1",
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{},
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
		assert.Equal(t, "checkRuleAvailability", requests.Action)
		assert.Equal(t, 10*time.Second, requests.Duration)
	})

	t.Run("rule exists -> subscribes and updates metadata subscription ID", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		metadata := &contexts.MetadataContext{
			Metadata: OnPipelineMetadata{
				Region: "us-east-1",
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						"aws.codepipeline:us-east-1": {
							Source:      Source,
							DetailTypes: []string{DetailTypePipelineExecutionStateChange},
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

		stored, ok := metadata.Get().(OnPipelineMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.NotEmpty(t, stored.SubscriptionID)
	})
}

func Test__OnPipeline__OnIntegrationMessage(t *testing.T) {
	trigger := &OnPipeline{}

	t.Run("invalid event payload shape -> error", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnPipelineMetadata{Region: "us-east-1"},
			},
			Configuration: OnPipelineConfiguration{
				Region: "us-east-1",
			},
			Message: "invalid-message-shape",
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to decode message")
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("region mismatch -> ignored", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnPipelineMetadata{Region: "us-east-1"},
			},
			Configuration: OnPipelineConfiguration{
				Region: "us-east-1",
			},
			Message: common.EventBridgeEvent{
				Region: "us-west-2",
				Detail: map[string]any{
					"pipeline":     "deploy-api",
					"execution-id": "exec-123",
					"state":        "SUCCEEDED",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("pipeline filter mismatch -> ignored", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnPipelineMetadata{Region: "us-east-1"},
			},
			Configuration: OnPipelineConfiguration{
				Region: "us-east-1",
				Pipelines: []configuration.Predicate{
					{
						Type:  configuration.PredicateTypeEquals,
						Value: "expected-pipeline",
					},
				},
			},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: map[string]any{
					"pipeline":     "other-pipeline",
					"execution-id": "exec-123",
					"state":        "SUCCEEDED",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("state filter mismatch -> ignored", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnPipelineMetadata{Region: "us-east-1"},
			},
			Configuration: OnPipelineConfiguration{
				Region: "us-east-1",
				States: []string{"FAILED"},
			},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: map[string]any{
					"pipeline":     "deploy-api",
					"execution-id": "exec-123",
					"state":        "SUCCEEDED",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("matching event -> emits aws.codepipeline.pipeline", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		msg := common.EventBridgeEvent{
			Region: "us-east-1",
			Detail: map[string]any{
				"pipeline":     "deploy-api",
				"execution-id": "exec-123",
				"state":        "SUCCEEDED",
			},
		}

		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnPipelineMetadata{Region: "us-east-1"},
			},
			Configuration: OnPipelineConfiguration{
				Region: "us-east-1",
				Pipelines: []configuration.Predicate{
					{
						Type:  configuration.PredicateTypeEquals,
						Value: "deploy-api",
					},
				},
				States: []string{"SUCCEEDED"},
			},
			Message: msg,
		})

		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "aws.codepipeline.pipeline", eventContext.Payloads[0].Type)
		assert.Equal(t, msg, eventContext.Payloads[0].Data)
	})

	t.Run("event state CANCELLED matches configured CANCELED", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		msg := common.EventBridgeEvent{
			Region: "us-east-1",
			Detail: map[string]any{
				"pipeline":     "deploy-api",
				"execution-id": "exec-456",
				"state":        "CANCELLED",
			},
		}

		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnPipelineMetadata{Region: "us-east-1"},
			},
			Configuration: OnPipelineConfiguration{
				Region: "us-east-1",
				States: []string{"CANCELED"},
			},
			Message: msg,
		})

		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "aws.codepipeline.pipeline", eventContext.Payloads[0].Type)
		assert.Equal(t, msg, eventContext.Payloads[0].Data)
	})

	t.Run("event state CANCELED matches configured CANCELLED", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		msg := common.EventBridgeEvent{
			Region: "us-east-1",
			Detail: map[string]any{
				"pipeline":     "deploy-api",
				"execution-id": "exec-789",
				"state":        "CANCELED",
			},
		}

		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnPipelineMetadata{Region: "us-east-1"},
			},
			Configuration: OnPipelineConfiguration{
				Region: "us-east-1",
				States: []string{"CANCELLED"},
			},
			Message: msg,
		})

		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "aws.codepipeline.pipeline", eventContext.Payloads[0].Type)
		assert.Equal(t, msg, eventContext.Payloads[0].Data)
	})
}

func Test__OnPipeline__Metadata(t *testing.T) {
	trigger := &OnPipeline{}

	assert.Equal(t, "aws.codepipeline.onPipeline", trigger.Name())
	assert.Equal(t, "CodePipeline • On Pipeline", trigger.Label())
	assert.Equal(t, "aws", trigger.Icon())
	assert.Equal(t, "gray", trigger.Color())
}

func Test__OnPipeline__Configuration(t *testing.T) {
	trigger := &OnPipeline{}
	fields := trigger.Configuration()

	// quick index by name
	byName := map[string]configuration.Field{}
	for _, f := range fields {
		byName[f.Name] = f
	}

	region, ok := byName["region"]
	require.True(t, ok, "region field should exist")
	assert.Equal(t, configuration.FieldTypeSelect, region.Type)
	assert.True(t, region.Required)
	assert.Equal(t, "us-east-1", region.Default)

	pipelines, ok := byName["pipelines"]
	require.True(t, ok, "pipelines field should exist")
	assert.Equal(t, configuration.FieldTypeAnyPredicateList, pipelines.Type)
	assert.False(t, pipelines.Required)

	states, ok := byName["states"]
	require.True(t, ok, "states field should exist")
	assert.Equal(t, configuration.FieldTypeMultiSelect, states.Type)
	assert.False(t, states.Required)
}
