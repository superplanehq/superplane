package prometheus

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

func Test__Query__Setup(t *testing.T) {
	component := &Query{}

	t.Run("missing query -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"query":     " ",
			},
		})

		require.ErrorContains(t, err, "query is required")
	})

	t.Run("valid configuration -> stores workspace alias in metadata", func(t *testing.T) {
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
				"query":     "up",
			},
			HTTP:        httpContext,
			Integration: validIntegrationContext(),
			Metadata:    metadata,
		})

		require.NoError(t, err)
		stored, ok := metadata.Get().(WorkspaceNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "metrics", stored.WorkspaceAlias)
	})
}

func Test__Query__Execute(t *testing.T) {
	component := &Query{}

	t.Run("valid request -> emits query result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"status": "success",
						"data": {
							"resultType": "vector",
							"result": [
								{"metric": {"job": "prometheus"}, "value": [1717846800, "1"]}
							]
						}
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"query":     " up ",
				"time":      "2026-06-08T09:00:00Z",
				"timeout":   "30s",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		assert.Equal(t, "aws.prometheus.query", execState.Type)

		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "vector", payload["resultType"])
		result, ok := payload["result"].([]any)
		require.True(t, ok)
		assert.Len(t, result, 1)

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "https://aps-workspaces.us-east-1.amazonaws.com/workspaces/ws-abc123/api/v1/query", request.URL.Scheme+"://"+request.URL.Host+request.URL.Path)
		assert.Equal(t, "up", request.URL.Query().Get("query"))
		assert.Equal(t, "2026-06-08T09:00:00Z", request.URL.Query().Get("time"))
		assert.Equal(t, "30s", request.URL.Query().Get("timeout"))
	})

	t.Run("Prometheus API status error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"status": "error",
						"errorType": "bad_data",
						"error": "invalid query"
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"query":     "invalid{",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.ErrorContains(t, err, "failed to execute PromQL query")
		require.ErrorContains(t, err, "prometheus API error (bad_data): invalid query")
		assert.False(t, execState.Finished)
		assert.Empty(t, execState.Payloads)
	})
}
