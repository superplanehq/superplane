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
		topic, ok := payload.(*Topic)
		require.True(t, ok)
		assert.Equal(t, "orders-events", topic.Name)
		assert.Equal(t, "Orders Events", topic.DisplayName)
		assert.Equal(t, "https://sns.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())
	})
}
