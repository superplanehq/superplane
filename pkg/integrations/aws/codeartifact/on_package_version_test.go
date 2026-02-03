package codeartifact

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

func Test__OnPackageVersion__Setup(t *testing.T) {
	trigger := &OnPackageVersion{}

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
			Configuration: OnPackageVersionConfiguration{
				Region:         "us-east-1",
				DomainName:     "example-domain",
				RepositoryName: "example-repo",
				PackageName:    "example-package",
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.ActionRequests, 1)
		assert.Equal(t, "provisionRule", integrationCtx.ActionRequests[0].ActionName)
		params := integrationCtx.ActionRequests[0].Parameters.(common.ProvisionRuleParameters)
		assert.Equal(t, "us-east-1", params.Region)
		assert.Equal(t, Source, params.Source)
		assert.Equal(t, DetailTypePackageVersionStateChange, params.DetailType)

		assert.Equal(t, "checkRuleAvailability", requests.Action)
		assert.Equal(t, 5*time.Second, requests.Duration)

		stored, ok := metadata.Get().(OnPackageVersionMetadata)
		require.True(t, ok)
		assert.Equal(t, "example-domain", stored.Filters.DomainName)
		assert.Equal(t, "example-repo", stored.Filters.RepositoryName)
	})

	t.Run("rule available -> subscribes", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						Source: {
							Source:      Source,
							DetailTypes: []string{DetailTypePackageVersionStateChange},
						},
					},
				},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Logger:      logrus.NewEntry(logrus.New()),
			Integration: integrationCtx,
			Metadata:    metadata,
			Configuration: OnPackageVersionConfiguration{
				Region:         "us-east-1",
				DomainName:     "example-domain",
				RepositoryName: "example-repo",
				PackageName:    "example-package",
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)
		stored, ok := metadata.Get().(OnPackageVersionMetadata)
		require.True(t, ok)
		assert.NotEmpty(t, stored.SubscriptionID)
		assert.Equal(t, "example-domain", stored.Filters.DomainName)
		assert.Equal(t, "example-repo", stored.Filters.RepositoryName)
	})
}

func Test__OnPackageVersion__HandleAction(t *testing.T) {
	trigger := &OnPackageVersion{}

	t.Run("rule missing -> reschedules check", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name:     "checkRuleAvailability",
			Logger:   logrus.NewEntry(logrus.New()),
			Requests: requests,
			Metadata: &contexts.MetadataContext{
				Metadata: OnPackageVersionMetadata{
					Region:  "us-east-1",
					Filters: OnPackageVersionConfiguration{Region: "us-east-1"},
				},
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
			Metadata: OnPackageVersionMetadata{
				Region:  "us-east-1",
				Filters: OnPackageVersionConfiguration{Region: "us-east-1", RepositoryName: "example-repo"},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						Source: {
							Source:      Source,
							DetailTypes: []string{DetailTypePackageVersionStateChange},
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
		stored, ok := metadata.Get().(OnPackageVersionMetadata)
		require.True(t, ok)
		assert.NotEmpty(t, stored.SubscriptionID)
	})
}

func Test__OnPackageVersion__OnIntegrationMessage(t *testing.T) {
	trigger := &OnPackageVersion{}

	t.Run("filter mismatch -> no event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			Configuration: OnPackageVersionConfiguration{
				Region:         "us-east-1",
				RepositoryName: "expected-repo",
			},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: map[string]any{"repositoryName": "other-repo"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("filter match -> emits event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			Configuration: OnPackageVersionConfiguration{
				Region:         "us-east-1",
				RepositoryName: "example-repo",
			},
			Message: common.EventBridgeEvent{
				Region: "us-east-1",
				Detail: map[string]any{"repositoryName": "example-repo"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "aws.codeartifact.package.version", eventContext.Payloads[0].Type)
	})
}
