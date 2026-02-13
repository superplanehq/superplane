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

	t.Run("missing subscription arn -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
			},
		})
		require.ErrorContains(t, err, "subscription ARN is required")
	})

	t.Run("invalid subscription arn format -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":          "us-east-1",
				"subscriptionArn": "invalid-subscription-arn",
			},
		})
		require.ErrorContains(t, err, "invalid subscription ARN format")
	})

	t.Run("valid china partition subscription arn -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":          "cn-northwest-1",
				"subscriptionArn": "arn:aws-cn:sns:cn-northwest-1:123456789012:orders-events:sub-123",
			},
		})
		require.NoError(t, err)
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
