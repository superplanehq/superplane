package ecs

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ExecuteCommand__Setup(t *testing.T) {
	component := &ExecuteCommand{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing command -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"cluster":   "demo",
				"task":      "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
				"container": "api",
			},
		})

		require.ErrorContains(t, err, "command is required")
	})

	t.Run("missing container -> no error", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"cluster": "demo",
				"task":    "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
				"command": "ls -la",
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
		require.NotNil(t, metadataCtx.Metadata)
	})
}

func Test__ExecuteCommand__Execute(t *testing.T) {
	component := &ExecuteCommand{}

	t.Run("valid request -> emits command session", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"clusterArn": "arn:aws:ecs:us-east-1:123456789012:cluster/demo",
							"containerArn": "arn:aws:ecs:us-east-1:123456789012:container/demo/abc/def",
							"containerName": "api",
							"interactive": false,
							"session": {
								"sessionId": "session-123",
								"streamUrl": "wss://example.com/stream",
								"tokenValue": "token-xyz"
							},
							"taskArn": "arn:aws:ecs:us-east-1:123456789012:task/demo/abc"
						}
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":      "us-east-1",
				"cluster":     "demo",
				"task":        "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
				"container":   "api",
				"command":     "ls -la",
				"interactive": false,
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)

		command, ok := payload["command"].(ExecuteCommandResponse)
		require.True(t, ok)
		assert.Equal(t, "api", command.ContainerName)
		assert.Equal(t, "session-123", command.Session.SessionID)

		require.Len(t, httpContext.Requests, 1)
		requestBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		payloadSent := map[string]any{}
		err = json.Unmarshal(requestBody, &payloadSent)
		require.NoError(t, err)
		assert.Equal(t, "demo", payloadSent["cluster"])
		assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:task/demo/abc", payloadSent["task"])
		assert.Equal(t, "api", payloadSent["container"])
		assert.Equal(t, "ls -la", payloadSent["command"])
		assert.Equal(t, false, payloadSent["interactive"])
	})

	t.Run("request without container -> omits container from payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"clusterArn": "arn:aws:ecs:us-east-1:123456789012:cluster/demo",
							"containerArn": "arn:aws:ecs:us-east-1:123456789012:container/demo/abc/def",
							"containerName": "api",
							"interactive": false,
							"session": {
								"sessionId": "session-123",
								"streamUrl": "wss://example.com/stream",
								"tokenValue": "token-xyz"
							},
							"taskArn": "arn:aws:ecs:us-east-1:123456789012:task/demo/abc"
						}
					`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":      "us-east-1",
				"cluster":     "demo",
				"task":        "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
				"command":     "ls -la",
				"interactive": false,
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		requestBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		payloadSent := map[string]any{}
		err = json.Unmarshal(requestBody, &payloadSent)
		require.NoError(t, err)
		assert.Equal(t, "demo", payloadSent["cluster"])
		assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:task/demo/abc", payloadSent["task"])
		assert.Equal(t, "ls -la", payloadSent["command"])
		assert.Equal(t, false, payloadSent["interactive"])
		assert.NotContains(t, payloadSent, "container")
	})
}
