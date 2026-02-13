package sns

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type testNodeWebhookContext struct {
	url string
}

func (t *testNodeWebhookContext) Setup() (string, error) {
	return t.url, nil
}

func (t *testNodeWebhookContext) GetSecret() ([]byte, error) {
	return []byte("secret"), nil
}

func (t *testNodeWebhookContext) ResetSecret() ([]byte, []byte, error) {
	return []byte("secret"), []byte("secret"), nil
}

func (t *testNodeWebhookContext) GetBaseURL() string {
	return "http://localhost:8000"
}

func Test__OnTopicMessage__Setup(t *testing.T) {
	trigger := &OnTopicMessage{}

	t.Run("valid configuration -> subscribes webhook endpoint", func(t *testing.T) {
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
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<SubscribeResponse>
						  <SubscribeResult>
							<SubscriptionArn>arn:aws:sns:us-east-1:123456789012:orders-events:sub-123</SubscriptionArn>
						  </SubscribeResult>
						</SubscribeResponse>
					`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<GetSubscriptionAttributesResponse>
						  <GetSubscriptionAttributesResult>
							<Attributes>
							  <entry><key>TopicArn</key><value>arn:aws:sns:us-east-1:123456789012:orders-events</value></entry>
							  <entry><key>Protocol</key><value>https</value></entry>
							  <entry><key>Endpoint</key><value>https://example.com/webhooks/sns-node</value></entry>
							</Attributes>
						  </GetSubscriptionAttributesResult>
						</GetSubscriptionAttributesResponse>
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
			Webhook: &testNodeWebhookContext{
				url: "https://example.com/webhooks/sns-node",
			},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 3)

		subscribeForm := requestFormValues(t, httpContext.Requests[1])
		assert.Equal(t, "Subscribe", subscribeForm.Get("Action"))
		assert.Equal(t, "https", subscribeForm.Get("Protocol"))
		assert.Equal(t, "https://example.com/webhooks/sns-node", subscribeForm.Get("Endpoint"))
		assert.Equal(t, "true", subscribeForm.Get("ReturnSubscriptionArn"))

		metadata, ok := metadataContext.Metadata.(OnTopicMessageMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", metadata.Region)
		assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:orders-events", metadata.TopicArn)
		assert.Equal(t, "https://example.com/webhooks/sns-node", metadata.WebhookURL)
		assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:orders-events:sub-123", metadata.SubscriptionArn)
	})

	t.Run("existing matching metadata -> no subscribe call", func(t *testing.T) {
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

		metadataContext := &contexts.MetadataContext{
			Metadata: OnTopicMessageMetadata{
				Region:          "us-east-1",
				TopicArn:        "arn:aws:sns:us-east-1:123456789012:orders-events",
				WebhookURL:      "https://example.com/webhooks/sns-node",
				SubscriptionArn: "arn:aws:sns:us-east-1:123456789012:orders-events:sub-123",
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			HTTP:        httpContext,
			Metadata:    metadataContext,
			Integration: testIntegrationContext(),
			Webhook: &testNodeWebhookContext{
				url: "https://example.com/webhooks/sns-node",
			},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
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
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		require.Len(t, eventContext.Payloads, 1)
		assert.Equal(t, "aws.sns.topic.message", eventContext.Payloads[0].Type)

		payload, ok := eventContext.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:orders-events", payload["topicArn"])
		assert.Equal(t, "us-east-1", payload["region"])
		assert.Equal(t, "123456789012", payload["account"])
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
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
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

func Test__OnTopicMessage__Cleanup(t *testing.T) {
	trigger := &OnTopicMessage{}

	t.Run("stored subscription -> unsubscribe request", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`<UnsubscribeResponse/>`)),
				},
			},
		}

		err := trigger.Cleanup(core.TriggerContext{
			HTTP: httpContext,
			Metadata: &contexts.MetadataContext{
				Metadata: OnTopicMessageMetadata{
					Region:          "us-east-1",
					SubscriptionArn: "arn:aws:sns:us-east-1:123456789012:orders-events:sub-123",
				},
			},
			Integration: testIntegrationContext(),
			Logger:      logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		form := requestFormValues(t, httpContext.Requests[0])
		assert.Equal(t, "Unsubscribe", form.Get("Action"))
		assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:orders-events:sub-123", form.Get("SubscriptionArn"))
	})
}

func requestFormValues(t *testing.T, request *http.Request) url.Values {
	t.Helper()

	bodyBytes, err := io.ReadAll(request.Body)
	require.NoError(t, err)

	require.NoError(t, request.Body.Close())
	request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	values, err := url.ParseQuery(string(bodyBytes))
	require.NoError(t, err)
	return values
}
