package prometheus

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

func Test__CreateWorkspace__Setup(t *testing.T) {
	component := &CreateWorkspace{}

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
				"alias":  "metrics",
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alias":  "metrics",
			},
		})

		require.NoError(t, err)
	})
}

func Test__CreateWorkspace__Execute(t *testing.T) {
	component := &CreateWorkspace{}

	t.Run("valid request -> emits workspace", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body: io.NopCloser(strings.NewReader(`{
						"arn": "arn:aws:aps:us-east-1:123456789012:workspace/ws-abc123",
						"status": {"statusCode": "CREATING"},
						"tags": {"env": "prod"},
						"workspaceId": "ws-abc123"
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":      "us-east-1",
				"alias":       " metrics ",
				"clientToken": "token-1",
				"tags": []any{
					map[string]any{"key": "env", "value": "prod"},
				},
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.True(t, execState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, "aws.prometheus.workspace", execState.Type)

		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		workspace, ok := payload["workspace"].(*CreateWorkspaceResponse)
		require.True(t, ok)
		assert.Equal(t, "ws-abc123", workspace.WorkspaceID)
		assert.Equal(t, "metrics", workspace.Alias)
		assert.Equal(t, "CREATING", workspace.Status.StatusCode)

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "https://aps.us-east-1.amazonaws.com/workspaces", request.URL.String())

		requestBody, err := io.ReadAll(request.Body)
		require.NoError(t, err)

		sentPayload := map[string]any{}
		err = json.Unmarshal(requestBody, &sentPayload)
		require.NoError(t, err)
		assert.Equal(t, "metrics", sentPayload["alias"])
		assert.Equal(t, "token-1", sentPayload["clientToken"])
		assert.Equal(t, map[string]any{"env": "prod"}, sentPayload["tags"])
	})
}
