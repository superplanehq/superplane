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

func Test__GetSubscription__Setup(t *testing.T) {
	component := &GetSubscription{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"subscriptionArn": "arn:aws:sns:us-east-1:123456789012:orders-events:sub-123",
			},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing topic arn -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
			},
		})
		require.ErrorContains(t, err, "topic ARN is required")
	})

	t.Run("missing subscription arn -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
		})
		require.ErrorContains(t, err, "subscription ARN is required")
	})
}

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
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
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
