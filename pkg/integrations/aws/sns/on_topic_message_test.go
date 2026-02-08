package sns

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// Test__OnTopicMessage__Setup validates trigger setup behavior.
func Test__OnTopicMessage__Setup(t *testing.T) {
	trigger := &OnTopicMessage{}

	t.Run("missing rule -> schedules rule provisioning", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<GetTopicAttributesResponse>
						  <GetTopicAttributesResult>
							<Attributes>
							  <entry><key>DisplayName</key><value>Orders Events</value></entry>
							</Attributes>
						  </GetTopicAttributesResult>
						</GetTopicAttributesResponse>
					`)),
				},
			},
		}

		metadataContext := &contexts.MetadataContext{}
		requestContext := &contexts.RequestContext{}
		integration := testIntegrationContext()
		integration.Metadata = common.IntegrationMetadata{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			HTTP:        httpContext,
			Metadata:    metadataContext,
			Requests:    requestContext,
			Integration: integration,
		})

		require.NoError(t, err)
		require.Len(t, integration.ActionRequests, 1)
		assert.Equal(t, "provisionRule", integration.ActionRequests[0].ActionName)
		assert.Equal(t, "checkRuleAvailability", requestContext.Action)
	})

	t.Run("rule exists -> subscribes immediately", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<GetTopicAttributesResponse>
						  <GetTopicAttributesResult>
							<Attributes>
							  <entry><key>DisplayName</key><value>Orders Events</value></entry>
							</Attributes>
						  </GetTopicAttributesResult>
						</GetTopicAttributesResponse>
					`)),
				},
			},
		}

		metadataContext := &contexts.MetadataContext{}
		integration := testIntegrationContext()
		integration.Metadata = common.IntegrationMetadata{
			EventBridge: &common.EventBridgeMetadata{
				Rules: map[string]common.EventBridgeRuleMetadata{
					Source: {
						Source:      Source,
						Region:      "us-east-1",
						DetailTypes: []string{DetailTypeTopicNotification},
					},
				},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			HTTP:        httpContext,
			Metadata:    metadataContext,
			Requests:    &contexts.RequestContext{},
			Integration: integration,
		})

		require.NoError(t, err)
		require.Len(t, integration.Subscriptions, 1)

		metadata, ok := metadataContext.Metadata.(OnTopicMessageMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", metadata.Region)
		assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:orders-events", metadata.TopicArn)
		assert.NotEmpty(t, metadata.SubscriptionID)
	})
}

// Test__OnTopicMessage__HandleAction validates trigger action behavior.
func Test__OnTopicMessage__HandleAction(t *testing.T) {
	trigger := &OnTopicMessage{}

	t.Run("rule exists -> subscribes and stores metadata", func(t *testing.T) {
		metadataContext := &contexts.MetadataContext{
			Metadata: OnTopicMessageMetadata{
				Region:   "us-east-1",
				TopicArn: "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
		}

		integration := testIntegrationContext()
		integration.Metadata = common.IntegrationMetadata{
			EventBridge: &common.EventBridgeMetadata{
				Rules: map[string]common.EventBridgeRuleMetadata{
					Source: {
						Source:      Source,
						Region:      "us-east-1",
						DetailTypes: []string{DetailTypeTopicNotification},
					},
				},
			},
		}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name:        "checkRuleAvailability",
			Metadata:    metadataContext,
			Integration: integration,
			Requests:    &contexts.RequestContext{},
			Logger:      logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		require.Len(t, integration.Subscriptions, 1)

		metadata, ok := metadataContext.Metadata.(OnTopicMessageMetadata)
		require.True(t, ok)
		assert.NotEmpty(t, metadata.SubscriptionID)
	})
}

// Test__OnTopicMessage__OnIntegrationMessage validates event filtering behavior.
func Test__OnTopicMessage__OnIntegrationMessage(t *testing.T) {
	trigger := &OnTopicMessage{}

	t.Run("matching topic -> emits event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"detail": map[string]any{
					"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
					"message":  "hello",
				},
			},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnTopicMessageMetadata{
					TopicArn: "arn:aws:sns:us-east-1:123456789012:orders-events",
				},
			},
			Events: eventContext,
			Logger: logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		require.Len(t, eventContext.Payloads, 1)
		assert.Equal(t, "aws.sns.topic.message", eventContext.Payloads[0].Type)
	})

	t.Run("different topic -> no event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"detail": map[string]any{
					"topicArn": "arn:aws:sns:us-east-1:123456789012:different-topic",
					"message":  "hello",
				},
			},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: OnTopicMessageMetadata{
					TopicArn: "arn:aws:sns:us-east-1:123456789012:orders-events",
				},
			},
			Events: eventContext,
			Logger: logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Empty(t, eventContext.Payloads)
	})
}
