package codebuild

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

func Test__OnBuild__Setup(t *testing.T) {
	trigger := &OnBuild{}

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
			Configuration: OnBuildConfiguration{
				Region: "",
			},
		})

		require.Error(t, err)
		assert.EqualError(t, err, "region is required")
	})

	t.Run("subscriptionId already present in metadata -> no-op", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: OnBuildMetadata{
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
			Configuration: OnBuildConfiguration{
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
			Configuration: OnBuildConfiguration{
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
		assert.Equal(t, DetailTypeBuildStateChange, params.DetailType)

		assert.Equal(t, "checkRuleAvailability", requests.Action)
		assert.Equal(t, 5*time.Second, requests.Duration)

		stored, ok := metadata.Get().(OnBuildMetadata)
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
						"aws.codebuild:us-east-1": {
							Source:      Source,
							DetailTypes: []string{DetailTypeBuildStateChange},
						},
					},
				},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Logger:      logrus.NewEntry(logrus.New()),
			Integration: integrationCtx,
			Metadata:    metadata,
			Configuration: OnBuildConfiguration{
				Region: "us-east-1",
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)

		stored, ok := metadata.Get().(OnBuildMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.NotEmpty(t, stored.SubscriptionID)
	})
}

func Test__OnBuild__HandleAction(t *testing.T) {
	trigger := &OnBuild{}

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
			Metadata: OnBuildMetadata{
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
			Metadata: OnBuildMetadata{
				Region: "us-east-1",
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						"aws.codebuild:us-east-1": {
							Source:      Source,
							DetailTypes: []string{DetailTypeBuildStateChange},
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

		stored, ok := metadata.Get().(OnBuildMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.NotEmpty(t, stored.SubscriptionID)
	})
}

func Test__OnBuild__OnIntegrationMessage(t *testing.T) {
	trigger := &OnBuild{}

	t.Run("invalid event payload shape -> error", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnBuildMetadata{Region: "us-east-1"},
			},
			Configuration: OnBuildConfiguration{
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
				Metadata: OnBuildMetadata{Region: "us-east-1"},
			},
			Configuration: OnBuildConfiguration{
				Region: "us-east-1",
			},
			Message: common.EventBridgeEvent{
				Region: "us-west-2",
				Detail: map[string]any{
					"project-name": "my-project",
					"build-id":     "my-project:build-123",
					"build-status": "SUCCEEDED",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("project filter mismatch -> ignored", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnBuildMetadata{Region: "us-east-1"},
			},
			Configuration: OnBuildConfiguration{
				Region: "us-east-1",
				Projects: []configuration.Predicate{
					{
						Type:  configuration.PredicateTypeEquals,
						Value: "expected-project",
					},
				},
			},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: map[string]any{
					"project-name": "other-project",
					"build-id":     "other-project:build-123",
					"build-status": "SUCCEEDED",
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
				Metadata: OnBuildMetadata{Region: "us-east-1"},
			},
			Configuration: OnBuildConfiguration{
				Region: "us-east-1",
				States: []string{"FAILED"},
			},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: map[string]any{
					"project-name": "my-project",
					"build-id":     "my-project:build-123",
					"build-status": "SUCCEEDED",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("matching event -> emits aws.codebuild.build", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		msg := common.EventBridgeEvent{
			Region: "us-east-1",
			Detail: map[string]any{
				"project-name": "my-project",
				"build-id":     "my-project:build-123",
				"build-status": "SUCCEEDED",
			},
		}

		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnBuildMetadata{Region: "us-east-1"},
			},
			Configuration: OnBuildConfiguration{
				Region: "us-east-1",
				Projects: []configuration.Predicate{
					{
						Type:  configuration.PredicateTypeEquals,
						Value: "my-project",
					},
				},
				States: []string{"SUCCEEDED"},
			},
			Message: msg,
		})

		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "aws.codebuild.build", eventContext.Payloads[0].Type)
		assert.Equal(t, msg, eventContext.Payloads[0].Data)
	})
}

func Test__OnBuild__Metadata(t *testing.T) {
	trigger := &OnBuild{}

	assert.Equal(t, "aws.codebuild.onBuild", trigger.Name())
	assert.Equal(t, "CodeBuild • On Build Completed", trigger.Label())
	assert.Equal(t, "aws", trigger.Icon())
	assert.Equal(t, "gray", trigger.Color())
}

func Test__OnBuild__Configuration(t *testing.T) {
	trigger := &OnBuild{}
	fields := trigger.Configuration()

	byName := map[string]configuration.Field{}
	for _, f := range fields {
		byName[f.Name] = f
	}

	region, ok := byName["region"]
	require.True(t, ok, "region field should exist")
	assert.Equal(t, configuration.FieldTypeSelect, region.Type)
	assert.True(t, region.Required)
	assert.Equal(t, "us-east-1", region.Default)

	projects, ok := byName["projects"]
	require.True(t, ok, "projects field should exist")
	assert.Equal(t, configuration.FieldTypeAnyPredicateList, projects.Type)
	assert.False(t, projects.Required)

	states, ok := byName["states"]
	require.True(t, ok, "states field should exist")
	assert.Equal(t, configuration.FieldTypeMultiSelect, states.Type)
	assert.False(t, states.Required)
}
