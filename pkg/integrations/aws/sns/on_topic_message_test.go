package sns

import (
	"io"
	"net/http"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnTopicMessage__Setup(t *testing.T) {
	trigger := &OnTopicMessage{}

	t.Run("valid configuration -> requests webhook endpoint", func(t *testing.T) {
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

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			HTTP:        httpContext,
			Metadata:    metadataContext,
			Integration: integration,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		require.Len(t, integration.WebhookRequests, 1)

		metadata, ok := metadataContext.Metadata.(OnTopicMessageMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", metadata.Region)
		assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:orders-events", metadata.TopicArn)

		webhookConfig, ok := integration.WebhookRequests[0].(common.WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", webhookConfig.Region)
		assert.Equal(t, common.WebhookTypeSNS, webhookConfig.Type)
		require.NotNil(t, webhookConfig.SNS)
		assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:orders-events", webhookConfig.SNS.TopicArn)
	})

	t.Run("existing matching metadata -> no subscribe call", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		metadataContext := &contexts.MetadataContext{
			Metadata: OnTopicMessageMetadata{
				Region:   "us-east-1",
				TopicArn: "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
		}

		integration := testIntegrationContext()
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			HTTP:        httpContext,
			Metadata:    metadataContext,
			Integration: integration,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 0)
		require.Len(t, integration.WebhookRequests, 0)
	})
}

func Test__OnTopicMessage__HandleWebhook(t *testing.T) {
	trigger := &OnTopicMessage{}

	t.Run("notification for configured topic -> emits event", func(t *testing.T) {
		eventContext := &contexts.EventContext{}

		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{
				"Type": "Notification",
				"MessageId": "msg-123",
				"TopicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
				"Subject": "order.created",
				"Message": "{\"orderId\":\"ord_123\"}",
				"Timestamp": "2026-01-10T10:00:00Z"
			}`),
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			Events: eventContext,
			HTTP:   &contexts.HTTPContext{},
			Logger: log.NewEntry(log.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		require.Len(t, eventContext.Payloads, 1)
		assert.Equal(t, "aws.sns.topic.message", eventContext.Payloads[0].Type)

		payload, ok := eventContext.Payloads[0].Data.(SubscriptionMessage)
		require.True(t, ok)
		assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:orders-events", payload.TopicArn)
		assert.Equal(t, "order.created", payload.Subject)
		assert.Equal(t, "{\"orderId\":\"ord_123\"}", payload.Message)
		assert.Equal(t, "2026-01-10T10:00:00Z", payload.Timestamp)
	})

	t.Run("subscription confirmation for different topic -> ignored", func(t *testing.T) {
		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{
				"Type": "SubscriptionConfirmation",
				"TopicArn": "arn:aws:sns:us-east-1:123456789012:different-topic",
				"SubscribeURL": "https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription"
			}`),
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			Events: &contexts.EventContext{},
			HTTP:   &contexts.HTTPContext{},
			Logger: log.NewEntry(log.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("confirmation for configured topic -> confirms subscription", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(``)),
			}},
		}

		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{
				"Type": "SubscriptionConfirmation",
				"TopicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
				"SubscribeURL": "https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription"
			}`),
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			HTTP:   httpCtx,
			Events: &contexts.EventContext{},
			Logger: log.NewEntry(log.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription", httpCtx.Requests[0].URL.String())
	})

	t.Run("unsupported message type -> bad request", func(t *testing.T) {
		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{
				"Type": "UnknownType",
				"TopicArn": "arn:aws:sns:us-east-1:123456789012:orders-events"
			}`),
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			Events: &contexts.EventContext{},
		})

		require.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, status)
	})
}
