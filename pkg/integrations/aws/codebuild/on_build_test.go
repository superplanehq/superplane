package codebuild

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnBuild__Setup(t *testing.T) {
	trigger := &OnBuild{}

	t.Run("rule missing -> schedules provisioning and check", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"projects": [
								{
									"name": "backend-build",
									"arn": "arn:aws:codebuild:us-east-1:123456789012:project/backend-build"
								}
							]
						}
					`)),
				},
			},
		}

		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{},
				},
			},
			Secrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Logger:        logrus.NewEntry(logrus.New()),
			HTTP:          httpContext,
			Integration:   integrationCtx,
			Metadata:      metadata,
			Requests:      requests,
			Configuration: OnBuildConfiguration{Region: "us-east-1", Project: "backend-build"},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.ActionRequests, 1)
		assert.Equal(t, "provisionRule", integrationCtx.ActionRequests[0].ActionName)
		params := integrationCtx.ActionRequests[0].Parameters.(common.ProvisionRuleParameters)
		assert.Equal(t, "us-east-1", params.Region)
		assert.Equal(t, Source, params.Source)
		assert.Equal(t, DetailTypeBuildStateChange, params.DetailType)

		assert.Equal(t, "checkRuleAvailability", requests.Action)
		assert.Equal(t, 5*time.Second, requests.Duration)

		stored, ok := metadata.Get().(OnBuildMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.Project)
		assert.Equal(t, "backend-build", stored.Project.ProjectName)
	})

	t.Run("rule available -> subscribes", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"projects": [
								{
									"name": "backend-build",
									"arn": "arn:aws:codebuild:us-east-1:123456789012:project/backend-build"
								}
							]
						}
					`)),
				},
			},
		}

		metadata := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						Source: {
							Source:      Source,
							DetailTypes: []string{DetailTypeBuildStateChange},
						},
					},
				},
			},
			Secrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Logger:        logrus.NewEntry(logrus.New()),
			HTTP:          httpContext,
			Integration:   integrationCtx,
			Metadata:      metadata,
			Configuration: OnBuildConfiguration{Region: "us-east-1", Project: "backend-build"},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)
		stored, ok := metadata.Get().(OnBuildMetadata)
		require.True(t, ok)
		assert.NotEmpty(t, stored.SubscriptionID)
		require.NotNil(t, stored.Project)
		assert.Equal(t, "backend-build", stored.Project.ProjectName)
	})
}

func Test__OnBuild__HandleAction(t *testing.T) {
	trigger := &OnBuild{}

	t.Run("rule missing -> reschedules check", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name:     "checkRuleAvailability",
			Logger:   logrus.NewEntry(logrus.New()),
			Requests: requests,
			Metadata: &contexts.MetadataContext{
				Metadata: OnBuildMetadata{
					Region: "us-east-1",
					Project: &Project{
						ProjectName: "backend-build",
					},
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
			Metadata: OnBuildMetadata{
				Region: "us-east-1",
				Project: &Project{
					ProjectName: "backend-build",
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						Source: {
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
		assert.NotEmpty(t, stored.SubscriptionID)
	})
}

func Test__OnBuild__OnIntegrationMessage(t *testing.T) {
	trigger := &OnBuild{}

	t.Run("project mismatch -> no event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnBuildMetadata{
					Project: &Project{ProjectName: "backend-build"},
				},
			},
			Message: common.EventBridgeEvent{
				Detail: map[string]any{"project-name": "frontend-build"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("project match -> emits event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnBuildMetadata{
					Project: &Project{ProjectName: "backend-build"},
				},
			},
			Message: common.EventBridgeEvent{
				Detail: map[string]any{"project-name": "backend-build"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "aws.codebuild.build", eventContext.Payloads[0].Type)
	})
}
