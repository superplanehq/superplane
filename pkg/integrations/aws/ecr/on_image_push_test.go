package ecr

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

func Test__OnImagePush__Setup(t *testing.T) {
	trigger := &OnImagePush{}

	t.Run("rule missing -> schedules provisioning and check", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"repositories": [
								{
									"repositoryName": "backend",
									"repositoryArn": "arn:aws:ecr:us-east-1:123456789012:repository/backend"
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
			Configuration: OnImagePushConfiguration{Region: "us-east-1", Repository: "backend"},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.ActionRequests, 1)
		assert.Equal(t, "provisionRule", integrationCtx.ActionRequests[0].ActionName)
		params := integrationCtx.ActionRequests[0].Parameters.(common.ProvisionRuleParameters)
		assert.Equal(t, "us-east-1", params.Region)
		assert.Equal(t, Source, params.Source)
		assert.Equal(t, DetailTypeECRImageAction, params.DetailType)

		assert.Equal(t, "checkRuleAvailability", requests.Action)
		assert.Equal(t, 5*time.Second, requests.Duration)

		stored, ok := metadata.Get().(OnImagePushMetadata)
		require.True(t, ok)
		assert.Equal(t, "backend", stored.Repository.RepositoryName)
	})

	t.Run("rule available -> subscribes", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"repositories": [
								{
									"repositoryName": "backend",
									"repositoryArn": "arn:aws:ecr:us-east-1:123456789012:repository/backend"
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
						"aws.ecr:us-east-1": {
							Source:      Source,
							DetailTypes: []string{DetailTypeECRImageAction},
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
			Configuration: OnImagePushConfiguration{Region: "us-east-1", Repository: "backend"},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)
		stored, ok := metadata.Get().(OnImagePushMetadata)
		require.True(t, ok)
		assert.NotEmpty(t, stored.SubscriptionID)
		assert.Equal(t, "backend", stored.Repository.RepositoryName)
	})
}

func Test__OnImagePush__HandleAction(t *testing.T) {
	trigger := &OnImagePush{}

	t.Run("rule missing -> reschedules check", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name:     "checkRuleAvailability",
			Logger:   logrus.NewEntry(logrus.New()),
			Requests: requests,
			Metadata: &contexts.MetadataContext{
				Metadata: OnImagePushMetadata{Region: "us-east-1"},
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
			Metadata: OnImagePushMetadata{Region: "us-east-1"},
		}
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.IntegrationMetadata{
				EventBridge: &common.EventBridgeMetadata{
					Rules: map[string]common.EventBridgeRuleMetadata{
						"aws.ecr:us-east-1": {
							Source:      Source,
							DetailTypes: []string{DetailTypeECRImageAction},
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
		stored, ok := metadata.Get().(OnImagePushMetadata)
		require.True(t, ok)
		assert.NotEmpty(t, stored.SubscriptionID)
	})
}

func Test__OnImagePush__OnIntegrationMessage(t *testing.T) {
	trigger := &OnImagePush{}

	t.Run("repository mismatch -> no event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnImagePushMetadata{
					Repository: &Repository{RepositoryName: "backend"},
				},
			},
			Message: common.EventBridgeEvent{
				Detail: map[string]any{"repository-name": "frontend"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("repository match -> emits event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnImagePushMetadata{
					Repository: &Repository{RepositoryName: "backend"},
				},
			},
			Message: common.EventBridgeEvent{
				Detail: map[string]any{"repository-name": "backend"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "aws.ecr.image.push", eventContext.Payloads[0].Type)
	})
}
