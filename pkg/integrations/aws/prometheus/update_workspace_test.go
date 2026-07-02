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

func Test__UpdateWorkspace__Setup(t *testing.T) {
	component := &UpdateWorkspace{}

	t.Run("missing alias -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"alias":     " ",
			},
		})

		require.ErrorContains(t, err, "alias is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"workspace": {
							"alias": "metrics",
							"arn": "arn:aws:aps:us-east-1:123456789012:workspace/ws-abc123",
							"status": {"statusCode": "ACTIVE"},
							"workspaceId": "ws-abc123"
						}
					}`)),
				},
			},
		}

		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"alias":     "metrics-v2",
			},
			HTTP:        httpContext,
			Integration: validIntegrationContext(),
			Metadata:    metadata,
		})

		require.NoError(t, err)
		stored, ok := metadata.Get().(WorkspaceNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Equal(t, "ws-abc123", stored.WorkspaceID)
		assert.Equal(t, "metrics", stored.WorkspaceAlias)
	})
}

func Test__UpdateWorkspace__Execute(t *testing.T) {
	component := &UpdateWorkspace{}

	t.Run("valid request -> emits update result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":      "us-east-1",
				"workspace":   "ws-abc123",
				"alias":       " metrics ",
				"clientToken": "token-1",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		assert.Equal(t, "aws.prometheus.workspace.updated", execState.Type)

		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "ws-abc123", payload["workspaceId"])
		assert.Equal(t, "metrics", payload["alias"])
		assert.Equal(t, true, payload["updated"])

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "https://aps.us-east-1.amazonaws.com/workspaces/ws-abc123/alias", request.URL.String())

		requestBody, err := io.ReadAll(request.Body)
		require.NoError(t, err)

		sentPayload := map[string]any{}
		err = json.Unmarshal(requestBody, &sentPayload)
		require.NoError(t, err)
		assert.Equal(t, "metrics", sentPayload["alias"])
		assert.Equal(t, "token-1", sentPayload["clientToken"])
	})
}
