package sns

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// Test__GetTopic__Setup validates GetTopic setup behavior.
func Test__GetTopic__Setup(t *testing.T) {
	component := &GetTopic{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   " ",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("invalid region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "invalid-region",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
		})
		require.ErrorContains(t, err, "invalid AWS region")
	})

	t.Run("missing topic arn -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
			},
		})
		require.ErrorContains(t, err, "topic ARN is required")
	})

	t.Run("invalid topic arn format -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "invalid-arn",
			},
		})
		require.ErrorContains(t, err, "invalid topic ARN format")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
		})
		require.NoError(t, err)
	})

	t.Run("valid china partition topic arn -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "cn-north-1",
				"topicArn": "arn:aws-cn:sns:cn-north-1:123456789012:orders-events",
			},
		})
		require.NoError(t, err)
	})
}

// Test__SNSComponents__Setup validates setup requirements across SNS components.
func Test__SNSComponents__Setup(t *testing.T) {
	t.Run("get subscription missing subscription arn -> error", func(t *testing.T) {
		component := &GetSubscription{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
			},
		})
		require.ErrorContains(t, err, "subscription ARN is required")
	})

	t.Run("get subscription invalid subscription arn format -> error", func(t *testing.T) {
		component := &GetSubscription{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":          "us-east-1",
				"subscriptionArn": "invalid-subscription-arn",
			},
		})
		require.ErrorContains(t, err, "invalid subscription ARN format")
	})

	t.Run("get subscription valid china partition subscription arn -> success", func(t *testing.T) {
		component := &GetSubscription{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":          "cn-northwest-1",
				"subscriptionArn": "arn:aws-cn:sns:cn-northwest-1:123456789012:orders-events:sub-123",
			},
		})
		require.NoError(t, err)
	})

	t.Run("create topic missing name -> error", func(t *testing.T) {
		component := &CreateTopic{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
			},
		})
		require.ErrorContains(t, err, "topic name is required")
	})

	t.Run("delete topic missing topic arn -> error", func(t *testing.T) {
		component := &DeleteTopic{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
			},
		})
		require.ErrorContains(t, err, "topic ARN is required")
	})

	t.Run("publish message missing message -> error", func(t *testing.T) {
		component := &PublishMessage{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
		})
		require.ErrorContains(t, err, "message is required")
	})

	t.Run("subscribe invalid protocol -> error", func(t *testing.T) {
		component := &Subscribe{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
				"protocol": "invalid",
				"endpoint": "https://example.com/hook",
			},
		})
		require.ErrorContains(t, err, "unsupported protocol")
	})

	t.Run("unsubscribe missing subscription arn -> error", func(t *testing.T) {
		component := &Unsubscribe{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
			},
		})
		require.ErrorContains(t, err, "subscription ARN is required")
	})
}

// Test__GetTopic__Execute validates GetTopic execution behavior.
func Test__GetTopic__Execute(t *testing.T) {
	component := &GetTopic{}

	t.Run("valid request -> emits topic payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<GetTopicAttributesResponse>
						  <GetTopicAttributesResult>
							<Attributes>
							  <entry><key>DisplayName</key><value>Orders Events</value></entry>
							  <entry><key>Owner</key><value>123456789012</value></entry>
							</Attributes>
						  </GetTopicAttributesResult>
						</GetTopicAttributesResponse>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    testIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)["data"]
		topic, ok := payload.(*Topic)
		require.True(t, ok)
		assert.Equal(t, "orders-events", topic.Name)
		assert.Equal(t, "Orders Events", topic.DisplayName)
		assert.Equal(t, "https://sns.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())
	})
}

// Test__GetSubscription__Execute validates GetSubscription execution behavior.
func Test__GetSubscription__Execute(t *testing.T) {
	component := &GetSubscription{}

	t.Run("valid request -> emits subscription payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<GetSubscriptionAttributesResponse>
						  <GetSubscriptionAttributesResult>
							<Attributes>
							  <entry><key>TopicArn</key><value>arn:aws:sns:us-east-1:123456789012:orders-events</value></entry>
							  <entry><key>Protocol</key><value>https</value></entry>
							  <entry><key>Endpoint</key><value>https://example.com/hook</value></entry>
							  <entry><key>RawMessageDelivery</key><value>true</value></entry>
							</Attributes>
						  </GetSubscriptionAttributesResult>
						</GetSubscriptionAttributesResponse>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":          "us-east-1",
				"subscriptionArn": "arn:aws:sns:us-east-1:123456789012:orders-events:sub",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    testIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)["data"]
		subscription, ok := payload.(*Subscription)
		require.True(t, ok)
		assert.Equal(t, "https", subscription.Protocol)
		assert.Equal(t, "https://example.com/hook", subscription.Endpoint)
		assert.True(t, subscription.RawMessageDelivery)
	})
}

// Test__CreateTopic__Execute validates CreateTopic execution behavior.
func Test__CreateTopic__Execute(t *testing.T) {
	component := &CreateTopic{}

	t.Run("valid request -> emits created topic payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<CreateTopicResponse>
						  <CreateTopicResult>
							<TopicArn>arn:aws:sns:us-east-1:123456789012:orders-events</TopicArn>
						  </CreateTopicResult>
						</CreateTopicResponse>
					`)),
				},
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

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"name":   "orders-events",
				"attributes": map[string]any{
					"DisplayName": "Orders Events",
				},
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    testIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)["data"]
		topic, ok := payload.(*Topic)
		require.True(t, ok)
		assert.Equal(t, "orders-events", topic.Name)
		assert.Equal(t, "Orders Events", topic.DisplayName)
	})
}

// Test__DeleteTopic__Execute validates DeleteTopic execution behavior.
func Test__DeleteTopic__Execute(t *testing.T) {
	component := &DeleteTopic{}

	t.Run("valid request -> emits deleted payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<DeleteTopicResponse></DeleteTopicResponse>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    testIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:orders-events", payload["topicArn"])
		assert.Equal(t, true, payload["deleted"])
	})
}

// Test__PublishMessage__Execute validates PublishMessage execution behavior.
func Test__PublishMessage__Execute(t *testing.T) {
	component := &PublishMessage{}

	t.Run("valid request -> emits publish payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<PublishResponse>
						  <PublishResult>
							<MessageId>msg-123</MessageId>
						  </PublishResult>
						</PublishResponse>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
				"message":  "hello world",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    testIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)["data"]
		result, ok := payload.(*PublishResult)
		require.True(t, ok)
		assert.Equal(t, "msg-123", result.MessageID)
	})
}

// Test__Subscribe__Execute validates Subscribe execution behavior.
func Test__Subscribe__Execute(t *testing.T) {
	component := &Subscribe{}

	t.Run("valid request -> emits subscription payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<SubscribeResponse>
						  <SubscribeResult>
							<SubscriptionArn>arn:aws:sns:us-east-1:123456789012:orders-events:sub</SubscriptionArn>
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
							  <entry><key>Endpoint</key><value>https://example.com/hook</value></entry>
							</Attributes>
						  </GetSubscriptionAttributesResult>
						</GetSubscriptionAttributesResponse>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
				"protocol": "https",
				"endpoint": "https://example.com/hook",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    testIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)["data"]
		subscription, ok := payload.(*Subscription)
		require.True(t, ok)
		assert.Equal(t, "https://example.com/hook", subscription.Endpoint)
	})
}

// Test__Unsubscribe__Execute validates Unsubscribe execution behavior.
func Test__Unsubscribe__Execute(t *testing.T) {
	component := &Unsubscribe{}

	t.Run("valid request -> emits unsubscribed payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<UnsubscribeResponse></UnsubscribeResponse>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":          "us-east-1",
				"subscriptionArn": "arn:aws:sns:us-east-1:123456789012:orders-events:sub",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    testIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:orders-events:sub", payload["subscriptionArn"])
		assert.Equal(t, true, payload["deleted"])
	})
}

// testIntegrationContext returns an integration context with AWS session secrets.
func testIntegrationContext() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Secrets: map[string]core.IntegrationSecret{
			"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
			"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
			"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
		},
	}
}
