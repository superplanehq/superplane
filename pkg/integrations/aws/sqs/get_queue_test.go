package sqs

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

func Test__GetQueue__Setup(t *testing.T) {
	component := &GetQueue{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": " ",
				"queue":  "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue",
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing queue -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"queue":  " ",
			},
		})

		require.ErrorContains(t, err, "queue is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"queue":  "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue",
			},
		})

		require.NoError(t, err)
	})
}

func Test__GetQueue__Execute(t *testing.T) {
	component := &GetQueue{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"queue":  "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue",
			},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("valid request -> emits queue attributes", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<GetQueueAttributesResponse>
						  <GetQueueAttributesResult>
						    <Attribute>
						      <Name>VisibilityTimeout</Name>
						      <Value>30</Value>
						    </Attribute>
						    <Attribute>
						      <Name>MessageRetentionPeriod</Name>
						      <Value>1209600</Value>
						    </Attribute>
						  </GetQueueAttributesResult>
						</GetQueueAttributesResponse>
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"queue":  "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.True(t, execState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, "aws.sqs.queue", execState.Type)

		require.Len(t, execState.Payloads, 1)
		payloadWrapper, ok := execState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := payloadWrapper["data"].(map[string]any)
		require.True(t, ok)

		assert.Equal(t, "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue", data["queueUrl"])
		attributes, ok := data["attributes"].(map[string]string)
		require.True(t, ok)
		assert.Equal(t, "30", attributes["VisibilityTimeout"])
		assert.Equal(t, "1209600", attributes["MessageRetentionPeriod"])

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue", httpContext.Requests[0].URL.String())
	})
}
