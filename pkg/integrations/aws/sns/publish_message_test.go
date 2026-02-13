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

func Test__PublishMessage__Setup(t *testing.T) {
	component := &PublishMessage{}

	t.Run("missing message -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
		})
		require.ErrorContains(t, err, "message is required")
	})
}

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
		result, ok := payload.(*PublishResult)
		require.True(t, ok)
		assert.Equal(t, "msg-123", result.MessageID)
	})
}
